package integrations

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type BuildFunctionARNProps struct {
	FuncType string
	EnvID    types.ID
	AppID    types.ID
}

func BuildFunctionARN(args BuildFunctionARNProps) string {
	lambdaName := args.AppID.String()

	if args.EnvID != 0 {
		obfEnvId := args.EnvID.String()

		// Fixes a problem that results in duplicate function names
		if !strings.Contains(lambdaName, fmt.Sprintf("-%s", obfEnvId)) {
			lambdaName = fmt.Sprintf(
				"%s-%s",
				lambdaName,
				obfEnvId,
			)
		}
	}

	if args.FuncType != "" && args.FuncType != FuncTypeRenderer {
		lambdaName = fmt.Sprintf("%s-%s", lambdaName, args.FuncType)
	}

	cnf := config.Get()

	return fmt.Sprintf(
		`arn:aws:lambda:%s:%s:function:%s`,
		cnf.AWS.Region,
		cnf.AWS.AccountID,
		lambdaName,
	)
}
