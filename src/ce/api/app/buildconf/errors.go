package buildconf

import (
	"errors"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

// Errors list
var (
	ErrSchemaExists      = errors.New("schema already exists")
	ErrInvalidSchemaName = errors.New("invalid schema name")

	// Legacy errors
	ErrMissingEnv             = shttperr.New(http.StatusBadRequest, "Environment is missing", "env-missing")
	ErrInvalidEnv             = shttperr.New(http.StatusBadRequest, "Environment can only contain alphanumeric characters and hypens.", "env-invalid")
	ErrInvalidEnvDoubleHypens = shttperr.New(http.StatusBadRequest, "Double hypens (--) are not allowed as they are reserved for Stormkit.", "env-invalid")
	ErrInvalidBranch          = shttperr.New(http.StatusBadRequest, "Branch name is required and can only contain following characters: alphanumeric, -, +, /, ., and =", "branch-invalid") // See https://wincent.com/wiki/Legal_Git_branch_names for more details.
	ErrDomainInvalidFormat    = shttperr.New(http.StatusBadRequest, "Domain format is not correct", "invalid-domain")
	ErrDomainInvalidToken     = shttperr.New(http.StatusBadRequest, "The verification token is not found. Please start the verification process by setting a domain first.", "invalid-token")
	ErrInvalidPercentage      = shttperr.New(http.StatusBadRequest, "The sum of percentages should be 100 in order to publish.", "invalid-percentage")
	ErrLambdaAlreadyExists    = shttperr.New(http.StatusBadRequest, "Lambda function name already exists.", "lambda-already-exists")
	ErrDuplicateEnvName       = shttperr.New(http.StatusBadRequest, "Environment name is duplicate. Choose a different name.", "duplicate-env")
)
