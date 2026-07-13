package app

// Settings represents any setting of the application, that is crucial
// for the settings page, but not for the rest of the application.
type Settings struct {
	// DeployTrigger is the link to trigger a deployment.
	DeployTrigger string `json:"deployTrigger,omitempty"`

	// Runtime defines the default runtime version of the application.
	Runtime string `json:"runtime"`

	// Envs represent an array of environment names available
	// for the application.
	Envs []string `json:"envs,omitempty"`
}
