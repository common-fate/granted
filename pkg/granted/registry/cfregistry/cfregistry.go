package cfregistry

import (
	"context"

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

	accountClient := grantedv1alpha1.NewFromConfig(cfg)

	p := Registry{
		opts:   opts,
		Client: accountClient,
	}

	return &p, nil
}

func (r Registry) AWSProfiles(ctx context.Context) (*ini.File, error) {
	// call common fate api to pull profiles

	done := false
	var pageToken string
	profiles := []*awsv1alpha1.Profile{}

	for !done {
		listProfiles, err := r.Client.ListProfiles(ctx, &connect.Request[awsv1alpha1.ListProfilesRequest]{
			Msg: &awsv1alpha1.ListProfilesRequest{
				PageToken: pageToken,
			},
		})
		if err != nil {
			return nil, err
		}

		profiles = append(profiles, listProfiles.Msg.Profiles...)

		if listProfiles.Msg.NextPageToken == "" {
			done = true
		} else {
			pageToken = listProfiles.Msg.NextPageToken
		}
	}

	result := ini.Empty()

	for _, profile := range profiles {

		section, err := result.NewSection(profile.Name)
		if err != nil {
			return nil, err
		}

		//expect all the attributes to come from the api with the correct key value pairs
		for _, attr := range profile.Attributes {
			_, err := section.NewKey(attr.Key, attr.Value)
			if err != nil {
				return nil, err
			}

		}

	}

	return result, nil
}
