package granted

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/granted/pkg/profilegen"
	"github.com/urfave/cli/v2"
	"gopkg.in/ini.v1"
)

var SSOCommand = cli.Command{
	Name:        "sso",
	Usage:       "Manage your local AWS configuration file from information available in AWS SSO",
	Subcommands: []*cli.Command{&GenerateCommand, &PopulateCommand},
}

var GenerateCommand = cli.Command{
	Name:      "generate",
	Usage:     "Prints an AWS configuration file to stdout with profiles from accounts and roles available in AWS SSO",
	UsageText: "granted [global options] sso generate [command options] [sso-start-url]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "prefix", Usage: "Specify a prefix for all generated profile names"},
		&cli.StringFlag{Name: "sso-region", Usage: "Specify the SSO region"},
		&cli.StringFlag{Name: "region", Usage: "The SSO region. Deprecated, use --sso-region instead. In future, this will be the AWS region for the profile to use", DefaultText: "us-east-1"},
		&cli.BoolFlag{Name: "no-credential-process", Usage: "Generate profiles without the Granted credential-process integration"},
		&cli.StringFlag{Name: "profile-template", Usage: "Specify profile name template", Value: profilegen.DefaultProfileNameTemplate}},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		startURL := c.Args().First()
		if startURL == "" {
			return clierr.New("Usage: granted sso populate [sso-start-url]", clierr.Info("For example, granted sso populate https://example.awsapps.com/start"))
		}

		// the --region flag behaviour will change in future: https://github.com/common-fate/granted/issues/360
		//
		// if neither --sso-region or --region were set, show a warning to the user as we plan to make --sso-region required in future
		if !c.IsSet("region") && !c.IsSet("sso-region") {
			clio.Warnf("Please specify the --sso-region flag: 'granted sso populate --sso-region us-east-1 %s'", startURL)
			clio.Warn("Currently, Granted defaults to using us-east-1 if this is not provided. In a future version, this flag will be required (https://github.com/common-fate/granted/issues/360)")
		}

		if c.IsSet("region") {
			clio.Warn("Please use --sso-region rather than --region.")
			clio.Warn("In a future version of Granted, the --region flag will be used to specify the 'region' field in generated profiles, rather than the 'sso_region' field. (https://github.com/common-fate/granted/issues/360)")
		}

		// try --sso-region first, then fall back to --region.
		region := c.String("sso-region")
		if region == "" {
			region = c.String("region")
		}

		// end of --region flag behaviour warnings. These can be removed once https://github.com/common-fate/granted/issues/360 is closed.

		g, err := profilegen.New(profilegen.Opts{
			Output:              os.Stdout,
			Config:              ini.Empty(),
			SSORegion:           region,
			SSOStartURL:         startURL,
			ProfileNameTemplate: c.String("profile-template"),
			NoCredentialProcess: c.Bool("no-credential-process"),
			Prefix:              c.String("prefix"),
		})
		if err != nil {
			return err
		}

		return g.Generate(ctx)
	},
}

var PopulateCommand = cli.Command{
	Name:      "populate",
	Usage:     "Populate your local AWS configuration file with profiles from accounts and roles available in AWS SSO",
	UsageText: "granted [global options] sso populate [command options] [sso-start-url]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "prefix", Usage: "Specify a prefix for all generated profile names"},
		&cli.StringFlag{Name: "sso-region", Usage: "Specify the SSO region"},
		&cli.StringFlag{Name: "region", Usage: "The SSO region. Deprecated, use --sso-region instead. In future, this will be the AWS region for the profile to use", DefaultText: "us-east-1"}, &cli.BoolFlag{Name: "no-credential-process", Usage: "Generate profiles without the Granted credential-process integration"},
		&cli.StringFlag{Name: "profile-template", Usage: "Specify profile name template", Value: profilegen.DefaultProfileNameTemplate}},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		startURL := c.Args().First()
		if startURL == "" {
			return clierr.New("Usage: granted sso populate [sso-start-url]", clierr.Info("For example, granted sso populate https://example.awsapps.com/start"))
		}

		// the --region flag behaviour will change in future: https://github.com/common-fate/granted/issues/360
		//
		// if neither --sso-region or --region were set, show a warning to the user as we plan to make --sso-region required in future
		if !c.IsSet("region") && !c.IsSet("sso-region") {
			clio.Warnf("Please specify the --sso-region flag: 'granted sso populate --sso-region us-east-1 %s'", startURL)
			clio.Warn("Currently, Granted defaults to using us-east-1 if this is not provided. In a future version, this flag will be required (https://github.com/common-fate/granted/issues/360)")
		}

		if c.IsSet("region") {
			clio.Warn("Please use --sso-region rather than --region.")
			clio.Warn("In a future version of Granted, the --region flag will be used to specify the 'region' field in generated profiles, rather than the 'sso_region' field. (https://github.com/common-fate/granted/issues/360)")
		}

		// try --sso-region first, then fall back to --region.
		region := c.String("sso-region")
		if region == "" {
			region = c.String("region")
		}

		// end of --region flag behaviour warnings. These can be removed once https://github.com/common-fate/granted/issues/360 is closed.
		configFilename := config.DefaultSharedConfigFilename()

		f, err := os.OpenFile(configFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		config, err := ini.LoadSources(ini.LoadOptions{
			AllowNonUniqueSections:  false,
			SkipUnrecognizableLines: false,
		}, f)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			config = ini.Empty()
		}

		g, err := profilegen.New(profilegen.Opts{
			Output:              f,
			Config:              config,
			SSORegion:           region,
			SSOStartURL:         startURL,
			ProfileNameTemplate: c.String("profile-template"),
			NoCredentialProcess: c.Bool("no-credential-process"),
			Prefix:              c.String("prefix"),
		})
		if err != nil {
			return err
		}

		return g.Generate(ctx)
	},
}
