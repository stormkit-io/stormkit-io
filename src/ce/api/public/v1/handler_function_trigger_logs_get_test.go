package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandleTriggerLogsGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandleTriggerLogsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandleTriggerLogsGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandleTriggerLogsGetSuite) Test_Success() {
	store := functiontrigger.NewStore()
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	tf := s.MockTriggerFunction(env, map[string]any{
		"Options": functiontrigger.Options{
			Method:  "POST",
			Payload: []byte(`{ "hello": "world" }`),
		},
	})

	logs := []functiontrigger.TriggerLog{
		{
			TriggerID: tf.ID,
			Request:   map[string]any{"Key": "Value"},
			Response:  map[string]any{"Hello": "World"},
		},
	}

	s.NoError(store.InsertLogs(context.Background(), logs))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/trigger/logs?appId=%d&envId=%d&triggerId=%d", app.ID, env.ID, tf.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.Contains(str, `"request":{"Key":"Value"}`)
	s.Contains(str, `"response":{"Hello":"World"}`)
}

func (s *HandleTriggerLogsGetSuite) Test_MasksHeaders() {
	store := functiontrigger.NewStore()
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tf := s.MockTriggerFunction(env)

	logs := []functiontrigger.TriggerLog{
		{
			TriggerID: tf.ID,
			Request: map[string]any{
				"url":     "https://example.org",
				"headers": map[string]any{"Authorization": "Bearer secret-token"},
			},
			Response: map[string]any{"code": float64(200)},
		},
	}

	s.NoError(store.InsertLogs(context.Background(), logs))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/trigger/logs?appId=%d&envId=%d&triggerId=%d", app.ID, env.ID, tf.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(str, "secret-token")
	s.Contains(str, `"Authorization":""`)
}

func TestHandlerTriggerLogsGet(t *testing.T) {
	suite.Run(t, &HandleTriggerLogsGetSuite{})
}
