package publicapiv1_test

import (
	"context"
	"net/http"
	"strings"
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

type HandlerFunctionTriggerDescriptionSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerFunctionTriggerDescriptionSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerFunctionTriggerDescriptionSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerFunctionTriggerDescriptionSuite) create(env *factory.MockEnv, body map[string]any) shttptest.Response {
	body["appId"] = env.AppID.String()
	body["envId"] = env.ID.String()

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/trigger",
		body,
		map[string]string{
			"Authorization": usertest.Authorization(env.GetApp().UserID),
		},
	)
}

func (s *HandlerFunctionTriggerDescriptionSuite) Test_Success_Description() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := s.create(env, map[string]any{
		"cron":        "*/1 * * * *",
		"status":      true,
		"description": "Autofill weekly newsletter",
		"options": map[string]any{
			"method": "GET",
			"url":    "https://test.com/api/cron",
		},
	})

	s.Equal(http.StatusCreated, response.Code)

	tfs, err := functiontrigger.NewStore().List(context.Background(), env.ID)

	s.Require().NoError(err)
	s.Require().Len(tfs, 1)
	s.Equal("Autofill weekly newsletter", tfs[0].Description)
}

// The description is rendered inline in listings, so it is bounded and kept to
// a single line rather than becoming a second documentation field.
func (s *HandlerFunctionTriggerDescriptionSuite) Test_Fail_DescriptionTooLong() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := s.create(env, map[string]any{
		"cron":        "*/1 * * * *",
		"status":      true,
		"description": strings.Repeat("a", 201),
		"options": map[string]any{
			"method": "GET",
			"url":    "https://test.com/api/cron",
		},
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "description")
}

func (s *HandlerFunctionTriggerDescriptionSuite) Test_Fail_DescriptionMultiline() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := s.create(env, map[string]any{
		"cron":        "*/1 * * * *",
		"status":      true,
		"description": "Autofill weekly newsletter\nand mail it",
		"options": map[string]any{
			"method": "GET",
			"url":    "https://test.com/api/cron",
		},
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "description")
}

func TestHandlerFunctionTriggerDescription(t *testing.T) {
	suite.Run(t, &HandlerFunctionTriggerDescriptionSuite{})
}
