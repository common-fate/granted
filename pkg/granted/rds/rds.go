package rds

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/session-manager-plugin/src/datachannel"
	"github.com/aws/session-manager-plugin/src/sessionmanagerplugin/session"
	"github.com/aws/session-manager-plugin/src/sessionmanagerplugin/session/portsession"
	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/grab"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/granted/pkg/hook/accessrequesthook"
	"github.com/common-fate/sdk/config"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	"github.com/common-fate/sdk/gen/commonfate/access/v1alpha1/accessv1alpha1connect"
	"github.com/common-fate/sdk/handshake"
	"github.com/common-fate/sdk/service/access"
	"github.com/common-fate/sdk/service/access/grants"
	"github.com/common-fate/xid"
	"github.com/fatih/color"
	"github.com/hashicorp/yamux"
	"github.com/mattn/go-runewidth"
	sethRetry "github.com/sethvargo/go-retry"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"
)

var Command = cli.Command{
	Name:        "rds",
	Usage:       "Granted RDS plugin",
	Description: "Granted RDS plugin",
	Subcommands: []*cli.Command{&proxyCommand},
}

var proxyCommand = cli.Command{
	Name:  "proxy",
	Usage: "The Proxy plugin is used in conjunction with a Commnon Fate deployment to request temporary access to an AWS RDS Database",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "target", Aliases: []string{"database"}},
		&cli.StringFlag{Name: "role", Aliases: []string{"user"}},
		&cli.IntFlag{Name: "port", Usage: "The local port to forward the mysql database connection to"},
		&cli.StringFlag{Name: "reason", Usage: "Provide a reason for requesting access to the role"},
		&cli.BoolFlag{Name: "confirm", Aliases: []string{"y"}, Usage: "Skip confirmation prompts for access requests"},
		&cli.BoolFlag{Name: "wait", Value: true, Usage: "Wait for the access request to be approved."},
		&cli.BoolFlag{Name: "no-cache", Usage: "Disables caching of session credentials and forces a refresh", EnvVars: []string{"GRANTED_NO_CACHE"}},
		&cli.DurationFlag{Name: "duration", Aliases: []string{"d"}, Usage: "The duration for your access request"},
		&cli.StringFlag{Name: "mode", Hidden: true, Usage: "What mode to run the proxy command in, [remote,local], local is used in development to connect to a local instance of the proxy server rather than remote via SSM", Value: "remote"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		cfg, err := config.LoadDefault(ctx)
		if err != nil {
			return err
		}

		err = cfg.Initialize(ctx, config.InitializeOpts{})
		if err != nil {
			return err
		}

		ensuredAccess, err := ensureAccess(ctx, cfg, ensureAccessInput{
			Database: c.String("target"),
			User:     c.String("role"),
			Duration: c.Duration("duration"),
			Reason:   c.String("reason"),
			Confirm:  c.Bool("confirm"),
			Wait:     c.Bool("wait"),
		})
		if err != nil {
			return err
		}

		requestURL, err := generateRequestURL(ctx, ensuredAccess.Grant)
		if err != nil {
			return err
		}

		serverPort, localPort, err := ports(c)
		if err != nil {
			return err
		}

		clio.Debugw("prepared ports for access", "serverPort", serverPort, "localPort", localPort)

		err = waitForSSMConnectionToProxyServer(c, ensuredAccess, serverPort, localPort)
		if err != nil {
			return err
		}

		underlyingProxyServerConn, yamuxStreamConnection, err := initiateSessionConnection(cfg, initiateSessionConnectionInput{
			EnsuredAccess: ensuredAccess,
			RequestURL:    requestURL,
			LocalPort:     localPort,
		})
		if err != nil {
			return err
		}
		defer underlyingProxyServerConn.Close()
		defer yamuxStreamConnection.Close()

		connectionString, cliString, clientConnectionPort, err := clientConnectionParameters(c, ensuredAccess)
		if err != nil {
			return err
		}

		printConnectionParameters(connectionString, cliString, clientConnectionPort, ensuredAccess.GrantOutput.RdsDatabase.Engine)

		return listenAndProxy(ctx, yamuxStreamConnection, clientConnectionPort, requestURL)
	},
}

