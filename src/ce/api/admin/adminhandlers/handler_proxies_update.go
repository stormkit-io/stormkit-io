package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type ProxiesUpdateRequest struct {
	Proxies map[string]*admin.ProxyRule `json:"proxies"`
	Remove  []string                    `json:"remove,omitempty"`
}

func handlerProxiesUpdate(req *user.RequestContext) *shttp.Response {
	data := ProxiesUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	vc, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	if vc.ProxyConfig == nil {
		vc.ProxyConfig = &admin.ProxyConfig{
			Rules: map[string]*admin.ProxyRule{},
		}
	}

	// Merge with existing rules
	for name, rule := range data.Proxies {
		vc.ProxyConfig.Rules[name] = rule
	}

	// Remove specified rules
	for _, name := range data.Remove {
		delete(vc.ProxyConfig.Rules, name)
	}

	if err := admin.Store().UpsertConfig(req.Context(), vc); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"proxies": vc.ProxyConfig,
		},
	}
}
