package libimdb

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func testModule(t testing.TB, path string) *Catalog {
	t.Helper()
	module := &Catalog{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := module.InitWithParams(&Params{DBPath: path}); err != nil {
		t.Fatal(err)
	}
	return module
}

func TestModuleRegistration(t *testing.T) {
	for _, kind := range []polochon.ModuleType{polochon.TypeDetailer, polochon.TypeSearcher} {
		module, err := polochon.GetModule("libimdb", kind)
		if err != nil || module.Name() != "libimdb" {
			t.Fatalf("registration: %v %v", module, err)
		}
		if !module.(polochon.ContinuingDetailer).ContinueOnSuccess() {
			t.Fatal("catalog must continue detailing")
		}
	}
}

func TestInit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params *Params
		want   error
	}{
		{"nil", nil, ErrMissingDBPath},
		{"empty", &Params{}, ErrMissingDBPath},
		{"nul", &Params{DBPath: "bad\x00path"}, ErrInvalidDBPath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := (&Catalog{}).InitWithParams(tc.params); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	path := filepath.Join(t.TempDir(), "missing.sqlite")
	module := &Catalog{}
	if err := module.Init([]byte("db_path: '"+path+"'\n"), nil); err != nil {
		t.Fatal(err)
	}
	if err := module.Init(nil, nil); err != nil || module.dbPath != path {
		t.Fatalf("repeated initialization changed params: %v %q", err, module.dbPath)
	}
	if err := module.InitWithParams(&Params{DBPath: "different"}); err != nil || module.dbPath != path {
		t.Fatal("repeated parameter initialization changed params")
	}
	if err := (&Catalog{}).Init([]byte("db_path: ["), nil); err == nil {
		t.Fatal("accepted malformed YAML")
	}
	if status, err := module.Status(); status != polochon.StatusFail || err == nil {
		t.Fatalf("missing database reported healthy: %s %v", status, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only module created database: %v", err)
	}
	if _, err := (&Catalog{}).Status(); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unconfigured status: %v", err)
	}
}

func TestInvalidCatalog(t *testing.T) {
	foreign := filepath.Join(t.TempDir(), "foreign.sqlite")
	if err := os.WriteFile(foreign, []byte("not a catalog"), 0o600); err != nil {
		t.Fatal(err)
	}
	if status, err := testModule(t, foreign).Status(); status != polochon.StatusFail || err == nil {
		t.Fatalf("foreign database: %s %v", status, err)
	}
}
