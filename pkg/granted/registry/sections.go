package registry

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/cfaws"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/common-fate/granted/pkg/testable"
	"gopkg.in/ini.v1"
)

func getGrantedGeneratedSections(config *ini.File) []*ini.Section {
	var grantedProfiles []*ini.Section

	isAutogeneratedSection := false
	for _, section := range config.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}

		if strings.HasPrefix(section.Name(), "granted_registry_start") && !isAutogeneratedSection {
			isAutogeneratedSection = true
			grantedProfiles = append(grantedProfiles, section)

			continue
		}

		if strings.HasPrefix(section.Name(), "granted_registry_end") {
			isAutogeneratedSection = false
			grantedProfiles = append(grantedProfiles, section)

			continue
		}

		if isAutogeneratedSection {
			grantedProfiles = append(grantedProfiles, section)
		}
	}

	return grantedProfiles

}

func removeAutogeneratedProfileByName(name string) error {
	awsConfigFilepath, err := getDefaultAWSConfigLocation()
	if err != nil {
		return err
	}

	cFile, _, err := loadAWSConfigFile()
	if err != nil {
		return err
	}

	profiles := getGeneratedSectionByName(cFile, name)

	for _, p := range profiles {
		cFile.DeleteSection(p.Name())
	}

	return cFile.SaveTo(awsConfigFilepath)

}

func getGeneratedSectionByName(config *ini.File, name string) []*ini.Section {
	var profiles []*ini.Section

	isAutogeneratedSection := false
	for _, section := range config.Sections() {
		if section.Name() == ini.DefaultSection {
			continue
		}

		if strings.HasPrefix(section.Name(), ("granted_registry_start "+name)) && !isAutogeneratedSection {
			isAutogeneratedSection = true
			profiles = append(profiles, section)

			continue
		}

		if strings.HasPrefix(section.Name(), ("granted_registry_end " + name)) {
			isAutogeneratedSection = false
			profiles = append(profiles, section)

			continue
		}

		if isAutogeneratedSection {
			profiles = append(profiles, section)
		}
	}

	return profiles
}

func removeAutogeneratedProfiles(configFile *ini.File, awsConfigPath string) error {
	grantedProfiles := getGrantedGeneratedSections(configFile)
	// delete all autogenerated sections if any
	if len(grantedProfiles) > 1 {
		for _, gp := range grantedProfiles {
			configFile.DeleteSection(gp.Name())
		}

	}

	err := configFile.SaveTo(awsConfigPath)
	if err != nil {
		return err
	}

	return nil
}

// return all profiles that are not part of granted registry section.
func getNonGrantedProfiles(config *ini.File) []*ini.Section {
	isAutogeneratedSection := false
	var grantedProfiles []string
	for _, section := range config.Sections() {
		if strings.HasPrefix(section.Name(), "granted_registry_start") && !isAutogeneratedSection {
			isAutogeneratedSection = true
			grantedProfiles = append(grantedProfiles, section.Name())

			continue
		}

		if strings.HasPrefix(section.Name(), "granted_registry_end") {
			isAutogeneratedSection = false
			grantedProfiles = append(grantedProfiles, section.Name())

			continue
		}

		if isAutogeneratedSection {
			grantedProfiles = append(grantedProfiles, section.Name())
		}
	}

	var nonGrantedProfiles []*ini.Section
	for _, sec := range config.Sections() {
		if sec.Name() == ini.DefaultSection {
			continue
		}

		if !Contains(grantedProfiles, sec.Name()) {
			nonGrantedProfiles = append(nonGrantedProfiles, sec)
		}
	}

	return nonGrantedProfiles
}

