package sys_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stretchr/testify/suite"
)

type SysSuite struct {
	suite.Suite
}

func (s *SysSuite) Test_Command_Success() {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "echo",
		Args: []string{"hello"},
	})

	output, err := cmd.Output()
	s.NoError(err)
	s.Equal("hello\n", string(output))
}

func (s *SysSuite) Test_Command_Parse() {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		String: "npm run test --print 'hello world'",
	}).(sys.CommandWrapper)

	s.Equal("npm", cmd.Name())
	s.Equal([]string{"run", "test", "--print", "hello world"}, cmd.Args())
}

func Test_SysSuite(t *testing.T) {
	suite.Run(t, new(SysSuite))
}
