package gitlab

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	gl "github.com/xanzy/go-gitlab"
)

const testHooksURL = "https://api.example.org/app/webhooks/gitlab"

type HookManagerSuite struct {
	suite.Suite

	manager hookManager
}

func (s *HookManagerSuite) SetupSuite() {
	utils.SetAppKey([]byte("12345678901234567890123456789012"))
}

func (s *HookManagerSuite) SetupTest() {
	s.manager = hookManager{baseURL: testHooksURL, appID: types.ID(15)}
}

func (s *HookManagerSuite) Test_IsInstalled_OwnSecret() {
	hooks := []*gl.ProjectHook{
		{ID: 1, URL: testHooksURL + "/" + utils.EncryptID(types.ID(15))},
	}

	s.True(s.manager.isInstalled(hooks))
}

// Test_IsInstalled_OtherAppSecret verifies that a hook of another app on the
// same repository does not count, since it only deploys that app.
func (s *HookManagerSuite) Test_IsInstalled_OtherAppSecret() {
	hooks := []*gl.ProjectHook{
		{ID: 1, URL: testHooksURL + "/" + utils.EncryptID(types.ID(16))},
	}

	s.False(s.manager.isInstalled(hooks))
}

func (s *HookManagerSuite) Test_IsInstalled_LegacyHook() {
	hooks := []*gl.ProjectHook{{ID: 1, URL: testHooksURL}}

	s.False(s.manager.isInstalled(hooks))
}

func (s *HookManagerSuite) Test_LegacyHook() {
	hooks := []*gl.ProjectHook{
		{ID: 1, URL: "https://ci.example.org/hook"},
		{ID: 2, URL: testHooksURL + "/" + utils.EncryptID(types.ID(16))},
		{ID: 3, URL: testHooksURL + "/"},
	}

	legacy := s.manager.legacyHook(hooks)

	s.Require().NotNil(legacy)
	s.Equal(3, legacy.ID)
}

func (s *HookManagerSuite) Test_LegacyHook_None() {
	hooks := []*gl.ProjectHook{
		{ID: 2, URL: testHooksURL + "/" + utils.EncryptID(types.ID(15))},
	}

	s.Nil(s.manager.legacyHook(hooks))
}

func TestHookManagerSuite(t *testing.T) {
	suite.Run(t, new(HookManagerSuite))
}
