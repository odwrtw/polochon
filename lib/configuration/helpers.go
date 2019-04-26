package configuration

import (
	"fmt"
	"path/filepath"

	polochon "github.com/odwrtw/polochon/lib"
)

func evalSymlink(dest *string, path string) error {
	var err error
	*dest, err = filepath.EvalSymlinks(path)
	return err
}

func checkQuality(qualities []polochon.Quality) error {
	for _, quality := range qualities {
		if !quality.IsAllowed() {
			return fmt.Errorf("configuration: invalid quality %q", quality)
		}
	}
	return nil
}
