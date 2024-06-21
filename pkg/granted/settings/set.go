package settings

import (
	"fmt"
	"reflect"
	"strconv"

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
		&cli.StringFlag{Name: "value", Aliases: []string{"v"}, Usage: "The value to set the configuration setting to"},
	},
	Action: func(c *cli.Context) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fields, fieldMap := FieldOptions(cfg)

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
		selectedFieldType := selectedField.ftype.Type
		//optional fields are pointers
		isPointer := selectedFieldType.Kind() == reflect.Ptr
		if isPointer {
			selectedFieldType = selectedFieldType.Elem()
		}
		switch selectedFieldType.Kind() {
		case reflect.Bool:
			if !c.IsSet("value") {
				prompt = &survey.Confirm{
					Message: fmt.Sprintf("Enter new value for %s:", selectedFieldName),
					Default: selectedField.fvalue.Bool(),
				}
				err = survey.AskOne(prompt, &value)
				if err != nil {
					return err
				}
			} else {
				valueStr := c.String("value")
				value, err = strconv.ParseBool(valueStr)
				if err != nil {
					return err
				}
			}

		case reflect.String:
			if !c.IsSet("value") {
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
				value = c.String("value")
			}
		case reflect.Int:
			if !c.IsSet("value") {
				prompt = &survey.Input{
					Message: fmt.Sprintf("Enter new value for %s:", selectedFieldName),
					Default: fmt.Sprintf("%v", selectedField.fvalue.Interface()),
				}
				err = survey.AskOne(prompt, &value)
				if err != nil {
					return err
				}
			} else {
				valueInt := c.String("value")
				value, err = strconv.Atoi(valueInt)
				if err != nil {
					return err
				}
			}
		}

		// Set the new value for the field
		newValue := reflect.ValueOf(value)
		if newValue.Type().ConvertibleTo(selectedFieldType) {
			selectedField.fvalue.Set(newValue.Convert(selectedFieldType))
		} else {
			return fmt.Errorf("invalid type for %s", selectedField.ftype.Name)
		}

		clio.Infof("Updating the value of %s to %v", selectedFieldName, value)
		err = cfg.Save()
		if err != nil {
			return err
		}
		clio.Success("Config updated successfully")
		return nil
	},
}

type field struct {
	ftype  reflect.StructField
	fvalue reflect.Value
}

func FieldOptions(cfg any) ([]string, map[string]field) {
	// Get the type and value of the Config struct
	configType := reflect.TypeOf(cfg)
	configValue := reflect.ValueOf(cfg)

	// Check if cfg is a pointer to a struct
	if configType.Kind() == reflect.Ptr && configType.Elem().Kind() == reflect.Struct {
		configType = configType.Elem()
		configValue = configValue.Elem()
	} else if configType.Kind() != reflect.Struct {
		// cfg is neither a struct nor a pointer to a struct
		return nil, nil
	}

	var fields []string
	var fieldMap = make(map[string]field)

	//traverseConfigFields goes through all config variables taking note of each of the types and saves them to the fieldmap
	//In the case where there are sub fields in the toml config, it is recursively called to traverse the sub config
	var traverseConfigFields func(reflect.Type, reflect.Value, string)
	traverseConfigFields = func(t reflect.Type, v reflect.Value, parent string) {
		for i := 0; i < t.NumField(); i++ {
			fieldType := t.Field(i)
			fieldValue := v.Field(i)
			kind := fieldType.Type.Kind()
			fieldName := fieldType.Name
			if parent != "" {
				fieldName = parent + "." + fieldType.Name
			}

			//subfield structs reflect as a pointer
			if kind == reflect.Ptr {
				// Dereference the pointer to get the underlying value
				if !fieldValue.IsNil() {
					fieldValue = fieldValue.Elem()
					kind = fieldValue.Kind()
				}
			}

			if kind == reflect.Bool || kind == reflect.String || kind == reflect.Int {
				fields = append(fields, fieldName)
				fieldMap[fieldName] = field{
					ftype:  fieldType,
					fvalue: fieldValue,
				}
			} else if kind == reflect.Struct {
				traverseConfigFields(fieldValue.Type(), fieldValue, fieldName)
			}
		}
	}
	traverseConfigFields(configType, configValue, "")

	return fields, fieldMap
}
