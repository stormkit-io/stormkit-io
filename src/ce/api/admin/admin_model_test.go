package admin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type AdminModelSuite struct {
	suite.Suite
	mockService *mocks.MicroServiceInterface
	mockMise    *mocks.MiseInterface
	mockCommand *mocks.CommandInterface
}

func (s *AdminModelSuite) BeforeTest(_, _ string) {
	utils.SetAppKey([]byte(utils.RandomToken(32)))
	s.mockMise = &mocks.MiseInterface{}
	s.mockCommand = &mocks.CommandInterface{}
	s.mockService = &mocks.MicroServiceInterface{}
	mise.DefaultMise = s.mockMise
	sys.DefaultCommand = s.mockCommand
	rediscache.DefaultService = s.mockService
}

func (s *AdminModelSuite) AfterTest(_, _ string) {
	mise.DefaultMise = nil
	sys.DefaultCommand = nil
	rediscache.DefaultService = nil
	admin.ResetMockLicense()
}

func (s *AdminModelSuite) Test_InstanceConfig_Scan() {
	vc := admin.InstanceConfig{}

	data := map[string]any{
		"volumes": map[string]string{
			"accessKey": utils.EncryptToString("hello"),
			"secretKey": utils.EncryptToString("world"),
		},
	}

	dataBytes, err := json.Marshal(data)

	s.NoError(err)
	s.NoError(vc.Scan(dataBytes))

	s.Equal("hello", vc.VolumesConfig.AccessKey)
	s.Equal("world", vc.VolumesConfig.SecretKey)
}

func (s *AdminModelSuite) Test_InstanceConfig_Value() {
	vc := admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			AccessKey: "hello",
			SecretKey: "world",
		},
	}

	val, err := vc.Value()
	s.NoError(err)

	data := map[string]map[string]string{}
	s.NoError(json.Unmarshal(val.([]byte), &data))
	s.Len(data["volumes"]["accessKey"], 44)
	s.Len(data["volumes"]["secretKey"], 44)
}

func (s *AdminModelSuite) Test_InstallDependencies() {
	ctx := context.Background()
	vc := admin.InstanceConfig{
		SystemConfig: &admin.SystemConfig{
			Runtimes: []string{"go@1.24", "node@22"},
		},
	}

	s.mockService.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()
	s.mockService.On("Key", rediscache.KEY_RUNTIMES_STATUS).Return("test-key").Once()

	s.NoError(admin.Store().UpsertConfig(ctx, vc))

	s.mockMise.On("InstallMise", ctx).Return(nil).Once()
	s.mockMise.On("Prune", ctx).Return(nil).Once()

	for _, runtime := range vc.SystemConfig.Runtimes {
		s.mockMise.On("InstallGlobal", ctx, runtime).Return(fmt.Sprintf("runtime installed: %s", runtime), nil).Once()
	}

	admin.InstallDependencies(ctx)
}

func (s *AdminModelSuite) Test_InstallDependencies_WithBackwardsCompatibility() {
	// This should trigger the backwards compatibility code
	os.Setenv("NODE_VERSION", "18")
	defer os.Unsetenv("NODE_VERSION")

	ctx := context.Background()
	vc := admin.InstanceConfig{
		SystemConfig: &admin.SystemConfig{
			Runtimes: []string{"go@1.24"},
		},
	}

	s.mockService.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()
	s.mockService.On("Key", rediscache.KEY_RUNTIMES_STATUS).Return("test-key-backwards").Once()

	s.NoError(admin.Store().UpsertConfig(ctx, vc))

	runtimes := []string{"go@1.24", "node@18", "yarn@1.22", "pnpm@latest"}

	s.mockMise.On("InstallMise", ctx).Return(nil).Once()
	s.mockMise.On("Prune", ctx).Return(nil).Once()

	for _, runtime := range runtimes {
		s.mockMise.On("InstallGlobal", ctx, runtime).Return(fmt.Sprintf("runtime installed: %s", runtime), nil).Once()
	}

	admin.InstallDependencies(ctx)
}

