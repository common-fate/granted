package profilegen

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/common-fate/awsconfigfile"
	"github.com/common-fate/granted/pkg/cfaws"
	"golang.org/x/sync/errgroup"
	"gopkg.in/ini.v1"
)

type Source interface {
	GetProfiles(ctx context.Context) ([]awsconfigfile.SSOProfile, error)
}

// Generator generates AWS profiles for ~/.aws/config.
// It reads profiles from sources and merges them with
// an existing ini config file.
type Generator struct {
	// Output to write to
	output              io.Writer
	sources             []Source
	config              *ini.File
	noCredentialProcess bool
	profileNameTemplate string
	prefix              string
}

type Opts struct {
	Output              io.Writer
	Config              *ini.File
	SSORegion           string
	SSOStartURL         string
	ProfileNameTemplate string
	NoCredentialProcess bool
	Prefix              string
}

func New(opts Opts) (*Generator, error) {
	region, err := cfaws.ExpandRegion(opts.SSORegion)
	if err != nil {
		return nil, err
	}

	g := &Generator{
		output:              opts.Output,
		config:              opts.Config,
		noCredentialProcess: opts.NoCredentialProcess,
		profileNameTemplate: opts.ProfileNameTemplate,
		sources: []Source{
			// use the AWS SSO as a default profile source
			AWSSSOSource{
				SSORegion: region,
				StartURL:  opts.SSOStartURL,
			},
		},
	}
	if opts.ProfileNameTemplate == "" {
		g.profileNameTemplate = DefaultProfileNameTemplate
	}

	return g, nil
}

const profileSectionIllegalChars = ` \][;'"`

// regular expression that matches on the characters \][;'" including whitespace, but does not match anything between {{ }} so it does not check inside go templates
// this regex is used as a basic safeguard to help users avoid mistakes in their templates
// for example "{{ .AccountName }} {{ .RoleName }}" this is invalid because it has a whitespace separating the template elements
var profileSectionIllegalCharsRegex = regexp.MustCompile(`(?s)((?:^|[^\{])[\s\][;'"]|[\][;'"][\s]*(?:$|[^\}]))`)
var matchGoTemplateSection = regexp.MustCompile(`\{\{[\s\S]*?\}\}`)

var DefaultProfileNameTemplate = "{{ .AccountName }}/{{ .RoleName }}"

// Generate AWS profiles and merge them with the existing config.
// Writes output to the generator's output.
func (g *Generator) Generate(ctx context.Context) error {
	var eg errgroup.Group
	var mu sync.Mutex
	var profiles []awsconfigfile.SSOProfile

	if strings.ContainsAny(g.prefix, profileSectionIllegalChars) {
		return fmt.Errorf("profile prefix must not contain any of these illegal characters (%s)", profileSectionIllegalChars)
	}

	// check the profile template for any invalid section name characters
	if g.profileNameTemplate != DefaultProfileNameTemplate {
		cleaned := matchGoTemplateSection.ReplaceAllString(g.profileNameTemplate, "")
		if profileSectionIllegalCharsRegex.MatchString(cleaned) {
			return fmt.Errorf("profile template must not contain any of these illegal characters (%s)", profileSectionIllegalChars)
		}
	}

	for _, s := range g.sources {
		scopy := s
		eg.Go(func() error {
			got, err := scopy.GetProfiles(ctx)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			profiles = append(profiles, got...)
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	err = awsconfigfile.Merge(awsconfigfile.MergeOpts{
		Config:              g.config,
		SectionNameTemplate: g.profileNameTemplate,
		Profiles:            profiles,
		NoCredentialProcess: g.noCredentialProcess,
		Prefix:              g.prefix,
	})
	if err != nil {
		return err
	}
	_, err = g.config.WriteTo(g.output)
	return err
}