func generateNewRegistrySection(r *Registry, configFile *ini.File, clonedFile *ini.File, opts syncOpts) error {
	sectionName := r.Config.Name
	clio.Debugf("generating section %s", sectionName)

	gconf, err := grantedConfig.Load()
	if err != nil {
		return err
	}

	err = configFile.NewSections(fmt.Sprintf("granted_registry_start %s", sectionName))
	if err != nil {
		return err
	}

	// add "do not edit" msg in the top of autogenerated code.
	if opts.isFirstSection {
		configFile.Section(fmt.Sprintf("granted_registry_start %s", sectionName)).Comment = getAutogeneratedTemplate()
	}

	currentProfiles := configFile.SectionStrings()
	namespace := sectionName

	// iterate each profile section from clonned repo
	// add them to aws config file
	// if there is collision in the profile names then prefix with namespace.
	for _, sec := range clonedFile.Sections() {
		if sec.Name() == ini.DefaultSection {
			continue
		}

		if strings.Contains(sec.Name(), "profile") {
			if cfaws.IsLegalProfileName(strings.TrimPrefix(sec.Name(), "profile ")) {

				if gconf.ProfileRegistry.PrefixAllProfiles || r.Config.PrefixAllProfiles {
					f, err := configFile.NewSection(appendNamespaceToDuplicateSections(sec.Name(), namespace))
					if err != nil {
						return err
					}

					err = copySectionContent(r, sec, f)
					if err != nil {
						return err
					}

					continue
				}

				if Contains(currentProfiles, sec.Name()) {
					// check global config to see if we should prefix all duplicate profiles for this registry.
					if !gconf.ProfileRegistry.PrefixDuplicateProfiles {
						// check registry level config to see if we should prefix the duplicate profiles
						if !r.Config.PrefixDuplicateProfiles {

							// automatically add prefix to duplicate profiles without prompting users.
							clio.Warnf("profile duplication found for '%s'", sec.Name())
							if opts.promptUserIfProfileDuplication {

								const (
									DUPLICATE = "Add registry name as prefix to all duplicate profiles for this registry"
									ABORT     = "Abort, I will manually fix this"
								)

								options := []string{DUPLICATE, ABORT}

								in := survey.Select{Message: "Please select which option would you like to choose to resolve: ", Options: options}
								var selected string
								err = testable.AskOne(&in, &selected)
								if err != nil {
									return err
								}

								if selected == ABORT {
									return fmt.Errorf("aborting sync for registry %s", sectionName)
								}
							}

							r.Config.PrefixDuplicateProfiles = true

							for i, configRegistry := range gconf.ProfileRegistry.Registries {
								if configRegistry.Name == r.Config.Name {
									configRegistry.PrefixDuplicateProfiles = true
									gconf.ProfileRegistry.Registries[i] = configRegistry

									err := gconf.Save()
									if err != nil {
										return err
									}
								}
							}
						}
					}

					clio.Debugf("Prefixing %s to avoid collision.", sec.Name())
					f, err := configFile.NewSection(appendNamespaceToDuplicateSections(sec.Name(), namespace))
					if err != nil {
						return err
					}

					err = copySectionContent(r, sec, f)
					if err != nil {
						return err
					}

					if f.Comment == "" {
						f.Comment = "# profile name has been prefixed due to duplication"
					} else {
						f.Comment = "# profile name has been prefixed due to duplication. \n" + sec.Comment
					}

					continue
				}

				f, err := configFile.NewSection(sec.Name())
				if err != nil {
					return err
				}

				err = copySectionContent(r, sec, f)
				if err != nil {
					return err
				}

				f.Comment = sec.Comment
			}

		} else {
			// any other section is copied as it is.
			f, err := configFile.NewSection(sec.Name())
			if err != nil {
				return err
			}

			err = copySectionContent(r, sec, f)
			if err != nil {
				return err
			}

			f.Comment = sec.Comment
		}

	}

	err = configFile.NewSections(fmt.Sprintf("granted_registry_end %s", sectionName))
	if err != nil {
		return err
	}

	return nil
}

func copySectionContent(r *Registry, s *ini.Section, d *ini.Section) error {
	for _, key := range s.KeyStrings() {
		value := s.Key(key).Value()

		// check if the value contains go-template
		if containsTemplate(value) {
			output, err := interpolateVariables(r, value, strings.TrimPrefix(s.Name(), "profile "))
			if err != nil {
				return err
			}

			_, err = d.NewKey(key, output)
			if err != nil {
				return err
			}

		} else {
			_, err := d.NewKey(key, s.Key(key).Value())
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func Contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}

	return false
}

func appendNamespaceToDuplicateSections(pName string, namespace string) string {
	regx := regexp.MustCompile(`(.*profile\s+)(?P<name>[^\n\r]*)`)

	if regx.MatchString(pName) {
		matches := regx.FindStringSubmatch(pName)
		nameIndex := regx.SubexpIndex("name")

		return "profile " + namespace + "." + matches[nameIndex]
	}

	return pName
}
