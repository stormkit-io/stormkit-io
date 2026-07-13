package utils_test

import (
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type CmdTestSuite struct {
	suite.Suite
}

func (s *CmdTestSuite) Test_ParseCommands() {
	// Test the command parsing logic
	cmd := strings.Join([]string{
		"echo Hello, World!",
		"npm run start",
		"pnpm start",
		"next start",
		"yarn develop",
		"remix-serve",
	}, " && ")

	expected := []utils.Command{{
		IsPackageManager: true,
		CommandName:      "npm",
		ScriptName:       "start",
		Arguments:        []string{"run", "start"},
	}, {
		IsPackageManager: true,
		CommandName:      "pnpm",
		ScriptName:       "start",
		Arguments:        []string{"start"},
	}, {
		IsPackageManager: false,
		CommandName:      "next",
		ScriptName:       "",
		Arguments:        []string{"start"},
	}, {
		IsPackageManager: true,
		CommandName:      "yarn",
		ScriptName:       "develop",
		Arguments:        []string{"develop"},
	}, {
		IsPackageManager: false,
		CommandName:      "remix-serve",
		Arguments:        []string{},
	}}

	parsedCmd := utils.ParseCommands(cmd)
	s.Equal(expected, parsedCmd)
}

func Test_CmdTestSuite(t *testing.T) {
	suite.Run(t, &CmdTestSuite{})
}
