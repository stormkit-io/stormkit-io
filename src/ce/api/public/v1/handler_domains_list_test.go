package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerDomainsListSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerDomainsListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDomainsListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDomainsListSuite) Test_Success_WithoutFilters() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	d1 := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	d2 := &buildconf.DomainModel{
		AppID:    app.ID,
		EnvID:    env.ID,
		Name:     "my.example.org",
		Token:    null.NewString("my-token", true),
		Verified: false,
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), d1))
	s.NoError(buildconf.DomainStore().Insert(context.Background(), d2))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/domains?appId=%s&envId=%s", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"domains": [
			{ "id": "%d", "domainName": "example.org", "verified": true, "token": "", "customCert": null, "lastPing": null, "analyticsExcluded": false },
			{ "id": "%d", "domainName": "my.example.org", "verified": false, "token": "my-token", "customCert": null, "lastPing": null, "analyticsExcluded": false }
		],
		"pagination": {
			"hasNextPage": false
		}
	}`, d1.ID, d2.ID)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerDomainsListSuite) Test_Success_Pagination() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	publicapiv1.DefaultDomainsLimit = 1

	defer func() {
		publicapiv1.DefaultDomainsLimit = 100
	}()

	d1 := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	d2 := &buildconf.DomainModel{
		AppID:    app.ID,
		EnvID:    env.ID,
		Name:     "my.example.org",
		Token:    null.NewString("my-token", true),
		Verified: false,
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), d1))
	s.NoError(buildconf.DomainStore().Insert(context.Background(), d2))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/domains?appId=%s&envId=%s", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"domains": [
			{ "id": "%d", "domainName": "example.org", "verified": true, "token": "", "customCert": null, "lastPing": null, "analyticsExcluded": false }
		],
		"pagination": {
			"hasNextPage": true,
			"afterId": "%d"
		}
	}`, d1.ID, d1.ID)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())

	response = shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/v1/domains?appId=%s&envId=%s&afterId=%d",
			app.ID.String(),
			env.ID.String(),
			d1.ID,
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected = fmt.Sprintf(`{
		"domains": [
			{ "id": "%d", "domainName": "my.example.org", "verified": false, "token": "my-token", "customCert": null, "lastPing": null, "analyticsExcluded": false }
		],
		"pagination": {
			"hasNextPage": false
		}
	}`, d2.ID)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerDomainsListSuite) Test_Success_WithFilters() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	d1 := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	d2 := &buildconf.DomainModel{
		AppID:    app.ID,
		EnvID:    env.ID,
		Name:     "my.example.org",
		Token:    null.NewString("my-token", true),
		Verified: false,
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), d1))
	s.NoError(buildconf.DomainStore().Insert(context.Background(), d2))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/domains?appId=%s&envId=%s&verified=true", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"domains": [
			{ "id": "%d", "domainName": "example.org", "verified": true, "token": "", "customCert": null, "lastPing": null, "analyticsExcluded": false }
		],
		"pagination": {
			"hasNextPage": false
		}
	}`, d1.ID)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerDomainsListSuite) Test_Success_WithFilters_DomainName() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	d1 := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	d2 := &buildconf.DomainModel{
		AppID:    app.ID,
		EnvID:    env.ID,
		Name:     "my.other.org",
		Token:    null.NewString("my-token", true),
		Verified: false,
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), d1))
	s.NoError(buildconf.DomainStore().Insert(context.Background(), d2))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/domains?appId=%s&envId=%s&domainName=xAMp", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"domains": [
			{ "id": "%d", "domainName": "example.org", "verified": true, "token": "", "customCert": null, "lastPing": null, "analyticsExcluded": false }
		],
		"pagination": {
			"hasNextPage": false
		}
	}`, d1.ID)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())

	response = shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/domains?appId=%s&envId=%s&domainName=hello", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "domains": [], "pagination": { "hasNextPage": false }}`, response.String())
}

func TestHandlerDomainsList(t *testing.T) {
	suite.Run(t, &HandlerDomainsListSuite{})
}
