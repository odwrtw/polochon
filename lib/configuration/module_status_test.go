package configuration

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/modules/mock"
)

func TestModulesStatus(t *testing.T) {
	polochon.ClearRegisteredModules()
	mock := &mock.Mock{}
	polochon.RegisterModule(mock)

	c := Config{
		Movie: polochon.MovieConfig{
			Torrenters: []polochon.Torrenter{mock},
			Detailers:  []polochon.Detailer{mock},
			Subtitlers: []polochon.Subtitler{mock},
			Explorers:  []polochon.Explorer{mock},
			Searchers:  []polochon.Searcher{mock},
		},
		Show: polochon.ShowConfig{
			Torrenters: []polochon.Torrenter{mock},
		},
	}
	modulesStatus := c.ModulesStatus()
	expectedModulesStatus := ModulesStatuses{
		"movie": {
			"searcher": []*ModuleStatus{
				{
					Name:   "mock",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"detailer": []*ModuleStatus{
				{
					Name:   "mock",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"explorer": []*ModuleStatus{
				{
					Name:   "mock",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"torrenter": []*ModuleStatus{
				{
					Name:   "mock",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"subtitler": []*ModuleStatus{
				{
					Name:   "mock",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
		},
		"show": {
			"torrenter": []*ModuleStatus{
				{
					Name:   "mock",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
		},
	}
	if !reflect.DeepEqual(modulesStatus, expectedModulesStatus) {
		t.Errorf("Didn't get expected module status \n %+v \n %+v", modulesStatus, expectedModulesStatus)
	}
}
