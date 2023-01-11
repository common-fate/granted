package granted

import (
	"log"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsre "github.com/aws/aws-sdk-go-v2/service/resourceexplorer2"
	"github.com/urfave/cli/v2"
)

var SearchCommand = cli.Command{
	Name:  "search",
	Usage: "Search for resources",
	Action: func(c *cli.Context) error {
		ctx := c.Context
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatalf("failed to load configuration, %v", err)
		}

		client := awsre.NewFromConfig(cfg)
		v, err := client.GetDefaultView(ctx, &awsre.GetDefaultViewInput{})
		if err != nil {
			return err
		}

		var val string
		err = survey.AskOne(&SelectNetwork{
			Message: "Search:",
			Options: func(filter string, page int) ([]string, int, error) {
				out, err := client.Search(ctx, &awsre.SearchInput{
					QueryString: aws.String(filter),
					ViewArn:     v.ViewArn,
				})
				if err != nil {
					return nil, 0, err
				}

				var labels []string
				for _, r := range out.Resources {
					labels = append(labels, *r.Arn)
				}

				return labels, 0, nil
			},
		}, &val)
		if err != nil {
			return err
		}

		// if err != nil {
		// 	return err
		// }
		// fmt.Println(out.Resources)
		// return nil
		return nil
	},
}
