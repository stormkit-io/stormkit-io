package audithandlers

import (
	_ "embed"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/audits").
		Middleware(user.WithEE).
		Handler(shttp.MethodGet, "", user.WithAuth(handlerAudits))

	return s
}
