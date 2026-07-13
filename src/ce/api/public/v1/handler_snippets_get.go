package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerSnippetsGet(req *RequestContext) *shttp.Response {
	return snippetshandlers.HandlerSnippetsGet(req.asAppContext())
}
