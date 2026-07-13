package integrations_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stretchr/testify/suite"
)

type AwsArnSuite struct {
	suite.Suite
}

func (s *AwsArnSuite) SetupSuite() {
	setAwsEnvVars()
}

func (s *AwsArnSuite) Test_BuildArn() {
	s.Equal(
		"arn:aws:lambda:eu-central-1:123456789:function:2691-4818-api",
		integrations.BuildFunctionARN(integrations.BuildFunctionARNProps{
			FuncType: integrations.FuncTypeAPI,
			EnvID:    4818,
			AppID:    2691,
		}),
	)

	s.Equal(
		"arn:aws:lambda:eu-central-1:123456789:function:2691-4818",
		integrations.BuildFunctionARN(integrations.BuildFunctionARNProps{
			FuncType: integrations.FuncTypeRenderer,
			EnvID:    4818,
			AppID:    2691,
		}),
	)
}

func TestAwsArn(t *testing.T) {
	suite.Run(t, &AwsArnSuite{})
}
