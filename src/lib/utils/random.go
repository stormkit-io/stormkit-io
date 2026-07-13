package utils

import (
	"math/rand"
)

// Random generates a random number between the given range.
func Random(min, max int) int {
	return rand.Intn(max-min) + min
}
