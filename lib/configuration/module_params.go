package configuration

import (
	"bytes"
	"fmt"
	"log/slog"

	polochon "github.com/odwrtw/polochon/lib"
	"gopkg.in/yaml.v2"
)

// ModulesParams holds the module params in raw yaml
type ModulesParams struct {
	params map[string][]byte
}

// UnmarshalYAML implements the Unmarshaler interface
func (mp *ModulesParams) UnmarshalYAML(unmarshal func(any) error) error {
	mp.params = map[string][]byte{}

	modules := []map[string]any{}
	if err := unmarshal(&modules); err != nil {
		return err
	}

	for _, module := range modules {
		// Check if the name is present
		nameField, ok := module["name"]
		if !ok {
			return fmt.Errorf("configuration: missing name field in module %+v", module)
		}

		// Check if the name is a string
		name, ok := nameField.(string)
		if !ok {
			return fmt.Errorf("configuration: invalid name field in module %+v", module)
		}

		// Check if the name is not already configured
		if _, ok := mp.params[name]; ok {
			return fmt.Errorf("configuration: duplicate configuration for module %s", name)
		}

		// Marshal the data for later use
		data, err := yaml.Marshal(module)
		if err != nil {
			return err
		}

		mp.params[name] = bytes.Trim(data, "\n")
	}

	return nil
}

// get returns the configured module of type t
func (mp ModulesParams) getModule(t polochon.ModuleType, name string, log *slog.Logger) (polochon.Module, error) {
	module, err := polochon.GetModule(name, t)
	if err != nil {
		return nil, err
	}

	return module, module.Init(mp.params[name], log)
}

func getModulesAs[T any](mp *ModulesParams, t polochon.ModuleType, log *slog.Logger, names []string) ([]T, error) {
	if len(names) == 0 {
		return nil, nil
	}
	res := make([]T, len(names))
	for i, name := range names {
		m, err := mp.getModule(t, name, log)
		if err != nil {
			return nil, err
		}
		res[i] = m.(T)
	}
	return res, nil
}

func getModuleAs[T any](mp *ModulesParams, t polochon.ModuleType, log *slog.Logger, name string) (T, error) {
	var zero T
	if name == "" {
		return zero, nil
	}
	m, err := mp.getModule(t, name, log)
	if err != nil {
		return zero, err
	}
	return m.(T), nil
}
