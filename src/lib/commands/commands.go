package commands

import "strings"

const ActionDeploy string = "DEPLOY"

var actionsMap = map[string]string{
	"deploy": ActionDeploy,
}

type CommandOutput struct {
	Action    string
	Arguments []string
	Flags     map[string]string
}

// Parse the given command and return a structured command output. If no command
// is found, or the action is not supported it returns nil.
func Parse(command string) *CommandOutput {
	if !strings.HasPrefix(command, "/stormkit") && !strings.HasPrefix(command, "@stormkit-io") {
		return nil
	}

	pieces := strings.Split(command, " ")
	action := actionsMap[strings.ToLower(pieces[1])]

	if action == "" {
		return nil
	}

	output := &CommandOutput{
		Action:    action,
		Arguments: []string{},
		Flags:     map[string]string{},
	}

	for _, piece := range pieces[2:] {
		if strings.HasPrefix(piece, "--") {
			flag := strings.Split(piece[2:], "=")
			key := strings.TrimSpace(strings.ToLower(flag[0]))
			val := "true"

			if len(flag) > 1 {
				val = flag[1]
			}

			output.Flags[key] = val
		} else {
			output.Arguments = append(output.Arguments, piece)
		}
	}

	return output
}
