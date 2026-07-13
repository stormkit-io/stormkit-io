package audithandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit/audithandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuditSuite struct {
	suite.Suite
	*factory.Factory
	conn      databasetest.TestDB
	user      *factory.MockUser
	app       *factory.MockApp
	auditLogs []*audit.Audit
}

func (s *HandlerAuditSuite) SetupSuite() {
	s.conn = databasetest.InitTx("handler_audit_suite")
	s.Factory = factory.New(s.conn)
	s.user = s.MockUser()
	s.app = s.MockApp(s.user)
	s.auditLogs = []*audit.Audit{
		{
			Action: "UPDATE:APP",
			UserID: s.user.ID,
			AppID:  s.app.ID,
			TeamID: s.app.TeamID,
			Diff: &audit.Diff{
				New: audit.DiffFields{
					AppName: "new-app",
					AppRepo: "github/stormkit-io/new-repo",
				},
				Old: audit.DiffFields{
					AppName: "old-app",
					AppRepo: "github/stormkit-io/old-repo",
				},
			},
		},
		{
			Action: "UPDATE:APP",
			UserID: s.user.ID,
			AppID:  s.app.ID,
			TeamID: s.app.TeamID,
			Diff: &audit.Diff{
				New: audit.DiffFields{
					AppName: "new-app-2",
					AppRepo: "github/stormkit-io/new-repo-2",
				},
				Old: audit.DiffFields{
					AppName: "old-app-2",
					AppRepo: "github/stormkit-io/old-repo-2",
				},
			},
		},
	}

	for _, log := range s.auditLogs {
		err := audit.NewStore().Log(context.Background(), log)

		if err != nil {
			panic(fmt.Sprintf("failed to insert audit log: %v", err))
		}
	}
}

func (s *HandlerAuditSuite) BeforeTest(suiteName, _ string) {
	admin.SetMockLicense()
}

func (s *HandlerAuditSuite) AfterTest(_, _ string) {
	admin.ResetMockLicense()
}

func (s *HandlerAuditSuite) TearDownSuite() {
	s.conn.CloseTx()
}

func (s *HandlerAuditSuite) Test_Success() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(audithandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/audits?appId=%s", s.app.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	re := regexp.MustCompile(`,"timestamp":"\d+"`)

	respBody := re.ReplaceAll(response.Byte(), nil)

	expected := fmt.Sprintf(`{
		"audits": [
			{
				"id": "2",
				"teamId": "%s",
				"appId": "%s",
				"envId": "",
				"action": "UPDATE:APP",
				"userDisplay": "",
				"tokenName": "",
				"envName": "",
				"diff": {
					"old": {
						"appName": "old-app-2",
						"appRepo": "github/stormkit-io/old-repo-2"
					},
					"new": {
						"appName": "new-app-2",
						"appRepo": "github/stormkit-io/new-repo-2"
					}
				}
			},
			{
				"id": "1",
				"teamId": "%s",
				"appId": "%s",
				"envId": "",
				"action": "UPDATE:APP",
				"tokenName": "",
				"userDisplay": "",
				"envName": "",
				"diff": {
					"old": {
						"appName": "old-app",
						"appRepo": "github/stormkit-io/old-repo"
					},
					"new": {
						"appName": "new-app",
						"appRepo": "github/stormkit-io/new-repo"
					}
				}
			}
		],
		"pagination": { 
			"hasNextPage": false
		}
	}`,
		s.app.TeamID.String(),
		s.app.ID.String(),
		s.app.TeamID.String(),
		s.app.ID.String(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, string(respBody))
}

func (s *HandlerAuditSuite) Test_Success_Pagination() {
	originalLimit := audithandlers.DefaultLimit
	audithandlers.DefaultLimit = 1

	defer func() {
		audithandlers.DefaultLimit = originalLimit
	}()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(audithandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/audits?appId=%s", s.app.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	re := regexp.MustCompile(`,"timestamp":"\d+"`)

	respBody := re.ReplaceAll(response.Byte(), nil)

	expected := fmt.Sprintf(`{
		"audits": [
			%s
		],
		"pagination": { 
			"hasNextPage": true,
			"beforeId": "2"
		}
	}`,
		s.auditLogs[1].JSON(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, string(respBody))

	response = shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(audithandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/audits?appId=%s&beforeId=2", s.app.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	respBody = re.ReplaceAll(response.Byte(), nil)

	expected = fmt.Sprintf(`{
		"audits": [
			%s
		],
		"pagination": { 
			"hasNextPage": false
		}
	}`,
		s.auditLogs[0].JSON(),
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, string(respBody))
}

func TestHandlerAudit(t *testing.T) {
	suite.Run(t, &HandlerAuditSuite{})
}