// listenAndProxy will listen for new client connections and start a stream over the established proxy server session.
// if the proxy server terminates the session, like when a grant expires, this listener will detect it and terminate the CLI commmand with an error explaining what happened
func listenAndProxy(ctx context.Context, yamuxStreamConnection *yamux.Session, clientConnectionPort string, requestURL string) error {
	ln, err := net.Listen("tcp", "localhost:"+clientConnectionPort)
	if err != nil {
		return fmt.Errorf("failed to start listening for connections on port: %s. %w", clientConnectionPort, err)
	}
	defer ln.Close()

	type result struct {
		conn net.Conn
		err  error
	}
	resultChan := make(chan result, 100)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := ln.Accept()
				result := result{
					err: err,
				}
				if err == nil {
					result.conn = conn
				}
				resultChan <- result
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-yamuxStreamConnection.CloseChan():
			return clierr.New("The connection to the proxy server has ended", clierr.Infof("Your grant may have expired, you can check the status here: %s and retry connecting", requestURL))
		case result := <-resultChan:
			if result.err != nil {
				return fmt.Errorf("failed to accept connection: %w", err)
			}
			if yamuxStreamConnection.IsClosed() {
				return clierr.New("failed to accept connection for database client because the proxy server connection has ended", clierr.Infof("Your grant may have expired, you can check the status here: %s and retry connecting", requestURL))
			}
			go func(databaseClientConn net.Conn) {
				// A stream is opened for this connection, streams are used just like a net.Conn and can read and write data
				// A stream can only be opened while the grant is still valid, and each new connection will validate the database parameters username and database
				sessionConn, err := yamuxStreamConnection.OpenStream()
				if err != nil {
					clio.Error("Failed to establish a new connection to the remote database via the proxy server.")
					clio.Error(err)
					clio.Infof("Your grant may have expired, you can check the status here: %s", requestURL)
					return
				}

				clio.Infof("Connection accepted for session [%v]", sessionConn.StreamID())

				// If a stream successfully connects, that means that a connection to the target database is now open
				// at this point the connection traffic is handed off and the connection is effectively directly from the database client and the target database
				// with queries being intercepted and logged to the audit trail in Common Fate
				// if the grant becomes incative at any time the connection is terminated immediately
				go func() {
					defer sessionConn.Close()
					_, err := io.Copy(sessionConn, databaseClientConn)
					if err != nil {
						clio.Debugw("error writing data from client to server usually this is just because the database proxy session ended.", "streamId", sessionConn.StreamID(), zap.Error(err))
					}
					clio.Infof("Connection ended for session [%v]", sessionConn.StreamID())
				}()
				go func() {
					defer sessionConn.Close()
					_, err := io.Copy(databaseClientConn, sessionConn)
					if err != nil {
						clio.Debugw("error writing data from server to client usually this is just because the database proxy session ended.", "streamId", sessionConn.StreamID(), zap.Error(err))
					}
				}()
			}(result.conn)
		}
	}
}

