package snippetshandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var DefaultSnippetsLimit = 50

func HandlerSnippetsGet(req *app.RequestContext) *shttp.Response {
	hosts := []string{}

	for _, v := range strings.Split(req.Query().Get("hosts"), ",") {
		if v != "" {
			hosts = append(hosts, strings.ToLower(strings.TrimSpace(v)))
		}
	}

	filters := buildconf.SnippetFilters{
		EnvID:   req.EnvID,
		Hosts:   hosts,
		Limit:   DefaultSnippetsLimit,
		Title:   req.Query().Get("title"),
		AfterID: utils.StringToID(req.Query().Get("afterId")),
	}

	snippets, err := buildconf.SnippetsStore().SnippetsByEnvID(req.Context(), filters)

	if err != nil {
		return shttp.Error(err)
	}

	snippetsLen := len(snippets)
	pagination := map[string]any{
		"hasNextPage": false,
	}

	if snippetsLen > DefaultSnippetsLimit {
		pagination = map[string]any{
			"hasNextPage": true,
			"afterId":     snippets[snippetsLen-2].ID.String(),
		}

		snippets = snippets[:snippetsLen-1]
	}

	result := []map[string]any{}

	for _, s := range snippets {
		result = append(result, s.JSON())
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"snippets":   result,
			"pagination": pagination,
		},
	}
}
