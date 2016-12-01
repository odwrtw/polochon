package index

import "sort"

// tool to extract the string keys of the map
func extractAndSortStringMapKeys(input map[string]string) []string {
	// Prepare the return slice
	ret := make([]string, len(input))

	i := 0
	for k := range input {
		ret[i] = k
		i++
	}

	// Sort the result for a deterministic result
	sort.Strings(ret)

	return ret
}

// tool to extract the int keys of the map
func extractAndSortIntMapKeys(input map[int]string) []int {
	// Prepare the return slice
	ret := make([]int, len(input))

	i := 0
	for k := range input {
		ret[i] = k
		i++
	}

	// Sort the result for a deterministic result
	sort.Ints(ret)

	return ret
}

// tool to extract the indexed seasons keys of the map
func extractAndSortIndexedSeasonsMapKeys(input map[int]IndexedSeason) []int {
	// Prepare the return slice
	ret := make([]int, len(input))

	i := 0
	for k := range input {
		ret[i] = k
		i++
	}

	// Sort the result for a deterministic result
	sort.Ints(ret)

	return ret
}