package index

import "sort"

// tool to extract the keys of the map
func extractAndSortMapKeys(input map[string]string) ([]string, error) {
	// Prepare the return slice
	ret := make([]string, len(input))

	i := 0
	for k := range input {
		ret[i] = k
		i++
	}

	// Sort the result for a deterministic result
	sort.Strings(ret)

	return ret, nil
}
