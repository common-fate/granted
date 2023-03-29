package settings

import (
	"fmt"
	"reflect"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

var SetConfigCommand = cli.Command{
	Name:  "set",
	Usage: "Set a value in settings",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "setting", Aliases: []string{"s"}, Usage: "The name of a configuration setting, currently only string, int and bool types are supported. e.g 'DisableUsageTips'. For other configuration, set the value using builtin commands or by directly modifying the config file for advanced use cases."},
		&cli.StringFlag{Name: "string-value"},
		&cli.BoolFlag{Name: "bool-value"},
		&cli.IntFlag{Name: "int-value"},
	},
	Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Get the type and value of the Config struct
		configType := reflect.TypeOf(*cfg)
		configValue := reflect.ValueOf(cfg).Elem()
		type field struct {
			ftype  reflect.StructField
			fvalue reflect.Value
		}
		var fields []string
		var fieldMap = make(map[string]field)
		// Iterate over the fields of the Config struct
		for i := 0; i < configType.NumField(); i++ {
			fieldType := configType.Field(i)
			kind := fieldType.Type.Kind()
			if kind == reflect.Bool || kind == reflect.String || kind == reflect.Int {
				fieldValue := configValue.Field(i)
				fields = append(fields, fieldType.Name)
				fieldMap[fieldType.Name] = field{
					fvalue: fieldValue,
					ftype:  fieldType,
				}
			}
		}

		var selectedFieldName = c.String("setting")
		if selectedFieldName == "" {
			p := &survey.Select{
				Message: "Select the configuration to change",
				Options: fields,
			}
			err = survey.AskOne(p, &selectedFieldName)
			if err != nil {
				return err
			}
		}

		var selectedField field
		var ok bool
		selectedField, ok = fieldMap[selectedFieldName]
		if !ok {
			return fmt.Errorf("the selected field %s is not a valid config parameter", selectedFieldName)
		}
		// Prompt the user to update the field
		var value interface{}
		var prompt survey.Prompt
		switch selectedField.ftype.Type.Kind() {
		case reflect.Bool:
			if c.Count("bool-value") == 0 {
				prompt = &survey.Confirm{
					Message: fmt.Sprintf("Enter new value for %s:", selectedFieldName),
					Default: selectedField.fvalue.Bool(),
				}
				err = survey.AskOne(prompt, &value)
				if err != nil {
					return err
				}
			} else {
				value = c.Bool("bool-value")
			}

		case reflect.String:
			if c.Count("string-value") == 0 {
				var str string
				prompt = &survey.Input{
					Message: fmt.Sprintf("Enter new value for %s:", selectedFieldName),
					Default: fmt.Sprintf("%v", selectedField.fvalue.Interface()),
				}
				err = survey.AskOne(prompt, &str)
				if err != nil {
					return err
				}
				value = str
			} else {
				value = c.String("string-value")
			}
		case reflect.Int:
			if c.Count("int-value") == 0 {
				prompt = &survey.Input{
					Message: fmt.Sprintf("Enter new value for %s:", selectedFieldName),
					Default: fmt.Sprintf("%v", selectedField.fvalue.Interface()),
				}
				err = survey.AskOne(prompt, &value)
				if err != nil {
					return err
				}
			} else {
				value = c.Int("int-value")
			}
		}

		// Set the new value for the field
		newValue := reflect.ValueOf(value)
		if newValue.Type().ConvertibleTo(selectedField.ftype.Type) {
			selectedField.fvalue.Set(newValue.Convert(selectedField.ftype.Type))
		} else {
			return fmt.Errorf("invalid type for %s", selectedField.ftype.Name)
		}

		clio.Infof("Updating the value of %s to %v", selectedFieldName, value)
		err = cfg.Save()
		if err != nil {
			return err
		}
		clio.Success("Config updated successfully")
		// Call the Save method to save the updated struct
		return nil
	},
}
