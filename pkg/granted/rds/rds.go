package rds

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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
	entityv1alpha1 "github.com/common-fate/sdk/gen/commonfate/entity/v1alpha1"
	"github.com/common-fate/sdk/handshake"
	"github.com/common-fate/sdk/service/access"
	"github.com/common-fate/sdk/service/access/grants"
	"github.com/common-fate/sdk/service/entity"
	"github.com/common-fate/xid"
	"github.com/fatih/color"
	"github.com/hashicorp/yamux"
	"github.com/mattn/go-runewidth"
	sethRetry "github.com/sethvargo/go-retry"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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
		&cli.StringFlag{Name: "target"},
		&cli.StringFlag{Name: "role"},
		&cli.IntFlag{Name: "port", Usage: "The local port to forward the mysql database connection to"},
		&cli.StringFlag{Name: "reason", Usage: "Provide a reason for requesting access to the role"},
		&cli.BoolFlag{Name: "confirm", Aliases: []string{"y"}, Usage: "Skip confirmation prompts for access requests"},
		&cli.BoolFlag{Name: "wait", Value: true, Usage: "Wait for the access request to be approved."},
		&cli.BoolFlag{Name: "no-cache", Usage: "Disables caching of session credentials and forces a refresh", EnvVars: []string{"GRANTED_NO_CACHE"}},
		&cli.DurationFlag{Name: "duration", Aliases: []string{"d"}, Usage: "The duration for your access request"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		cfg, err := config.LoadDefault(ctx)
		if err != nil {
			return err
		}
		apiURL, err := url.Parse(cfg.APIURL)
		if err != nil {
			return err
		}

		err = cfg.Initialize(ctx, config.InitializeOpts{})
		if err != nil {
			return err
		}

		target := c.String("target")
		role := c.String("role")
		client := access.NewFromConfig(cfg)

		if target == "" && role == "" {
			entitlements, err := grab.AllPages(ctx, func(ctx context.Context, nextToken *string) ([]*accessv1alpha1.Entitlement, *string, error) {
				res, err := client.QueryEntitlements(ctx, connect.NewRequest(&accessv1alpha1.QueryEntitlementsRequest{
					PageToken:  grab.Value(nextToken),
					TargetType: grab.Ptr("AWS::RDS::Database"),
				}))
				if err != nil {
					return nil, nil, err
				}
				return res.Msg.Entitlements, &res.Msg.NextPageToken, nil
			})
			if err != nil {
				return err
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
				target := lipgloss.NewStyle().Padding(0).Render(style.Render(runewidth.Truncate(entitlement.Target.Display(), cols[0].Width, "…")))

				style = lipgloss.NewStyle().Width(cols[1].Width).MaxWidth(cols[1].Width).Inline(true)
				role := lipgloss.NewStyle().Padding(0).Render(style.Render(runewidth.Truncate(entitlement.Role.Display(), cols[1].Width, "…")))

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
				return err
			}

			entitlement := selector.GetValue().(*accessv1alpha1.Entitlement)

			target = entitlement.Target.Eid.Display()
			role = entitlement.Role.Eid.Display()

		}

		var duration *durationpb.Duration
		if c.Duration("duration") != 0 {
			duration = durationpb.New(c.Duration("duration"))
		}

		input := accessrequesthook.NoEntitlementAccessInput{
			Entitlements: []*accessv1alpha1.EntitlementInput{
				{
					Target: &accessv1alpha1.Specifier{
						Specify: &accessv1alpha1.Specifier_Lookup{
							Lookup: target,
						},
					},
					Role: &accessv1alpha1.Specifier{
						Specify: &accessv1alpha1.Specifier_Lookup{
							Lookup: role,
						},
					},
				},
			},
			Reason:    c.String("reason"),
			Duration:  duration,
			Confirm:   c.Bool("confirm"),
			Wait:      c.Bool("wait"),
			StartTime: time.Now(),
		}
		hook := accessrequesthook.Hook{}
		retry, result, err := hook.NoEntitlementAccess(ctx, cfg, input)
		if err != nil {
			return err
		}

		retryDuration := time.Minute * 1
		if c.Bool("wait") {
			//if wait is specified, increase the timeout to 15 minutes.
			retryDuration = time.Minute * 15
		}

		if retry {
			// reset the start time for the timer (otherwise it shows 2s, 7s, 12s etc)
			input.StartTime = time.Now()

			b := sethRetry.NewConstant(5 * time.Second)
			b = sethRetry.WithMaxDuration(retryDuration, b)
			err = sethRetry.Do(c.Context, b, func(ctx context.Context) (err error) {

				//also proactively check if request has been approved and attempt to activate
				result, err = hook.RetryNoEntitlementAccess(ctx, cfg, input)
				if err != nil {

					return sethRetry.RetryableError(err)
				}

				return nil
			})
			if err != nil {
				return err
			}

		}

		clio.Info("Grant is activated")

		if result == nil || len(result.Grants) == 0 {
			return errors.New("could not load grant from Common Fate")
		}

		grant := result.Grants[0]

		grantsClient := grants.NewFromConfig(cfg)

		children, err := grab.AllPages(ctx, func(ctx context.Context, nextToken *string) ([]*entityv1alpha1.Entity, *string, error) {
			res, err := grantsClient.QueryGrantChildren(ctx, connect.NewRequest(&accessv1alpha1.QueryGrantChildrenRequest{
				Id:        grant.Grant.Id,
				PageToken: grab.Value(nextToken),
			}))
			if err != nil {
				return nil, nil, err
			}
			return res.Msg.Entities, &res.Msg.NextPageToken, nil
		})
		if err != nil {
			return err
		}

		// find an unused local port to use for the ssm server
		// the user doesn't directly connect to this, they connect through our local proxy
		// which adds authentication
		ssmPortforwardLocalPort, err := GrabUnusedPort()
		if err != nil {
			return err
		}

		clio.Debugf("starting SSM portforward on local port: %s", ssmPortforwardLocalPort)

		commandData := CommandData{
			// the proxy server always runs on port 7070
			SSMPortForwardServerPort: "8080",
			SSMPortForwardLocalPort:  ssmPortforwardLocalPort,
		}

		// in local dev we run on a different port because the control plane already runs on 8080
		if os.Getenv("CF_DEV_PROXY") == "true" {
			commandData.SSMPortForwardServerPort = "7070"
		}

		for _, child := range children {
			if child.Eid.Type == GrantOutputType {
				err = entity.Unmarshal(child, &commandData.GrantOutput)
				if err != nil {
					return err
				}
			}
		}

		if commandData.GrantOutput.Grant.ID == "" {
			return errors.New("did not find a grant output entity in query grant children response")
		}

		clio.Debugw("command data", "commandData", commandData)

		p := &cfaws.Profile{
			Name:        grant.Grant.Id,
			ProfileType: "AWS_SSO",
			AWSConfig: awsConfig.SharedConfig{
				SSOAccountID: commandData.GrantOutput.Database.Account.ID,
				SSORoleName:  grant.Grant.Id,
				SSORegion:    commandData.GrantOutput.SSORegion,
				SSOStartURL:  commandData.GrantOutput.SSOStartURL,
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

		// the port that the user connects to
		overridePort := c.Int("port")

		ssmReadyForConnectionsChan := make(chan struct{})

		awscfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)))
		if err != nil {
			return err
		}
		awscfg.Region = commandData.GrantOutput.Database.Region
		ssmClient := ssm.NewFromConfig(awscfg)

		// listen for interrupt signals and forward them on
		// listen for a context cancellation

		// Set up a channel to receive OS signals
		sigs := make(chan os.Signal, 1)
		// Notify sigs on os.Interrupt (Ctrl+C)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		eg, ctx := errgroup.WithContext(ctx)

		startingProxySpinner := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		startingProxySpinner.Suffix = " Starting database proxy..."
		startingProxySpinner.Writer = os.Stderr

		var sessionOutput *ssm.StartSessionOutput
		// in local dev you can skip using ssm and just use a local port forward instead
		if os.Getenv("CF_DEV_PROXY") == "true" {
			commandData.SSMPortForwardLocalPort = commandData.SSMPortForwardServerPort
			go func() { ssmReadyForConnectionsChan <- struct{}{} }()
		} else {
			documentName := "AWS-StartPortForwardingSession"
			startSessionInput := ssm.StartSessionInput{
				Target:       &commandData.GrantOutput.SSMSessionTarget,
				DocumentName: &documentName,
				Parameters: map[string][]string{
					"portNumber":      {commandData.SSMPortForwardServerPort},
					"localPortNumber": {commandData.SSMPortForwardLocalPort},
				},
			}

			sessionOutput, err = ssmClient.StartSession(ctx, &startSessionInput)
			if err != nil {
				return err
			}

			// Connect to the Proxy server using SSM
			eg.Go(func() error {
				clientId := xid.New("gtd")
				ssmSession := session.Session{
					StreamUrl:             *sessionOutput.StreamUrl,
					SessionId:             *sessionOutput.SessionId,
					TokenValue:            *sessionOutput.TokenValue,
					IsAwsCliUpgradeNeeded: false,
					Endpoint:              "localhost:" + commandData.SSMPortForwardLocalPort,
					DataChannel:           &datachannel.DataChannel{},
					ClientId:              clientId,
				}

				startingProxySpinner.Start()
				defer startingProxySpinner.Stop()

				// registers the PortSession feature within the ssm library
				_ = portsession.PortSession{}

				// Terminate this process if the context is cancelled
				go func() {
					<-ctx.Done()
					err := ssmSession.TerminateSession(&SSMDebugLogger{
						Writers: []io.Writer{DebugWriter{}},
					})
					if err != nil {
						clio.Debug(err)
					}
				}()

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

				// Execute starts the ssm connection
				err = ssmSession.Execute(ssmLogger)
				if err != nil {
					return clierr.New(fmt.Errorf("AWS SSM port forward session closed with an error: %w", err).Error(),
						clierr.Info("You can try re-running this command with the verbose flag to see detailed logs, 'cf --verbose aws rds proxy'"),
						clierr.Infof("In rare cases, where the database proxy has been re-deployed while your grant was active, you will need to close your request in Common Fate and request access again 'cf access close request --id=%s' This is usually indicated by an error message containing '(TargetNotConnected) when calling the StartSession'", grant.Grant.AccessRequestId))
				}
				return nil
			})
		}

		// Wait for SSM to be connected and ready, then handle database client connections
		eg.Go(func() error {
			select {
			case <-ssmReadyForConnectionsChan:
				startingProxySpinner.Stop()
			case <-ctx.Done():
				startingProxySpinner.Stop()
				return nil
			}

			// First dial the local SSM portforward, which will be running on a randomly chosen port
			// this establishes our initial connection to the Proxy server
			rawServerConn, err := net.Dial("tcp", "localhost:"+commandData.SSMPortForwardLocalPort)
			if err != nil {
				return fmt.Errorf("failed to establish a connection to the remote proxy server: %w", err)
			}

			// Next, a handshake is performed between the cli client and the Proxy server
			// this handshake establishes the users identity to the Proxy, and also the validity of a Database grant
			handshaker := handshake.NewHandshakeClient(rawServerConn, grant.Grant.Id, cfg.TokenSource)
			handshakeResult, err := handshaker.Handshake()
			if err != nil {
				return clierr.New("failed to authenticate connection to the remote proxy server while accepting local connection", clierr.Error(err), clierr.Infof("Your grant may have expired, you can check the status here: %s", requestURL(apiURL, grant.Grant)))
			}
			clio.Debugw("handshakeResult", "result", handshakeResult)

			// When the handshake process has completed successfully, we use yamux to establish a multiplexed stream over the existing connection
			// We use a multiplexed stream here so that multiple database clients can be connected and have their logs attributed to the same session in our audit trail
			// To the database clients, this is completely opaque
			multiplexedServerClient, err := yamux.Client(rawServerConn, nil)
			if err != nil {
				return err
			}

			// Sanity check to confirm that the multiplexed stream is working
			_, err = multiplexedServerClient.Ping()
			if err != nil {
				return err
			}

			// Print the connection information to the user based on the database they are connecting to
			// the passwords are always 'password' while the username and database will match that of the target being connected to
			var connectionString, cliString, port string
			yellow := color.New(color.FgYellow)
			switch commandData.GrantOutput.Database.Engine {
			case "postgres":
				port = grab.If(overridePort != 0, strconv.Itoa(overridePort), "5432")
				connectionString = yellow.Sprintf("postgresql://%s:password@127.0.0.1:%s/%s?sslmode=disable", commandData.GrantOutput.User.Username, port, commandData.GrantOutput.Database.Database)
				cliString = yellow.Sprintf(`psql "postgresql://%s:password@127.0.0.1:%s/%s?sslmode=disable"`, commandData.GrantOutput.User.Username, port, commandData.GrantOutput.Database.Database)
			case "mysql":
				port = grab.If(overridePort != 0, strconv.Itoa(overridePort), "3306")
				connectionString = yellow.Sprintf("%s:password@tcp(127.0.0.1:%s)/%s", commandData.GrantOutput.User.Username, port, commandData.GrantOutput.Database.Database)
				cliString = yellow.Sprintf(`mysql -u %s -p'password' -h 127.0.0.1 -P %s %s`, commandData.GrantOutput.User.Username, port, commandData.GrantOutput.Database.Database)
			default:
				return fmt.Errorf("unsupported database engine: %s, maybe you need to update your `cf` cli", commandData.GrantOutput.Database.Engine)
			}

			clio.NewLine()
			clio.Infof("Database proxy ready for connections on 127.0.0.1:%s", port)
			clio.NewLine()

			clio.Infof("You can connect now using this connection string: %s", connectionString)
			clio.NewLine()

			clio.Infof("Or using the %s cli: %s", commandData.GrantOutput.Database.Engine, cliString)
			clio.NewLine()

			defer cancel()
			defer rawServerConn.Close()

			ln, err := net.Listen("tcp", "localhost:"+port)
			if err != nil {
				clio.Errorw(fmt.Sprintf("failed to start listening for connections on port: %s", port), zap.Error(err))
			}
			defer ln.Close()

			for {
				connChan := make(chan net.Conn)
				errChan := make(chan error, 10)

				go func() {
					conn, err := ln.Accept()
					if err != nil {
						errChan <- err
						return
					}

					clio.Debug("accepted connection for database client")
					connChan <- conn
				}()

				select {
				case <-ctx.Done():
					clio.Debug("context cancelled shutting down port forward")
					return nil // Context cancelled, exit the loop
				case err := <-errChan:
					clio.Errorw("failed to accept new connection", zap.Error(err))
					return err
				case databaseClientConn := <-connChan:
					// When a database client connection is accepted, start processing the connection in a go routine and continue listening
					eg.Go(func() error {
						// A stream is opened for this connection, streams are used just like a net.Conn and can read and write data
						// A stream can only be opened while the grant is still valid, and each new connection will validate the database parameters username and database
						sessionConn, err := multiplexedServerClient.OpenStream()
						if err != nil {
							return clierr.New("Failed to authenticate connection to the remote proxy server while accepting local connection", clierr.Error(err), clierr.Infof("Your grant may have expired, you can check the status here: %s", requestURL(apiURL, grant.Grant)))
						}

						clio.Infof("Connection accepted for session [%v]", sessionConn.StreamID())

						// If a stream successfully connects, that means that a connection to the target database is now open
						// at this point the connection traffic is handed off and the connection is effectively directly from the database client and the target database
						// with queries being intercepted and logged to the audit trail in Common Fate
						// if the grant becomes incative at any time the connection is terminated immediately
						go func() {
							defer databaseClientConn.Close()
							defer sessionConn.Close()
							_, err := io.Copy(sessionConn, databaseClientConn)
							if err != nil {
								clio.Debugw("error writing data from client to server usually this is just because the database proxy session ended.", "streamId", sessionConn.StreamID(), zap.Error(err))
							}
							clio.Infof("Connection ended for session [%v]", sessionConn.StreamID())
						}()
						go func() {
							defer databaseClientConn.Close()
							defer sessionConn.Close()
							_, err := io.Copy(databaseClientConn, sessionConn)
							if err != nil {
								clio.Debugw("error writing data from server to client usually this is just because the database proxy session ended.", "streamId", sessionConn.StreamID(), zap.Error(err))
							}
						}()
						return nil

					})

				}
			}

		})

		eg.Go(func() error {
			select {
			case <-sigs:
				clio.Info("Received interrupt signal, shutting down database proxy...")
				cancel()
			case <-ctx.Done():
				clio.Info("Shutting down database proxy...")
			}

			return nil
		})

		return eg.Wait()
	},
}

func requestURL(apiURL *url.URL, grant *accessv1alpha1.Grant) string {
	p := apiURL.JoinPath("access", "requests", grant.AccessRequestId)
	return p.String()
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

type CommandData struct {
	GrantOutput              AWSRDS
	SSMPortForwardLocalPort  string
	SSMPortForwardServerPort string
}
