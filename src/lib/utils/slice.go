package utils

import "strings"

// InSliceStringCS is a helper function to check whether the
// given slice string contains the given string value. This function
// is case sensitive.
func InSliceStringCS(slice []string, value string) bool {
	if slice == nil {
		return false
	}

	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}

// InSliceString is a helper function to check whether the
// given slice string contains the given string value. This
// function is case insensitive.
func InSliceString(slice []string, value string) bool {
	if slice == nil {
		return false
	}

	for _, v := range slice {
		if strings.EqualFold(v, value) {
			return true
		}
	}

	return false
}
