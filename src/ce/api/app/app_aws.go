package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/saws"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type FunctionConfiguration struct {
	AppID   types.ID
	EnvID   types.ID
	Runtime *string
	Vars    map[string]string
}

// UpdateFunctionConfiguration updates the function configuration for the given input.
// If the `conf` parameter includes an EnvID, this function will update the configuration
// only for that environment. If it does not contain an environment ID, it will update
// for all environments. When a variable is empty, it's value is not updated.
func UpdateFunctionConfiguration(ctx context.Context, conf FunctionConfiguration) error {
	if config.Get().AWS == nil || config.IsTest() {
		return nil
	}

	if conf.EnvID != 0 {
		return updateFunctionConfigurationForEnv(conf)
	}

	envs, err := buildconf.NewStore().ListEnvironments(ctx, conf.AppID)

	if err != nil {
		return fmt.Errorf("cannot list environments err=%v", err)
	}

	for _, env := range envs {
		err := updateFunctionConfigurationForEnv(FunctionConfiguration{
			AppID:   conf.AppID,
			EnvID:   env.ID,
			Runtime: conf.Runtime,
			Vars:    conf.Vars,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func updateFunctionConfigurationForEnv(conf FunctionConfiguration) error {
	funcTypes := []string{saws.FuncTypeAPI, saws.FuncTypeRenderer}

	var env *lambda.Environment

	if conf.Vars != nil {
		env = &lambda.Environment{
			Variables: map[string]*string{},
		}

		for key, value := range conf.Vars {
			env.Variables[key] = aws.String(value)
		}
	}

	runtime := conf.Runtime

	if runtime != nil && strings.HasPrefix(*runtime, "bun") {
		runtime = aws.String(config.NodeRuntime18)
	}

	for _, funcType := range funcTypes {
		lambdaName := saws.BuildFunctionARN(saws.BuildFunctionARNProps{
			AppID:    conf.AppID,
			EnvID:    conf.EnvID,
			FuncType: funcType,
		})

		slog.Infof("updating function configuration with env vars: %v", env != nil)

		_, err := saws.
			Instance().
			UpdateFunction(&lambda.UpdateFunctionConfigurationInput{
				FunctionName: aws.String(lambdaName),
				Runtime:      runtime,
				Environment:  env,
			})

		if err != nil {
			var rnf *lambda.ResourceNotFoundException

			if !errors.As(err, &rnf) {
				slog.Errorf("error while updating lambda runtime: name=%s err=%v", lambdaName, err)
				return err
			}
		}
	}

	return nil
}
