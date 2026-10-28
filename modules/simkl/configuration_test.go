package simkl

import (
	"fmt"
	"strings"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
)

func TestRegistration(t *testing.T) {
	for _, kind := range []polochon.ModuleType{polochon.TypeExplorer, polochon.TypeSearcher, polochon.TypeDetailer, polochon.TypeCalendar} {
		module, err := polochon.GetModule("simkl", kind)
		if err != nil || module.Name() != "simkl" {
			t.Errorf("GetModule(%s) = %v, %v", kind, module, err)
		}
	}
	if _, err := polochon.GetModule("simkl", polochon.TypeWishlister); err == nil {
		t.Error("SIMKL should not implement an authenticated wishlister")
	}
}

func TestConfiguration(t *testing.T) {
	dir := t.TempDir()
	data := fmt.Sprintf(`
movie:
  dir: %q
  explorers: [simkl]
  searchers: [simkl]
  detailers: [simkl]
show:
  dir: %q
  explorers: [simkl]
  searchers: [simkl]
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
	if len(conf.Movie.Explorers) != 1 || len(conf.Movie.Searchers) != 1 || len(conf.Movie.Detailers) != 1 || len(conf.Show.Explorers) != 1 || len(conf.Show.Searchers) != 1 || len(conf.Show.Detailers) != 1 || conf.Show.Calendar == nil {
		t.Fatalf("missing configured roles: %+v", conf)
	}
	module := conf.Movie.Explorers[0].(*Simkl)
	if module.client.clientID != "test-client" || module.client.appVersion != "test" || conf.Show.Calendar != module {
		t.Errorf("configuration did not initialize one shared module: %+v", module)
	}
}