// promptForDatabaseAndUser lists all available database entitlements for the user and displays a table selector UI
func promptForDatabaseAndUser(ctx context.Context, accessClient accessv1alpha1connect.AccessServiceClient) (*accessv1alpha1.Entitlement, error) {
	entitlements, err := grab.AllPages(ctx, func(ctx context.Context, nextToken *string) ([]*accessv1alpha1.Entitlement, *string, error) {
		res, err := accessClient.QueryEntitlements(ctx, connect.NewRequest(&accessv1alpha1.QueryEntitlementsRequest{
			PageToken:  grab.Value(nextToken),
			TargetType: grab.Ptr("AWS::RDS::Database"),
		}))
		if err != nil {
			return nil, nil, err
		}
		return res.Msg.Entitlements, &res.Msg.NextPageToken, nil
	})
	if err != nil {
		return nil, err
	}

	// check here to avoid nil pointer errors later
	if len(entitlements) == 0 {
		return nil, errors.New("you don't have access to any RDS databases")
	}

	type Column struct {
		Title string
		Width int
	}
	cols := []Column{{Title: "Database", Width: 40}, {Title: "Role", Width: 40}}
	var s = make([]string, 0, len(cols))
	for _, col := range cols {
		style := lipgloss.NewStyle().Width(col.Width).MaxWidth(col.Width).Inline(true)
		renderedCell := style.Render(runewidth.Truncate(col.Title, col.Width, "…"))
		s = append(s, lipgloss.NewStyle().Bold(true).Padding(0).Render(renderedCell))
	}
	header := lipgloss.NewStyle().PaddingLeft(2).Render(lipgloss.JoinHorizontal(lipgloss.Left, s...))
	var options []huh.Option[*accessv1alpha1.Entitlement]

	for _, entitlement := range entitlements {
		style := lipgloss.NewStyle().Width(cols[0].Width).MaxWidth(cols[0].Width).Inline(true)
		target := lipgloss.NewStyle().Bold(true).Padding(0).Render(style.Render(runewidth.Truncate(entitlement.Target.Display(), cols[0].Width, "…")))

		style = lipgloss.NewStyle().Width(cols[1].Width).MaxWidth(cols[1].Width).Inline(true)
		role := lipgloss.NewStyle().Bold(true).Padding(0).Render(style.Render(runewidth.Truncate(entitlement.Role.Display(), cols[1].Width, "…")))

		options = append(options, huh.Option[*accessv1alpha1.Entitlement]{
			Key:   lipgloss.JoinHorizontal(lipgloss.Left, target, role),
			Value: entitlement,
		})
	}

	selector := huh.NewSelect[*accessv1alpha1.Entitlement]().
		Options(options...).
		Title("Select a database to connect to").
		Description(header).WithTheme(huh.ThemeBase())
	err = selector.Run()
	if err != nil {
		return nil, err
	}

	selectorVal := selector.GetValue()

	if selectorVal == nil {
		return nil, errors.New("no database selected")
	}

	return selectorVal.(*accessv1alpha1.Entitlement), nil
}

func durationOrDefault(duration time.Duration) *durationpb.Duration {
	var out *durationpb.Duration
	if duration != 0 {
		out = durationpb.New(duration)
	}
	return out
}

type ensureAccessInput struct {
	Database string
	User     string
	Duration time.Duration
	Reason   string
	Confirm  bool
	Wait     bool
}
type ensureAccessOutput struct {
	GrantOutput *accessv1alpha1.AWSRDSOutput
	Grant       *accessv1alpha1.Grant
}

