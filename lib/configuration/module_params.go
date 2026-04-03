package configuration

import (
	"fmt"
	"log/slog"

	"github.com/goccy/go-yaml"

	polochon "github.com/odwrtw/polochon/lib"
)

// ModulesParams holds the module params in raw yaml
type ModulesParams struct {
	params map[string]yaml.RawMessage
}

// UnmarshalYAML implements the Unmarshaler interface
func (mp *ModulesParams) UnmarshalYAML(unmarshal func(any) error) error {
	var rawEntries []yaml.RawMessage
	if err := unmarshal(&rawEntries); err != nil {
		return err
	}

	mp.params = make(map[string]yaml.RawMessage, len(rawEntries))
	for _, raw := range rawEntries {
		var entry struct {
			Name string `yaml:"name"`
		}
		if err := yaml.Unmarshal(raw, &entry); err != nil {
			return err
		}
		if entry.Name == "" {
			return fmt.Errorf("configuration: missing name field in module")
		}
		if _, ok := mp.params[entry.Name]; ok {
			return fmt.Errorf("configuration: duplicate configuration for module %s", entry.Name)
		}
		mp.params[entry.Name] = raw
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
