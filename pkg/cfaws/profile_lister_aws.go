package cfaws

import "context"

type ShareConfigLister struct {
	loadedProfiles CFSharedConfigs
}

func (s *ShareConfigLister) Load(ctx context.Context) error {
	sc, err := GetProfilesFromDefaultSharedConfig(ctx)
	if err != nil {
		s.loadedProfiles = make(map[string]*CFSharedConfig)
		return err
	}
	s.loadedProfiles = sc
	return nil
}
func (s *ShareConfigLister) Profiles() CFSharedConfigs {
	return s.loadedProfiles
}
func (s *ShareConfigLister) FrecencyKey() string { return "aws_profiles_frecency" }
