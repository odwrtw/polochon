package simkl

import (
	"fmt"
	"strings"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
)

func TestRegistration(t *testing.T) {
	for _, kind := range []polochon.ModuleType{polochon.TypeExplorer, polochon.TypeDetailer, polochon.TypeCalendar} {
		module, err := polochon.GetModule("simkl", kind)
		if err != nil || module.Name() != "simkl" {
			t.Errorf("GetModule(%s) = %v, %v", kind, module, err)
		}
	}
	for _, kind := range []polochon.ModuleType{polochon.TypeSearcher, polochon.TypeWishlister} {
		if _, err := polochon.GetModule("simkl", kind); err == nil {
			t.Errorf("SIMKL should not implement %s", kind)
		}
	}
}

func TestConfiguration(t *testing.T) {
	dir := t.TempDir()
	data := fmt.Sprintf(`
movie:
  dir: %q
  explorers: [simkl]
  detailers: [simkl]
show:
  dir: %q
  explorers: [simkl]
  detailers: [simkl]
  calendar: simkl
modules_params:
  - name: simkl
    client_id: test-client
    app_version: test
`, dir, dir)
	conf, err := configuration.LoadConfig(strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(conf.Movie.Explorers) != 1 || len(conf.Movie.Searchers) != 0 || len(conf.Movie.Detailers) != 1 || len(conf.Show.Explorers) != 1 || len(conf.Show.Searchers) != 0 || len(conf.Show.Detailers) != 1 || conf.Show.Calendar == nil {
		t.Fatalf("missing configured roles: %+v", conf)
	}
	module := conf.Movie.Explorers[0].(*Simkl)
	if module.client.clientID != "test-client" || module.client.appVersion != "test" || conf.Show.Calendar != module {
		t.Errorf("configuration did not initialize one shared module: %+v", module)
	}
	for _, kind := range []string{"movie", "show"} {
		t.Run("reject searcher/"+kind, func(t *testing.T) {
			invalid := strings.Replace(data, kind+":\n", kind+":\n  searchers: [simkl]\n", 1)
			if _, err := configuration.LoadConfig(strings.NewReader(invalid)); err == nil {
				t.Fatal("accepted SIMKL as a searcher")
			}
		})
	}
}
