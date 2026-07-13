package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func TestInSliceString(t *testing.T) {
	s := []string{"my", "awesome", "string", "slice"}

	assert.Equal(t, utils.InSliceString(s, "my"), true)
	assert.Equal(t, utils.InSliceString(s, "awesome"), true)
	assert.Equal(t, utils.InSliceString(s, "slice"), true)
	assert.Equal(t, utils.InSliceString(s, "string"), true)
	assert.Equal(t, utils.InSliceString(s, "stringg"), false)
	assert.Equal(t, utils.InSliceString(nil, "value"), false)
}
