package registry

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"gopkg.in/yaml.v3"
)

type Registry struct {
	Config         grantedConfig.Registry
	AwsConfigPaths []string               `yaml:"awsConfig"`
	Variables      map[string]interface{} `yaml:"variables"`
	RequiredKeys   map[string]string      `yaml:"requiredKeys"`
}

// GetRegistryLocation returns the directory path where cloned repo is located.
func (r *Registry) getRegistryLocation() (string, error) {
	gConfigPath, err := grantedConfig.GrantedConfigFolder()
	if err != nil {
		return "", err
	}

	return path.Join(gConfigPath, "registries", r.Config.Name), nil
}

func (r *Registry) Parse() error {
	const defaultConfigFilename string = "granted.yml"

	filepath, err := r.getRegistryLocation()
	if err != nil {
		return err
	}

	if r.Config.Path != nil {
		filepath = path.Join(filepath, *r.Config.Path)
	}

	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return err
	}

	// if the provided path is a directory then
	// we will assume that it has default config file i.e `granted.yml` file inside the given directory.
	if fileInfo.IsDir() {
		filepath = path.Join(filepath, defaultConfigFilename)
	}

	clio.Debugf("verifying if valid config exists in %s", filepath)
	file, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(file, r)
	if err != nil {
		return err
	}

	return nil
}

type registryOptions struct {
	name                    string
	path                    string
	ref                     string
	url                     string
	priority                int
	prefixAllProfiles       bool
	prefixDuplicateProfiles bool
}

func NewProfileRegistry(rOpts registryOptions) Registry {
	newRegistry := Registry{
		Config: grantedConfig.Registry{
			Name:                    rOpts.name,
			URL:                     rOpts.url,
			PrefixAllProfiles:       rOpts.prefixAllProfiles,
			PrefixDuplicateProfiles: rOpts.prefixDuplicateProfiles,
			Priority:                &rOpts.priority,
		},
	}

	if rOpts.path != "" {
		newRegistry.Config.Path = &rOpts.path
	}

	if rOpts.ref != "" {
		newRegistry.Config.Ref = &rOpts.ref
	}

	return newRegistry
}

func GetProfileRegistries() ([]Registry, error) {
	gConf, err := grantedConfig.Load()
	if err != nil {
		return nil, err
	}

	if len(gConf.ProfileRegistry.Registries) == 0 {
		return []Registry{}, nil
	}

	registries := []Registry{}
	for name := range gConf.ProfileRegistry.Registries {
		registries = append(registries, Registry{
			Config: gConf.ProfileRegistry.Registries[name],
		})
	}

	// sort registries bases on their priority.
	sort.Slice(registries, func(i, j int) bool {
		var a, b int = 0, 0
		if registries[i].Config.Priority != nil {
			a = *registries[i].Config.Priority
		}

		if registries[j].Config.Priority != nil {
			b = *registries[j].Config.Priority
		}

		return a > b
	})

	return registries, nil
}

// Get the key=value from the provided required variable flag.
func formattedRequiredVars(s string) (string, string, error) {
	value := strings.Split(s, "=")
	if len(value) == 2 {
		return value[0], value[1], nil
	}

	return "", "", fmt.Errorf("invalid value '%s'provided for the required key", s)
}

// granted.yml config might contain user specific variables
// in such case we would prompt users to add them before registry is added.
func (r Registry) PromptRequiredVars(passedRequiredVars []string) error {
	var questions []*survey.Question

	var requiredVarsThroughFlags = make(map[string]string)
	if r.RequiredKeys != nil {
		if len(passedRequiredVars) != 0 {
			for _, val := range passedRequiredVars {
				key, value, err := formattedRequiredVars(val)
				if err != nil {
					return err
				}

				requiredVarsThroughFlags[key] = value
			}
		}

		for key, prompt := range r.RequiredKeys {
			// if the key was passed through cli then skip the prompt
			if _, ok := requiredVarsThroughFlags[key]; ok {
				err := SaveKey(key, requiredVarsThroughFlags[key])
				if err != nil {
					return err
				}

				continue
			}

			qs := survey.Question{
				Name:     key,
				Prompt:   &survey.Input{Message: fmt.Sprintf("'%s': %s", key, prompt)},
				Validate: survey.Required}
			questions = append(questions, &qs)
		}

		ansmap := make(map[string]interface{})

		clio.Info("Your Profile Registry requires you to input values for the following keys:")

		err := survey.Ask(questions, &ansmap)
		if err != nil {
			return err
		}

		err = SaveKeys(ansmap)
		if err != nil {
			return err
		}
	}

	return nil
}

// This is used when user enters the required key through cli prompts.
func SaveKeys(ansmap map[string]interface{}) error {
	gConf, err := grantedConfig.Load()
	if err != nil {
		return err
	}

	for k, v := range ansmap {
		if len(gConf.ProfileRegistry.RequiredKeys) == 0 {
			var requiredKeys = make(map[string]string)
			requiredKeys[k] = v.(string)
			gConf.ProfileRegistry.RequiredKeys = requiredKeys
		} else {
			gConf.ProfileRegistry.RequiredKeys[k] = v.(string)
		}
	}

	err = gConf.Save()
	if err != nil {
		return err
	}

	return nil
}

// This is used when user passed the required value through flag.
func SaveKey(key string, value string) error {
	gConf, err := grantedConfig.Load()
	if err != nil {
		return err
	}

	if len(gConf.ProfileRegistry.RequiredKeys) == 0 {
		var requiredKeys = make(map[string]string)
		requiredKeys[key] = value
		gConf.ProfileRegistry.RequiredKeys = requiredKeys
	} else {
		gConf.ProfileRegistry.RequiredKeys[key] = value
	}

	err = gConf.Save()
	if err != nil {
		return err
	}

	return nil
}
