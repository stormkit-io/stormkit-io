package main

import (
	_ "embed"
	"flag"
	"os"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// These are flags
var (
	payload string
	rootDir string
)

func mockValues() *deployservice.DeploymentMessage {
	return &deployservice.DeploymentMessage{
		Client: deployservice.ClientConfig{
			Repo: "https://github.com/stormkit-io/sample-project",
		},
		Build: deployservice.BuildConfig{
			Branch:       "main",
			DeploymentID: "some-id",
			HeadersFile:  "headers",
			Vars: map[string]string{
				"STORMKIT": "true",
				"SECRET":   "my-secret",
				"APP_KEY":  "another-secret",
				"PASSWORD": "top-secret",
				"MY_TOKEN": "another one",
				"NODE_ENV": "production",
			},
		},
	}
}

// Entry point for the runner.
//
// This program relies on several commands in the environment.
// - git
// - sh
// - node
// - bun
// - npm
// - pnpm
// - yarn
// - tar
// - zip
func main() {
	flag.StringVar(&payload, "payload", "", "Payload object with necessary information on the deployment")
	flag.StringVar(&rootDir, "root-dir", "", "Root directory for the runner")
	flag.Parse()

	// Load config
	config.Get()

	// This is a test deployment in this case
	if payload == "" {
		runner.FlagPrintLogs = true
		msg, err := mockValues().Encrypt()

		if err != nil {
			panic(err)
		}

		payload = msg

		os.Setenv("STORMKIT_APP_SECRET", utils.RandomToken(32))
	}

	if err := runner.Start(payload, rootDir); err != nil {
		panic(err)
	}
}
