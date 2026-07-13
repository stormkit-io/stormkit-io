package utils

import "strings"

// List of commands to exclude
var excludedCommands = map[string]bool{
	"awk":   true,
	"bash":  true,
	"cat":   true,
	"cd":    true,
	"chmod": true,
	"chown": true,
	"cp":    true,
	"echo":  true,
	"exit":  true,
	"find":  true,
	"grep":  true,
	"ls":    true,
	"mkdir": true,
	"mv":    true,
	"node":  true,
	"pwd":   true,
	"rmdir": true,
	"rm":    true,
	"sed":   true,
	"sh":    true,
	"touch": true,
}

var packageManagers = map[string]bool{
	"npm":  true,
	"pnpm": true,
	"yarn": true,
	"bun":  true,
}

type Command struct {
	IsPackageManager bool
	CommandName      string
	ScriptName       string // ScriptName is the name of the script that the command references (e.g. "start": "npm run develop")
	Arguments        []string
}

// ParseCommands takes a string of server commands and returns the command names.
func ParseCommands(serverCmd string) []Command {
	// Split the serverCmd string by "&&" to get individual commands
	commands := strings.Split(serverCmd, "&&")
	filteredCommands := []Command{}

	for _, cmd := range commands {
		// Trim whitespace from the command
		cmd = strings.TrimSpace(cmd)

		// Extract the first word (the actual command)
		parts := strings.Fields(cmd)

		if len(parts) == 0 {
			continue
		}

		commandName := strings.ToLower(parts[0])
		arguments := parts[1:]
		argsLen := len(arguments)

		// Check if the command is in the exclusion list
		if _, excluded := excludedCommands[commandName]; !excluded {
			isPackageManager := packageManagers[commandName]
			scriptName := ""

			if isPackageManager && argsLen > 0 {
				if argsLen == 1 {
					scriptName = arguments[0]
				} else if argsLen > 1 && arguments[0] == "run" {
					scriptName = arguments[1]
				}
			}

			filteredCommands = append(filteredCommands, Command{
				IsPackageManager: isPackageManager,
				CommandName:      commandName,
				ScriptName:       scriptName,
				Arguments:        parts[1:],
			})
		}
	}

	return filteredCommands
}
