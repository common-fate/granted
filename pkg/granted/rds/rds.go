package rds

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"connectrpc.com/connect"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-fate/clio"
	"github.com/common-fate/grab"
	"github.com/common-fate/granted/pkg/cfcfg"
	"github.com/common-fate/granted/pkg/granted/proxy"
	"github.com/common-fate/sdk/config"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	"github.com/common-fate/sdk/service/access"
	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"

	"github.com/urfave/cli/v2"
)

var Command = cli.Command{
	Name:        "rds",
	Usage:       "Granted RDS plugin",
	Description: "Granted RDS plugin",
	Subcommands: []*cli.Command{&proxyCommand},
}

// isLocalMode is used where some behaviour needs to be changed to run against a local development proxy server
func isLocalMode(c *cli.Context) bool {
	return c.String("mode") == "local"
}

var proxyCommand = cli.Command{
	Name:  "proxy",
	Usage: "The Proxy plugin is used in conjunction with a Commnon Fate deployment to request temporary access to an AWS RDS Database",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "target", Aliases: []string{"database"}},
		&cli.StringFlag{Name: "role", Aliases: []string{"user"}},
		&cli.IntFlag{Name: "port", Usage: "The local port to forward the database connection to"},
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

		ensuredAccess, err := proxy.EnsureAccess(ctx, cfg, proxy.EnsureAccessInput[*accessv1alpha1.AWSRDSOutput]{
			Target:               c.String("target"),
			Role:                 c.String("role"),
			Duration:             c.Duration("duration"),
			Reason:               c.String("reason"),
			Confirm:              c.Bool("confirm"),
			Wait:                 c.Bool("wait"),
			PromptForEntitlement: promptForDatabaseAndUser,
			GetGrantOutput: func(msg *accessv1alpha1.GetGrantOutputResponse) (*accessv1alpha1.AWSRDSOutput, error) {
				output := msg.GetOutputAwsRds()
				if output == nil {
					return nil, errors.New("unexpected grant output, this indicates an error in the Common Fate Provisioning process, you should contect your Common Fate administrator")
				}
				return output, nil
			},
		})
		if err != nil {
			return err
		}

		requestURL, err := cfcfg.GenerateRequestURL(cfg.APIURL, ensuredAccess.Grant.AccessRequestId)
		if err != nil {
			return err
		}

		serverPort, localPort, err := proxy.Ports(isLocalMode(c))
		if err != nil {
			return err
		}

		clio.Debugw("prepared ports for access", "serverPort", serverPort, "localPort", localPort)
		if !isLocalMode(c) {
			err = proxy.WaitForSSMConnectionToProxyServer(ctx, proxy.WaitForSSMConnectionToProxyServerOpts{
				AWSConfig: proxy.AWSConfig{
					SSOAccountID:     ensuredAccess.GrantOutput.RdsDatabase.AccountId,
					SSORoleName:      ensuredAccess.Grant.Id,
					SSORegion:        ensuredAccess.GrantOutput.SsoRegion,
					SSOStartURL:      ensuredAccess.GrantOutput.SsoStartUrl,
					Region:           ensuredAccess.GrantOutput.RdsDatabase.Region,
					SSMSessionTarget: ensuredAccess.GrantOutput.SsmSessionTarget,
					NoCache:          c.Bool("no-cache"),
				},
				DisplayOpts: proxy.DisplayOpts{
					Command:     "aws rds proxy",
					SessionType: "RDS Proxy",
				},
				ConnectionOpts: proxy.ConnectionOpts{
					ServerPort: serverPort,
					LocalPort:  localPort,
				},
				GrantID:   ensuredAccess.Grant.Id,
				RequestID: ensuredAccess.Grant.AccessRequestId,
			})
			if err != nil {
				return err
			}
		}

		underlyingProxyServerConn, yamuxStreamConnection, err := proxy.InitiateSessionConnection(cfg, proxy.InitiateSessionConnectionInput{
			GrantID:    ensuredAccess.Grant.Id,
			RequestURL: requestURL,
			LocalPort:  localPort,
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

		return proxy.ListenAndProxy(ctx, yamuxStreamConnection, clientConnectionPort, requestURL)
	},
}

// promptForDatabaseAndUser lists all available database entitlements for the user and displays a table selector UI
func promptForDatabaseAndUser(ctx context.Context, cfg *config.Context) (*accessv1alpha1.Entitlement, error) {
	accessClient := access.NewFromConfig(cfg)
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
		// show the filter dialog when there are 2 or more options
		Filtering(len(options) > 1).
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

func clientConnectionParameters(c *cli.Context, ensuredAccess *proxy.EnsureAccessOutput[*accessv1alpha1.AWSRDSOutput]) (connectionString, cliString, port string, err error) {
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
