package utils

// GetInt returns the first non-zero value.
func GetInt(values ...int) int {
	for _, val := range values {
		if val != 0 {
			return val
		}
	}

	return 0
}