func (s *AdminModelSuite) Test_Store() {
	ctx := context.Background()
	vc := admin.InstanceConfig{
		SystemConfig: &admin.SystemConfig{
			Runtimes: []string{},
		},
		ProxyConfig: &admin.ProxyConfig{
			Rules: map[string]*admin.ProxyRule{
				"example.com": {
					Target: "app.example.com",
					Headers: map[string]string{
						"X-Forwarded-Host":  "example.com",
						"X-Forwarded-Proto": "https",
					},
				},
			},
		},
	}

	s.mockService.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Twice()

	s.NoError(admin.Store().UpsertConfig(ctx, vc))

	cnf, err := admin.Store().Config(ctx)
	s.NoError(err)
	s.Equal(vc.ProxyConfig.Rules["example.com"].Target, cnf.ProxyConfig.Rules["example.com"].Target)
	s.Equal(vc.ProxyConfig.Rules["example.com"].Headers, cnf.ProxyConfig.Rules["example.com"].Headers)
}

func (s *AdminModelSuite) Test_SignUpMode() {
	vc := admin.InstanceConfig{}
	s.Equal(admin.SIGNUP_MODE_ON, vc.SignUpMode())

	vc.AuthConfig = &admin.AuthConfig{
		UserManagement: admin.UserManagement{
			SignUpMode: admin.SIGNUP_MODE_OFF,
		},
	}

	s.Equal(admin.SIGNUP_MODE_OFF, vc.SignUpMode())

	vc.AuthConfig = &admin.AuthConfig{
		UserManagement: admin.UserManagement{
			SignUpMode: admin.SIGNUP_MODE_WAITLIST,
		},
	}

	s.Equal(admin.SIGNUP_MODE_WAITLIST, vc.SignUpMode())
}

func (s *AdminModelSuite) Test_SignUpMode_CE() {
	config.SetIsSelfHosted(true)
	defer config.SetIsSelfHosted(false)

	vc := admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_WAITLIST,
			},
		},
	}

	// We fallback to OFF because waitlist is an enterprise feature.
	// If the license expires, we want to ensure the instance does not allow new sign ups
	// unless explicitly set to ON.
	s.Equal(admin.SIGNUP_MODE_OFF, vc.SignUpMode())
}

func (s *AdminModelSuite) Test_IsUserWhitelisted() {
	// Sign up mode waitlist
	vc := admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_WAITLIST,
				Whitelist:  []string{"example.com"},
			},
		},
	}

	s.True(vc.IsUserWhitelisted("user@example.com"))
	s.False(vc.IsUserWhitelisted("user@notwhitelisted.com"))

	// Sign up mode off
	vc = admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_OFF,
			},
		},
	}

	s.False(vc.IsUserWhitelisted("user@example.com"))
	s.False(vc.IsUserWhitelisted("user@notwhitelisted.com"))

	// Sign up mode on
	vc = admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_ON,
			},
		},
	}

	s.True(vc.IsUserWhitelisted("user@EXAMPLE.com"))
	s.True(vc.IsUserWhitelisted("user@notwhitelisted.com"))
}

func (s *AdminModelSuite) Test_IsUserWhitelisted_DenialMode() {
	// Sign up mode waitlist
	vc := admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{
				SignUpMode: admin.SIGNUP_MODE_WAITLIST,
				Whitelist:  []string{"!example.com", "!test.com"},
			},
		},
	}

	s.False(vc.IsUserWhitelisted("user@example.com"))
	s.False(vc.IsUserWhitelisted("user@TEST.com"))
	s.True(vc.IsUserWhitelisted("user@notblacklisted.com"))

}

func TestAdminModel(t *testing.T) {
	suite.Run(t, &AdminModelSuite{})
}
