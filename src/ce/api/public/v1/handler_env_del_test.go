package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type HandlerEnvDelSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvDelSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvDelSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvDelSuite) Test_Success() {
	now := time.Now()
	usr := s.MockUser()
	app := s.MockApp(usr)
	key := s.MockAPIKey(nil, nil, map[string]any{
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"UserID": types.ID(0),
		"AppID":  app.ID,
	})
	env := s.MockEnv(app, map[string]any{
		"Name": "development",
	})

	// Required for audits
	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/v1/env?envId=%s", env.ID),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	conf, err := buildconf.NewStore().Environment(context.Background(), 2, "development")

	s.NoError(err)
	s.Nil(conf)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		AppID: app.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.GreaterOrEqual(now.Unix()+1000, audits[0].Timestamp.Unix())
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Action:      "DELETE:ENV",
		UserDisplay: "",
		AppID:       app.ID,
		EnvID:       env.ID,
		TeamID:      app.TeamID,
		TokenName:   key.Name,
		Timestamp:   audits[0].Timestamp,
		EnvName:     env.Name,
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				EnvName: env.Name,
			},
		},
	}, audits[0])
}

func TestHandlerEnvDelete(t *testing.T) {
	suite.Run(t, &HandlerEnvDelSuite{})
}
