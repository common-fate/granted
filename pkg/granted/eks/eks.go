package eks

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/common-fate/cli/printdiags"
	"github.com/common-fate/clio"
	"github.com/common-fate/grab"
	"github.com/common-fate/granted/pkg/cfaws"
	"github.com/common-fate/sdk/config"
	accessv1alpha1 "github.com/common-fate/sdk/gen/commonfate/access/v1alpha1"
	entityv1alpha1 "github.com/common-fate/sdk/gen/commonfate/entity/v1alpha1"
	"github.com/common-fate/sdk/service/access"
	"github.com/common-fate/sdk/service/access/grants"
	"github.com/common-fate/sdk/service/entity"

	"github.com/urfave/cli/v2"
)

var Command = cli.Command{
	Name:        "eks",
	Usage:       "Granted EKS plugin",
	Description: "Granted EKS plugin",
	Subcommands: []*cli.Command{&proxyCommand},
}

var proxyCommand = cli.Command{
	Name:  "get-token",
	Usage: "Retrieves a token for Just-In-Time access to an EKS cluster",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "cluster-arn"},
		&cli.StringFlag{Name: "role"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		cfg, err := config.LoadDefault(ctx)
		if err != nil {
			return err
		}

		target := c.String("cluster-arn")
		role := c.String("role")
		client := access.NewFromConfig(cfg)

		req := accessv1alpha1.BatchEnsureRequest{
			Entitlements: []*accessv1alpha1.EntitlementInput{
				{
					Target: &accessv1alpha1.Specifier{
						Specify: &accessv1alpha1.Specifier_Eid{
							Eid: &entityv1alpha1.EID{
								Type: "AWS::EKS::Cluster",
								Id:   target,
							},
						},
					},
					Role: &accessv1alpha1.Specifier{
						Specify: &accessv1alpha1.Specifier_Lookup{
							Lookup: role,
						},
					},
				},
			},
			Justification: &accessv1alpha1.Justification{},
		}

		result, err := client.BatchEnsure(ctx, connect.NewRequest(&req))
		if err != nil {
			return err
		}

		printdiags.Print(result.Msg.Diagnostics, nil)

		if result == nil || len(result.Msg.Grants) == 0 {
			return errors.New("could not load grant from Common Fate")
		}

		grant := result.Msg.Grants[0]

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

		var grantOutput AWSEKSGrantOutput
		var found bool

		for _, child := range children {
			clio.Debugw("grant child", "child", child)
			if child.Eid.Type == EKSGrantOutputType {
				found = true
				err = entity.Unmarshal(child, &grantOutput)
				if err != nil {
					return err
				}
			}
		}

		if !found {
			return errors.New("did not find a grant output entity in query grant children response")
		}

		clio.Debugw("grant output", "grantoutput", grantOutput)

		p := &cfaws.Profile{
			Name:        grant.Grant.Id,
			ProfileType: "AWS_SSO",
			AWSConfig: awsConfig.SharedConfig{
				SSOAccountID: grantOutput.AWSAccountID,
				SSORoleName:  grant.Grant.Id,
				SSORegion:    grantOutput.SSORegion,
				SSOStartURL:  grantOutput.SSOStartURL,
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

		awscfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)))
		if err != nil {
			return err
		}
		awscfg.Region = grantOutput.ClusterRegion
		stsClient := sts.NewFromConfig(awscfg)

		token, err := getToken(ctx, stsClient, "example-cluster")
		if err != nil {
			return err
		}

		auth, err := getExecAuth(token)
		if err != nil {
			return err
		}

		fmt.Println(auth)
		return nil
	},
}
