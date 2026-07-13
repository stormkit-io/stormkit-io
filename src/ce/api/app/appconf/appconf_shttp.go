package appconf

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// RequestContext is the argument that the AppConfHandlers receive.
type RequestContext struct {
	*shttp.RequestContext

	User         *user.User
	App          *app.App
	EnvName      string
	DomainName   string
	DisplayName  string
	DeploymentID types.ID
}

func getDevEndpoint() string {
	pieces := strings.Split(admin.MustConfig().DomainConfig.Dev, "//")

	if len(pieces) <= 1 {
		return ""
	}

	return pieces[1]
}

func ParseHost(host string) *RequestContext {
	context := &RequestContext{DomainName: host}
	devHost := fmt.Sprintf(".%s", getDevEndpoint())

	// Custom domain
	if !strings.HasSuffix(host, devHost) {
		return context
	}

	subdomain := strings.TrimSuffix(host, devHost)

	if strings.Contains(subdomain, "--") {
		pieces := strings.Split(subdomain, "--")
		context.DisplayName = pieces[0]

		if id, err := strconv.ParseInt(pieces[1], 10, 64); err == nil {
			context.DeploymentID = types.ID(id)
			context.DisplayName = pieces[0]
		} else {
			context.EnvName = pieces[1]
		}
	} else {
		context.DisplayName = subdomain
	}

	return context
}

// isStormkitDev checks whether the given hostname matches one of
// the following formats:
//
// <display-name>.stormkit.dev
// <display-name>--<environment-name>.stormkit.dev
// <display-name>--<deployment-id>.stormkit.dev
//
// If the Deployer.Service is set to `local`, `.stormkit` domains
// will be matched. For instance deployment--1515818.stormkit
func IsStormkitDev(hostName string) bool {
	devHost := getDevEndpoint()

	if devHost == "" {
		return false
	}

	return strings.HasSuffix(hostName, fmt.Sprintf(".%s", devHost))
}

// Checks if the hostName is equal to stormkit.dev or the configured local dns.
func IsStormkitDevStrict(hostName string) bool {
	pieces := strings.Split(admin.MustConfig().DomainConfig.Dev, "//")

	if len(pieces) == 0 {
		return false
	}

	return hostName == pieces[1]
}

func FetchConfig(hostName string) ([]*Config, error) {
	reqContext := ParseHost(hostName)

	// We may still receive deployment ids such as: stormkit--592128846360
	// If that's the case, do not bother with checking the db, simply return nil
	if reqContext.DeploymentID > math.MaxInt32 {
		return nil, nil
	}

	return NewStore().Configs(context.TODO(), ConfigFilters{
		EnvName:      reqContext.EnvName,
		HostName:     reqContext.DomainName,
		DisplayName:  reqContext.DisplayName,
		DeploymentID: reqContext.DeploymentID,
	})
}