// ensureAccess checks for an existing grant or creates a new one if it does not exist
func ensureAccess(ctx context.Context, cfg *config.Context, input ensureAccessInput) (*ensureAccessOutput, error) {

	accessRequestInput := accessrequesthook.NoEntitlementAccessInput{
		Target:    input.Database,
		Role:      input.User,
		Reason:    input.Reason,
		Duration:  durationOrDefault(input.Duration),
		Confirm:   input.Confirm,
		Wait:      input.Wait,
		StartTime: time.Now(),
	}

	if accessRequestInput.Target == "" && accessRequestInput.Role == "" {
		selectedEntitlement, err := promptForDatabaseAndUser(ctx, access.NewFromConfig(cfg))
		if err != nil {
			return nil, err
		}
		clio.Debugw("selected database and user manually", "selectedEntitlement", selectedEntitlement)
		accessRequestInput.Target = selectedEntitlement.Target.Eid.Display()
		accessRequestInput.Role = selectedEntitlement.Role.Eid.Display()
	}

	hook := accessrequesthook.Hook{}
	retry, result, _, err := hook.NoEntitlementAccess(ctx, cfg, accessRequestInput)
	if err != nil {
		return nil, err
	}

	retryDuration := time.Minute * 1
	if input.Wait {
		//if wait is specified, increase the timeout to 15 minutes.
		retryDuration = time.Minute * 15
	}

	if retry {
		// reset the start time for the timer (otherwise it shows 2s, 7s, 12s etc)
		accessRequestInput.StartTime = time.Now()

		b := sethRetry.NewConstant(5 * time.Second)
		b = sethRetry.WithMaxDuration(retryDuration, b)
		err = sethRetry.Do(ctx, b, func(ctx context.Context) (err error) {

			//also proactively check if request has been approved and attempt to activate
			result, err = hook.RetryNoEntitlementAccess(ctx, cfg, accessRequestInput)
			if err != nil {

				return sethRetry.RetryableError(err)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

	}

	if result == nil || len(result.Grants) == 0 {
		return nil, errors.New("could not load grant from Common Fate")
	}

	grant := result.Grants[0]

	grantsClient := grants.NewFromConfig(cfg)

	grantOutput, err := grantsClient.GetGrantOutput(ctx, connect.NewRequest(&accessv1alpha1.GetGrantOutputRequest{
		Id: grant.Grant.Id,
	}))
	if err != nil {
		return nil, err
	}

	clio.Debugw("found grant output", "output", grantOutput)

	rdsOutput, ok := grantOutput.Msg.Output.(*accessv1alpha1.GetGrantOutputResponse_OutputAwsRds)
	if !ok {
		return nil, errors.New("unexpected grant output, this indicates an error in the Common Fate Provisioning process, you should contect your Common Fate administrator")
	}

	return &ensureAccessOutput{
		GrantOutput: rdsOutput.OutputAwsRds,
		Grant:       grant.Grant,
	}, nil
}

// isLocalMode is used where some behaviour needs to be changed to run against a local development proxy server
func isLocalMode(c *cli.Context) bool {
	return c.String("mode") == "local"
}

// Returns the proxy port to connect to and a local port to send client connections to
// in production, an SSM portforward process is running which is used to connect to the proxy server
// and over the top of this connection, a handshake process takes place and connection multiplexing is used to handle multiple database clients
func ports(c *cli.Context) (serverPort, localPort string, err error) {
	// in local mode the SSM port forward is not used can skip using ssm and just use a local port forward instead
	if isLocalMode(c) {
		return "7070", "7070", nil
	}
	// find an unused local port to use for the ssm server
	// the user doesn't directly connect to this, they connect through our local proxy
	// which adds authentication
	ssmPortforwardLocalPort, err := GrabUnusedPort()
	if err != nil {
		return "", "", err
	}
	return "8080", ssmPortforwardLocalPort, nil
}

// waitForSSMConnectionToProxyServer starts a session with SSM and waits for the connection to be ready
func waitForSSMConnectionToProxyServer(c *cli.Context, ensuredAccess *ensureAccessOutput, serverPort, localPort string) error {
	// In local mode ssm is not used, instead, the command connects directly to the proxy service running in local dev
	// Return early because there is nothing to startup
	if isLocalMode(c) {
		return nil
	}

	ctx := c.Context

	p := &cfaws.Profile{
		Name:        ensuredAccess.Grant.Id,
		ProfileType: "AWS_SSO",
		AWSConfig: awsConfig.SharedConfig{
			SSOAccountID: ensuredAccess.GrantOutput.RdsDatabase.AccountId,
			SSORoleName:  ensuredAccess.Grant.Id,
			SSORegion:    ensuredAccess.GrantOutput.SsoRegion,
			SSOStartURL:  ensuredAccess.GrantOutput.SsoStartUrl,
		},
		Initialised: true,
	}

	creds, err := p.AssumeTerminal(ctx, cfaws.ConfigOpts{
		ShouldRetryAssuming: grab.Ptr(true),
		DisableCache:        c.Bool("no-cache"),
	})
	if err != nil {
		return err
	}

	ssmReadyForConnectionsChan := make(chan struct{})

	awscfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)))
	if err != nil {
		return err
	}
	awscfg.Region = ensuredAccess.GrantOutput.RdsDatabase.Region
	ssmClient := ssm.NewFromConfig(awscfg)

	var sessionOutput *ssm.StartSessionOutput

	documentName := "AWS-StartPortForwardingSession"
	startSessionInput := ssm.StartSessionInput{
		Target:       &ensuredAccess.GrantOutput.SsmSessionTarget,
		DocumentName: &documentName,
		Parameters: map[string][]string{
			"portNumber":      {serverPort},
			"localPortNumber": {localPort},
		},
		Reason: grab.Ptr(fmt.Sprintf("Session started for Granted RDS Proxy connection with Common Fate. GrantID: %s, AccessRequestID: %s", ensuredAccess.Grant.Id, ensuredAccess.Grant.AccessRequestId)),
	}

	sessionOutput, err = ssmClient.StartSession(ctx, &startSessionInput)
	if err != nil {
		return clierr.New("Failed to start AWS SSM port forward session",
			clierr.Error(err),
			clierr.Info("You can try re-running this command with the verbose flag to see detailed logs, 'cf --verbose aws rds proxy'"),
			clierr.Infof("In rare cases, where the database proxy has been re-deployed while your grant was active, you will need to close your request in Common Fate and request access again 'cf access close request --id=%s' This is usually indicated by an error message containing '(TargetNotConnected) when calling the StartSession'", ensuredAccess.Grant.AccessRequestId))
	}

	clientId := xid.New("gtd")
	ssmSession := session.Session{
		StreamUrl:             *sessionOutput.StreamUrl,
		SessionId:             *sessionOutput.SessionId,
		TokenValue:            *sessionOutput.TokenValue,
		IsAwsCliUpgradeNeeded: false,
		Endpoint:              "localhost:" + localPort,
		DataChannel:           &datachannel.DataChannel{},
		ClientId:              clientId,
	}

	startingProxySpinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	startingProxySpinner.Suffix = " Starting database proxy..."
	startingProxySpinner.Writer = os.Stderr
	startingProxySpinner.Start()
	defer startingProxySpinner.Stop()

	// registers the PortSession feature within the ssm library
	_ = portsession.PortSession{}

	// the SSMDebugLogger serves two purposes here
	// 1. writes ssm session logs to clio.Debug which can be viewed using the --verbose flag
	// 2. scans the output for the string "Waiting for connections..." which indicates that the SSM connection was successful
	// The notifier will notify the ssmReadyForConnectionsChan which means we can connect to the proxy to complete the initial handshake
	ssmLogger := &SSMDebugLogger{
		Writers: []io.Writer{
			&NotifyOnSubstringMatchWriter{
				Phrase:   "Waiting for connections...",
				Callback: func() { ssmReadyForConnectionsChan <- struct{}{} },
			},
			DebugWriter{},
		},
	}

	// Connect to the Proxy server using SSM
	go func() {
		// Execute starts the ssm connection
		err = ssmSession.Execute(ssmLogger)
		if err != nil {
			clio.Error("AWS SSM port forward session closed with an error")
			clio.Error(err)
			clio.Info("You can try re-running this command with the verbose flag to see detailed logs, 'cf --verbose aws rds proxy'")
			clio.Infof("In rare cases, where the database proxy has been re-deployed while your grant was active, you will need to close your request in Common Fate and request access again 'cf access close request --id=%s' This is usually indicated by an error message containing '(TargetNotConnected) when calling the StartSession'", ensuredAccess.Grant.AccessRequestId)
		}
	}()

	// waits for the ssm session to start or context to be cancelled
	select {
	case <-ssmReadyForConnectionsChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type initiateSessionConnectionInput struct {
	EnsuredAccess *ensureAccessOutput
	RequestURL    string
	LocalPort     string
}

// initiateSessionConnection starts a new tcp connection to through the SSM port forward and completes a handshake with the proxy server
// the result is a yamux session which is used to multiplex database client connections
func initiateSessionConnection(cfg *config.Context, input initiateSessionConnectionInput) (net.Conn, *yamux.Session, error) {

	// First dial the local SSM portforward, which will be running on a randomly chosen port
	// or the local proxy server instance if it's local dev mode
	// this establishes the initial connection to the Proxy server
	clio.Debugw("dialing proxy server", "host", "localhost:"+input.LocalPort)
	rawServerConn, err := net.Dial("tcp", "localhost:"+input.LocalPort)
	if err != nil {
		return nil, nil, clierr.New("failed to establish a connection to the remote proxy server", clierr.Error(err), clierr.Infof("Your grant may have expired, you can check the status here: %s and retry connecting", input.RequestURL))
	}
	// Next, a handshake is performed between the cli client and the Proxy server
	// this handshake establishes the users identity to the Proxy, and also the validity of a Database grant
	handshaker := handshake.NewHandshakeClient(rawServerConn, input.EnsuredAccess.Grant.Id, cfg.TokenSource)
	handshakeResult, err := handshaker.Handshake()
	if err != nil {
		return nil, nil, clierr.New("failed to authenticate connection to the remote proxy server", clierr.Error(err), clierr.Infof("Your grant may have expired, you can check the status here: %s and retry connecting", input.RequestURL))
	}
	clio.Debugw("handshakeResult", "result", handshakeResult)

	// When the handshake process has completed successfully, we use yamux to establish a multiplexed stream over the existing connection
	// We use a multiplexed stream here so that multiple database clients can be connected and have their logs attributed to the same session in our audit trail
	// To the database clients, this is completely opaque
	multiplexedServerClient, err := yamux.Client(rawServerConn, nil)
	if err != nil {
		return nil, nil, err
	}

	// Sanity check to confirm that the multiplexed stream is working
	_, err = multiplexedServerClient.Ping()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to healthcheck the network connection to the proxy server: %w", err)
	}
	return rawServerConn, multiplexedServerClient, nil
}
func clientConnectionParameters(c *cli.Context, ensuredAccess *ensureAccessOutput) (connectionString, cliString, port string, err error) {
	// Print the connection information to the user based on the database they are connecting to
	// the passwords are always 'password' while the username and database will match that of the target being connected to
	yellow := color.New(color.FgYellow)
	// the port that the user connects to
	overridePort := c.Int("port")
	switch ensuredAccess.GrantOutput.RdsDatabase.Engine {
	case "postgres", "aurora-postgresql":
		port = grab.If(overridePort != 0, strconv.Itoa(overridePort), "5432")
		connectionString = yellow.Sprintf("postgresql://%s:password@127.0.0.1:%s/%s?sslmode=disable", ensuredAccess.GrantOutput.User.Username, port, ensuredAccess.GrantOutput.RdsDatabase.Database)
		cliString = yellow.Sprintf(`psql "postgresql://%s:password@127.0.0.1:%s/%s?sslmode=disable"`, ensuredAccess.GrantOutput.User.Username, port, ensuredAccess.GrantOutput.RdsDatabase.Database)
	case "mysql", "aurora-mysql":
		port = grab.If(overridePort != 0, strconv.Itoa(overridePort), "3306")
		connectionString = yellow.Sprintf("%s:password@tcp(127.0.0.1:%s)/%s", ensuredAccess.GrantOutput.User.Username, port, ensuredAccess.GrantOutput.RdsDatabase.Database)
		cliString = yellow.Sprintf(`mysql -u %s -p'password' -h 127.0.0.1 -P %s %s`, ensuredAccess.GrantOutput.User.Username, port, ensuredAccess.GrantOutput.RdsDatabase.Database)
	default:
		return "", "", "", fmt.Errorf("unsupported database engine: %s, maybe you need to update your `cf` cli", ensuredAccess.GrantOutput.RdsDatabase.Engine)
	}
	return
}
func printConnectionParameters(connectionString, cliString, port, engine string) {
	clio.NewLine()
	clio.Infof("Database proxy ready for connections on 127.0.0.1:%s", port)
	clio.NewLine()

	clio.Infof("You can connect now using this connection string: %s", connectionString)
	clio.NewLine()

	clio.Infof("Or using the %s cli: %s", engine, cliString)
	clio.NewLine()
}

func generateRequestURL(ctx context.Context, grant *accessv1alpha1.Grant) (string, error) {
	cfg, err := config.LoadDefault(ctx)
	if err != nil {
		return "", err
	}

	err = cfg.Initialize(ctx, config.InitializeOpts{})
	if err != nil {
		return "", err
	}
	apiURL, err := url.Parse(cfg.APIURL)
	if err != nil {
		return "", err
	}
	p := apiURL.JoinPath("access", "requests", grant.AccessRequestId)
	return p.String(), nil
}

func GrabUnusedPort() (string, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}

	port := listener.Addr().(*net.TCPAddr).Port
	err = listener.Close()
	if err != nil {
		return "", err
	}
	return strconv.Itoa(port), nil
}
