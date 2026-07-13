package deployservice

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

var (
	ErrPaymentRequired = shttperr.New(http.StatusPaymentRequired, "You have reached your deployment limit. Please upgrade your package to continue.", "deployment-limit-reached")
)
