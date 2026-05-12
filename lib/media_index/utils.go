package index

import (
	"sort"

	polochon "github.com/odwrtw/polochon/lib"
)


// tool to extract the string keys of the map
func extractAndSortStringMapKeys(input map[string]*polochon.Movie) []string {
	ret := make([]string, len(input))

	i := 0
	for k := range input {
		ret[i] = k
		i++
	}

	sort.Strings(ret)

	return ret
}

