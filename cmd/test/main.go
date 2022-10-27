package main

// func main() {
// 	err := run()
// 	if err != nil {
// 		fmt.Println(err.Error())
// 	}
// }

// func run() error {
// 	configPath := config.DefaultSharedConfigFilename()
// 	f, err := ini.LoadSources(ini.LoadOptions{
// 		AllowNonUniqueSections:  false,
// 		SkipUnrecognizableLines: false,
// 	}, configPath)
// 	if err != nil {
// 		return err
// 	}
// 	type registry struct {
// 		sections []*ini.Section
// 		hasEnd   bool
// 	}
// 	var registries []registry
// 	var collecting bool
// 	registryIndex := -1
// 	for _, section := range f.Sections() {
// 		copy := section
// 		if strings.HasPrefix(section.Name(), "granted_registry ") {
// 			collecting = true
// 			registries = append(registries, registry{
// 				sections: []rawConfig *ini.Section{copy},
// 			})
// 			registryIndex += 1
// 		} else if collecting {
// 			r := registries[registryIndex]
// 			r.sections = append(r.sections, copy)
// 			registries[registryIndex] = r
// 		}
// 		if strings.HasPrefix(section.Name(), "granted_registry_end ") {
// 			collecting = false
// 			r := registries[registryIndex]
// 			r.hasEnd = true
// 			registries[registryIndex] = r
// 		}
// 	}

// 	for _, reg := range registries[:1] {
// 		for _, section := range reg.sections {
// 			f.DeleteSection(section.Name())
// 		}
// 	}

// 	return f.SaveTo(configPath)
// }
