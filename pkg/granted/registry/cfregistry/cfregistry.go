package cfregistry

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/common-fate/sdk/config"
	awsv1alpha1 "github.com/common-fate/sdk/gen/granted/registry/aws/v1alpha1"
	"github.com/common-fate/sdk/gen/granted/registry/aws/v1alpha1/awsv1alpha1connect"
	grantedv1alpha1 "github.com/common-fate/sdk/service/granted/registry"
	"gopkg.in/ini.v1"
)

type Registry struct {
	opts   Opts
	Client awsv1alpha1connect.ProfileRegistryServiceClient
}

type Opts struct {
	Name string
	URL  string
}

func New(opts Opts) (*Registry, error) {

	cfg, err := config.LoadDefault(context.Background())
	if err != nil {
		return nil, err
	}

	if cfg.APIURL != opts.URL {
		return nil, fmt.Errorf("passed url does not match url in Common Fate (cf) config. active context API URL is: %s", cfg.APIURL)
	}

	accountClient := grantedv1alpha1.NewFromConfig(cfg)

	p := Registry{
		opts:   opts,
		Client: accountClient,
	}

	return &p, nil
}

func (r Registry) AWSProfiles(ctx context.Context) (*ini.File, error) {
	// call common fate api to pull profiles

	profiles, err := r.Client.ListProfiles(ctx, &connect.Request[awsv1alpha1.ListProfilesRequest]{})
	if err != nil {
		return nil, err
	}

	result := ini.Empty()

	//todo pagination
	for _, profile := range profiles.Msg.Profiles {

		section, err := result.NewSection(profile.Name)
		if err != nil {
			return nil, err
		}

		//expect all the attributes to come from the api with the correct key value pairs
		for _, attr := range profile.Attributes {
			section.NewKey(attr.Key, attr.Value)
		}

	}

	return result, nil
}
