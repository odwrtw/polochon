package configuration

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Fake is a structure which implements a shitload of stuff
type Fake struct{}

func (f *Fake) Name() string {
	return "fake"
}

func (f *Fake) Init([]byte) error {
	return nil
}

func (f *Fake) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusOK, nil
}

func (f *Fake) GetTorrents(interface{}, *logrus.Entry) error {
	return nil
}

func (f *Fake) SearchTorrents(string) ([]*polochon.Torrent, error) {
	return nil, nil
}

func (f *Fake) GetDetails(interface{}, *logrus.Entry) error {
	return nil
}

func (f *Fake) GetSubtitle(interface{}, polochon.Language, *logrus.Entry) (polochon.Subtitle, error) {
	return nil, nil
}

func (f *Fake) AvailableMovieOptions() []string {
	return nil
}

func (f *Fake) GetMovieList(option string, log *logrus.Entry) ([]*polochon.Movie, error) {
	return nil, nil
}

func (f *Fake) AvailableShowOptions() []string {
	return nil
}

func (f *Fake) GetShowList(option string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, nil
}

func (f *Fake) SearchMovie(key string, log *logrus.Entry) ([]*polochon.Movie, error) {
	return nil, nil
}

func (f *Fake) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, nil
}

func TestModulesStatus(t *testing.T) {
	fake := &Fake{}
	c := Config{
		Movie: polochon.MovieConfig{
			Torrenters: []polochon.Torrenter{fake},
			Detailers:  []polochon.Detailer{fake},
			Subtitlers: []polochon.Subtitler{fake},
			Explorers:  []polochon.Explorer{fake},
			Searchers:  []polochon.Searcher{fake},
		},
		Show: polochon.ShowConfig{
			Torrenters: []polochon.Torrenter{fake},
		},
	}
	modulesStatus := c.ModulesStatus()
	expectedModulesStatus := ModulesStatuses{
		"movie": {
			"searcher": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"detailer": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"explorer": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"torrenter": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"subtitler": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
		},
		"show": {
			"torrenter": []ModuleStatus{
				{
					Name:   "fake",
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
