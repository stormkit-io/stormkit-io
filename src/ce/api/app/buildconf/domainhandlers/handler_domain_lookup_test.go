package domainhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/domainhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type HandlerDomainLookupSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerDomainLookupSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDomainLookupSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

// These are taken from real data. It will fail if you change the IDs, or TXT records.
func (s *HandlerDomainLookupSuite) Test_DomainVerified() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	domain := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(domainhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/domains/lookup?appId=%s&envId=%s&domainId=%d",
			app.ID.String(),
			env.ID.String(),
			domain.ID,
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.Contains(response.String(), `"domainName":"stormkit.io"`)
}

func (s *HandlerDomainLookupSuite) Test_DomainNotVerified() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	domain := &buildconf.DomainModel{
		AppID:    app.ID,
		EnvID:    env.ID,
		Name:     "www.google.com",
		Verified: false,
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(domainhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/domains/lookup?appId=%s&envId=%s&domainId=%d",
			app.ID.String(),
			env.ID.String(),
			domain.ID,
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	expected1 := `{"tlsError":null,"tls":null,"dns":{"verified":false`
	expected2 := `"lookup":"22c6d6c1871fb4655e2b6e01aa59c8e7.www.google.com","name":"22c6d6c1871fb4655e2b6e01aa59c8e7","records":null,"value":"3745d5f22e46c8eb4a9b1400f92c8b0d"`
	resString := response.String()

	s.Contains(resString, expected1)
	s.Contains(resString, expected2)
}

func TestHandlerDomainLookup(t *testing.T) {
	suite.Run(t, &HandlerDomainLookupSuite{})
}
