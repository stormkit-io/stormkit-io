package volumeshandlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var allowed = []string{volumes.AWSS3, volumes.FileSys}

func handlerVolumesConfigSet(req *user.RequestContext) *shttp.Response {
	vcfg := &admin.VolumesConfig{}

	if err := req.Post(vcfg); err != nil {
		return shttp.Error(err)
	}

	if !utils.InSliceString(allowed, vcfg.MountType) {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": fmt.Sprintf("Invalid mount type given. Valid values are: %s", strings.Join(allowed, ", ")),
			},
		}
	}

	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	cfg.VolumesConfig = vcfg

	if err := admin.Store().UpsertConfig(req.Context(), cfg); err != nil {
		return shttp.Error(err)
	}

	// Reset cache
	volumes.CachedAWS = nil
	volumes.CachedFileSys = nil

	return shttp.OK()
}
