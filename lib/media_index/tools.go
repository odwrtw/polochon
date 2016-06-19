package index

// tool to extract the keys of the map
func extractMapKeys(input map[string]string) ([]string, error) {
	// Prepare the return slice
	ret := make([]string, len(input))

	i := 0
	for k := range input {
		ret[i] = k
		i++
	}

	return ret, nil
}
