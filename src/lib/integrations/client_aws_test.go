package integrations_test

import (
	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

func setAwsEnvVars() {
	config.Get().AWS = &config.AwsConfig{
		AccountID: "123456789",
		Region:    "eu-central-1",
	}
}
