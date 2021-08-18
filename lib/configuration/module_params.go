package configuration

import (
	"bytes"
	"errors"
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
	"gopkg.in/yaml.v2"
)

// Custom errors
var (
	ErrMissingDetailerNames  = errors.New("configuration: missing detailer names")
	ErrMissingTorrenterNames = errors.New("configuration: missing torrenter names")
)

// ModulesParams holds the module params in raw yaml
type ModulesParams struct {
	params map[string][]byte
}

// UnmarshalYAML implements the Unmarshaler interface
func (mp *ModulesParams) UnmarshalYAML(unmarshal func(interface{}) error) error {
	mp.params = map[string][]byte{}

	modules := []map[string]interface{}{}
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
func (mp ModulesParams) getModule(t polochon.ModuleType, name string) (polochon.Module, error) {
	module, err := polochon.GetModule(name, t)
	if err != nil {
		return nil, err
	}

	return module, module.Init(mp.params[name])
}

func (mp ModulesParams) getModules(t polochon.ModuleType, names ...string) ([]polochon.Module, error) {
	modules := []polochon.Module{}

	for _, name := range names {
		module, err := mp.getModule(t, name)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	return modules, nil
}

func (mp ModulesParams) getDetailers(names []string) ([]polochon.Detailer, error) {
	if len(names) == 0 {
		return nil, ErrMissingDetailerNames
	}

	modules, err := mp.getModules(polochon.TypeDetailer, names...)
	if err != nil {
		return nil, err
	}

	res := []polochon.Detailer{}
	for _, m := range modules {
		res = append(res, m.(polochon.Detailer))
	}
	return res, nil
}

func (mp ModulesParams) getSubtitlers(names []string) ([]polochon.Subtitler, error) {
	modules, err := mp.getModules(polochon.TypeSubtitler, names...)
	if err != nil {
		return nil, err
	}

	res := []polochon.Subtitler{}
	for _, m := range modules {
		res = append(res, m.(polochon.Subtitler))
	}
	return res, nil
}

func (mp ModulesParams) getExplorers(names []string) ([]polochon.Explorer, error) {
	modules, err := mp.getModules(polochon.TypeExplorer, names...)
	if err != nil {
		return nil, err
	}

	res := []polochon.Explorer{}
	for _, m := range modules {
		res = append(res, m.(polochon.Explorer))
	}
	return res, nil
}

func (mp ModulesParams) getSearchers(names []string) ([]polochon.Searcher, error) {
	modules, err := mp.getModules(polochon.TypeSearcher, names...)
	if err != nil {
		return nil, err
	}

	res := []polochon.Searcher{}
	for _, m := range modules {
		res = append(res, m.(polochon.Searcher))
	}
	return res, nil
}

func (mp ModulesParams) getWishlisters(names []string) ([]polochon.Wishlister, error) {
	modules, err := mp.getModules(polochon.TypeWishlister, names...)
	if err != nil {
		return nil, err
	}

	res := []polochon.Wishlister{}
	for _, m := range modules {
		res = append(res, m.(polochon.Wishlister))
	}
	return res, nil
}

func (mp ModulesParams) getNotifiers(names []string) ([]polochon.Notifier, error) {
	modules, err := mp.getModules(polochon.TypeNotifier, names...)
	if err != nil {
		return nil, err
	}

	res := []polochon.Notifier{}
	for _, m := range modules {
		res = append(res, m.(polochon.Notifier))
	}
	return res, nil
}

func (mp ModulesParams) getTorrenters(names []string) ([]polochon.Torrenter, error) {
	if len(names) == 0 {
		return nil, ErrMissingTorrenterNames
	}

	modules, err := mp.getModules(polochon.TypeTorrenter, names...)
	if err != nil {
		return nil, err
	}

	torrenters := []polochon.Torrenter{}
	for _, m := range modules {
		torrenters = append(torrenters, m.(polochon.Torrenter))
	}
	return torrenters, nil
}

func (mp ModulesParams) getFsNotifier(name string) (polochon.FsNotifier, error) {
	module, err := mp.getModule(polochon.TypeFsNotifier, name)
	if err != nil {
		return nil, err
	}

	return module.(polochon.FsNotifier), nil
}

func (mp ModulesParams) getGuessers(names []string) ([]polochon.Guesser, error) {
	modules, err := mp.getModules(polochon.TypeGuesser, names...)
	if err != nil {
		return nil, err
	}

	res := []polochon.Guesser{}
	for _, m := range modules {
		res = append(res, m.(polochon.Guesser))
	}
	return res, nil
}

func (mp ModulesParams) getDownloader(name string) (polochon.Downloader, error) {
	module, err := mp.getModule(polochon.TypeDownloader, name)
	if err != nil {
		return nil, err
	}

	return module.(polochon.Downloader), nil
}

func (mp ModulesParams) getCalendar(name string) (polochon.Calendar, error) {
	module, err := mp.getModule(polochon.TypeCalendar, name)
	if err != nil {
		return nil, err
	}

	return module.(polochon.Calendar), nil
}
