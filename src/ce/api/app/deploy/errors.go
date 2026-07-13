package deploy

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

// Http errors
var (
	ErrMissingEnv          = shttperr.New(http.StatusBadRequest, "Environment name is a required field", "missing-env")
	ErrMissingEnvID        = shttperr.New(http.StatusBadRequest, "Environment ID is a required field", "missing-env-id")
	ErrMissingDeploymentID = shttperr.New(http.StatusBadRequest, "Deployment id is a required field", "missing-id")
	ErrRequestDataMerge    = shttperr.New(http.StatusBadRequest, "Unable to merge request data.", "merge-data")
	ErrDeployServer        = shttperr.New(http.StatusServiceUnavailable, "Deploy server is not reachable", "deploy-server")
)

// Deployment errors
var (
	ErrImageBuildFailed      = "Failed creating image for container"
	ErrContainerDead         = "Docker container was killed. This could be a failure caused by excessive memory usage."
	ErrContainerStopped      = "The deployment has been stopped manually"
	ErrTimeout               = "The deployment has timed out. Each deployment has a maximum of 10 minutes time to complete."
	ErrSpinUp                = "The process has failed while spinning up your container. Please retry again or reach us out at hello@stormkit.io"
	ErrCredentials           = "There has been a problem during the authentication process. Please retry again or reach us out at hello@stormkit.io"
	ErrDeployClient          = "An error occurred while uploading files to the CDN. Please retry again or reach us out at hello@stormkit.io"
	ErrRequestEntityTooLarge = "Your deployment package is too large. It can be maximum 50MB zipped and 250MB unzipped."
)
