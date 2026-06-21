package publicapiv1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerDomainAddSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerDomainAddSuite) BeforeTest(suiteName, _ string) {
	// Pin self-hosted so the Verified assertion is deterministic: other suites
	// in this package mutate the shared edition flag without resetting it.
	config.SetIsSelfHosted(true)

	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDomainAddSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsSelfHosted(false)
}

func (s *HandlerDomainAddSuite) Test_Success() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	// Required for audits
	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/domains",
		map[string]any{
			"envId":  env.ID.String(),
			"appId":  app.ID.String(),
			"domain": "example.org",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]string{}

	s.NoError(json.Unmarshal(response.Byte(), &data))
	s.Equal(http.StatusOK, response.Code)
	s.Len(data["token"], 32)

	domain, err := buildconf.DomainStore().DomainByID(context.Background(), utils.StringToID(data["domainId"]))

	s.NoError(err)
	s.Equal(data["token"], domain.Token.ValueOrZero())
	s.True(domain.Verified) // Because it's self-hosted

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "CREATE:DOMAIN",
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		TeamID:      app.TeamID,
		AppID:       app.ID,
		EnvID:       env.ID,
		EnvName:     env.Name,
		Diff: &audit.Diff{
			New: audit.DiffFields{
				DomainName: "example.org",
			},
		},
	}, audits[0])
}

func (s *HandlerDomainAddSuite) Test_IsValidDomain() {
	// input => expected host
	validDomains := map[string]string{
		"https://www.stormkit.io":       "www.stormkit.io",
		"http://www.stormkit.io":        "www.stormkit.io",
		"www.stormkit.io":               "www.stormkit.io",
		"stormkit.io":                   "stormkit.io",
		"stormkit.co.uk":                "stormkit.co.uk",
		"www.stormkit.co.uk":            "www.stormkit.co.uk",
		"app.stormkit.dev":              "app.stormkit.dev",
		"https://stormkit.io/with/path": "stormkit.io",
		"stormkit.io:8888":              "stormkit.io",
	}

	for input, expectedHost := range validDomains {
		parsed := publicapiv1.IsValidDomain(input)
		s.NotNil(parsed)
		s.Equal(expectedHost, parsed.Hostname())
	}
	// input => expected host
	invalidDomains := []string{
		"http://www",
		"invalid",
	}

	for _, input := range invalidDomains {
		s.Nil(publicapiv1.IsValidDomain(input))
	}
}

func (s *HandlerDomainAddSuite) Test_Duplicate_AlreadyVerified() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	s.NoError(buildconf.DomainStore().Insert(context.Background(), &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/domains",
		map[string]any{
			"envId":  env.ID.String(),
			"appId":  app.ID.String(),
			"domain": "example.org",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerDomainAddSuite) Test_InvalidDomain() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/domains",
		map[string]any{
			"envId":  env.ID.String(),
			"appId":  app.ID.String(),
			"domain": "invalid-domain",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{"error":"Please provide a valid domain name."}`
	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func TestDomainSet(t *testing.T) {
	suite.Run(t, &HandlerDomainAddSuite{})
}
