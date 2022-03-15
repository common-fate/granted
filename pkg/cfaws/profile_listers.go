package cfaws

import "context"

type ProfileLister interface {
	// load should fetch all the profiles then cache them
	Load(ctx context.Context) error
	// list profiles should return the cached profiles
	Profiles() CFSharedConfigs
	FrecencyKey() string
}

var profileListers []ProfileLister = []ProfileLister{&ShareConfigLister{}}

func LoadProfiles(ctx context.Context) ([]ProfileLister, error) {
	for _, pl := range profileListers {
		err := pl.Load(ctx)
		if err != nil {
			return nil, err
		}
	}
	return profileListers, nil

}

// RegisterProfileLister allows ProfileListers to be registered when using this library as a package in other projects
// position = -1 will append the ProfileLister
// position to insert ProfileLister
func RegisterProfileLister(a ProfileLister, position int) {
	if position < 0 || position > len(profileListers)-1 {
		profileListers = append(profileListers, a)
	} else {
		newProfileListers := append([]ProfileLister{}, profileListers[:position]...)
		newProfileListers = append(newProfileListers, a)
		profileListers = append(newProfileListers, profileListers[position:]...)
	}
}

// list profiles
// get frecent profiles with a frecency key
// merge all the frecent profiles
