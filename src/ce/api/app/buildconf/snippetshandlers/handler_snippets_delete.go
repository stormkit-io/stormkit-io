package snippetshandlers

import (
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func HandlerSnippetsDelete(req *app.RequestContext) *shttp.Response {
	ids := strings.Split(req.Query().Get("ids"), ",")

	if req.Query().Get("id") != "" {
		ids = []string{req.Query().Get("id")}
	}

	if len(ids) == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"Nothing to delete."}})
	}

	counter := 0
	idsLen := len(ids)
	delete := make([]types.ID, idsLen)
	deleteStr := make([]string, idsLen)

	if idsLen > 100 {
		return shttp.BadRequest(map[string]any{"errors": []string{"Please delete maximum 100 snippets at a time."}})
	}

	for _, id := range ids {
		idInt, err := strconv.Atoi(id)

		if err != nil {
			return shttp.BadRequest(map[string]any{"errors": []string{"ID should be an integer."}})
		}

		delete[counter] = types.ID(idInt)
		deleteStr[counter] = utils.Int64ToString(int64(idInt))
		counter++
	}

	if err := buildconf.SnippetsStore().Delete(req.Context(), delete, req.EnvID); err != nil {
		return shttp.Error(err)
	}

	diff := &audit.Diff{
		Old: audit.DiffFields{
			Snippets: deleteStr,
		},
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeSnippet).
			WithDiff(diff).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	if err := appcache.Service().Reset(req.EnvID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
