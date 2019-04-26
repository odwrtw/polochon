package configuration

import polochon "github.com/odwrtw/polochon/lib"

// ModuleFetcher is an interface which allows to get torrenters and detailers ...
type ModuleFetcher interface {
	GetDetailers() []polochon.Detailer
	GetTorrenters() []polochon.Torrenter
	GetSearchers() []polochon.Searcher
	GetExplorers() []polochon.Explorer
	GetSubtitlers() []polochon.Subtitler
}

// ModuleStatus represent the status of a module
type ModuleStatus struct {
	Name   string                `json:"name"`
	Status polochon.ModuleStatus `json:"status"`
	Error  string                `json:"error"`
}

// ModulesStatuses represent the status of all the modules
type ModulesStatuses map[string]map[string][]ModuleStatus

// ModulesStatus gives the status of the modules configured
func (c *Config) ModulesStatus() ModulesStatuses {
	type module struct {
		moduleType string
		config     ModuleFetcher
	}

	result := ModulesStatuses{}
	for _, module := range []module{
		{
			"movie",
			&c.Movie,
		},
		{
			"show",
			&c.Show,
		},
	} {
		result[module.moduleType] = map[string][]ModuleStatus{}
		for _, detailer := range module.config.GetDetailers() {
			status, err := detailer.Status()
			result[module.moduleType]["detailer"] = append(result[module.moduleType]["detailer"], ModuleStatus{
				Name:   detailer.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
		for _, subtitler := range module.config.GetSubtitlers() {
			status, err := subtitler.Status()
			result[module.moduleType]["subtitler"] = append(result[module.moduleType]["subtitler"], ModuleStatus{
				Name:   subtitler.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
		for _, torrenter := range module.config.GetTorrenters() {
			status, err := torrenter.Status()
			result[module.moduleType]["torrenter"] = append(result[module.moduleType]["torrenter"], ModuleStatus{
				Name:   torrenter.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
		for _, explorer := range module.config.GetExplorers() {
			status, err := explorer.Status()
			result[module.moduleType]["explorer"] = append(result[module.moduleType]["explorer"], ModuleStatus{
				Name:   explorer.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
		for _, searcher := range module.config.GetSearchers() {
			status, err := searcher.Status()
			result[module.moduleType]["searcher"] = append(result[module.moduleType]["searcher"], ModuleStatus{
				Name:   searcher.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
	}

	return result
}

func errorMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
