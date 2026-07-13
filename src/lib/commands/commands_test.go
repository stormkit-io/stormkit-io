package commands_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/commands"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	output := commands.Parse("/stormkit deploy staging --publish")

	assert.NotNil(t, output)
	assert.Equal(t, output.Action, commands.ActionDeploy)
	assert.Equal(t, output.Flags["publish"], "true")
	assert.Equal(t, output.Flags["flag-2"], "")
	assert.Equal(t, output.Arguments, []string{"staging"})
}
