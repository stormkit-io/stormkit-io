package utils_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type IntSuite struct {
	suite.Suite
}

func (s *IntSuite) Test_GetInt() {
	s.Equal(1, utils.GetInt(1, 0, 2, 3))
	s.Equal(2, utils.GetInt(0, 0, 2, 3))
	s.Equal(3, utils.GetInt(0, 0, 0, 3))
}

func TestInt(t *testing.T) {
	suite.Run(t, &IntSuite{})
}
