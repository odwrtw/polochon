package configuration

import (
	"sync"

	polochon "github.com/odwrtw/polochon/lib"
)

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
type ModulesStatuses map[string]map[string][]*ModuleStatus

// ModulesStatus gives the status of the modules configured
func (c *Config) ModulesStatus() ModulesStatuses {
	mc := newModuleChecker()

	result := ModulesStatuses{}
	for _, module := range []struct {
		t  string
		mf ModuleFetcher
	}{
		{t: "movie", mf: &c.Movie},
		{t: "show", mf: &c.Show},
	} {
		result[module.t] = map[string][]*ModuleStatus{}
		for _, detailer := range module.mf.GetDetailers() {
			result[module.t]["detailer"] = append(result[module.t]["detailer"], mc.check(detailer))
		}
		for _, subtitler := range module.mf.GetSubtitlers() {
			result[module.t]["subtitler"] = append(result[module.t]["subtitler"], mc.check(subtitler))
		}
		for _, torrenter := range module.mf.GetTorrenters() {
			result[module.t]["torrenter"] = append(result[module.t]["torrenter"], mc.check(torrenter))
		}
		for _, explorer := range module.mf.GetExplorers() {
			result[module.t]["explorer"] = append(result[module.t]["explorer"], mc.check(explorer))
		}
		for _, searcher := range module.mf.GetSearchers() {
			result[module.t]["searcher"] = append(result[module.t]["searcher"], mc.check(searcher))
		}
	}

	mc.wg.Wait()
	return result
}

type moduleChecker struct {
	modules map[string]*ModuleStatus
	wg      sync.WaitGroup
}

func newModuleChecker() *moduleChecker {
	return &moduleChecker{
		modules: map[string]*ModuleStatus{},
	}
}

func (mc *moduleChecker) check(m polochon.Module) *ModuleStatus {
	s, ok := mc.modules[m.Name()]
	if ok {
		return s
	}

	moduleStatus := &ModuleStatus{Name: m.Name()}
	mc.modules[m.Name()] = moduleStatus

	mc.wg.Add(1)
	go func() {
		defer mc.wg.Done()

		status, err := m.Status()
		if err != nil {
			moduleStatus.Error = err.Error()
		}
		moduleStatus.Status = status
	}()

	return moduleStatus
}
