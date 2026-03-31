package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stretchr/testify/suite"
)

type PackageSuite struct {
	suite.Suite

	appSecret string
}

func (s *PackageSuite) BeforeTest(_, _ string) {
	s.appSecret = "gS9u8RZ*3^7^3*jRfDdnTVv9@rrqqr#5"

	os.Setenv("AWS_REGION", "eu-central-1")
	os.Setenv("STORMKIT_APP_SECRET", s.appSecret)
}

func (s *PackageSuite) AfterTest(_, _ string) {
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("STORMKIT_APP_SECRET")
	os.Unsetenv("STORMKIT_HTTP_READ_TIMEOUT")
	os.Unsetenv("STORMKIT_HTTP_WRITE_TIMEOUT")
	os.Unsetenv("STORMKIT_HTTP_IDLE_TIMEOUT")
}

func (s *PackageSuite) Test_HTTPTimeouts_Defaults() {
	c := config.New()
	s.Equal(30*time.Second, c.HTTPTimeouts.ReadTimeout)
	s.Equal(30*time.Second, c.HTTPTimeouts.WriteTimeout)
	s.Equal(60*time.Second, c.HTTPTimeouts.IdleTimeout)
}

func (s *PackageSuite) Test_HTTPTimeouts_ValidValues() {
	os.Setenv("STORMKIT_HTTP_READ_TIMEOUT", "5s")
	os.Setenv("STORMKIT_HTTP_WRITE_TIMEOUT", "10s")
	os.Setenv("STORMKIT_HTTP_IDLE_TIMEOUT", "2m")

	c := config.New()
	s.Equal(5*time.Second, c.HTTPTimeouts.ReadTimeout)
	s.Equal(10*time.Second, c.HTTPTimeouts.WriteTimeout)
	s.Equal(2*time.Minute, c.HTTPTimeouts.IdleTimeout)
}

func (s *PackageSuite) Test_HTTPTimeouts_InvalidValue_FallsBackToDefault() {
	// Unparseable duration should fall back to default.
	os.Setenv("STORMKIT_HTTP_READ_TIMEOUT", "notaduration")
	// Non-positive durations should also fall back to defaults.
	os.Setenv("STORMKIT_HTTP_WRITE_TIMEOUT", "0s")
	os.Setenv("STORMKIT_HTTP_IDLE_TIMEOUT", "-1s")

	c := config.New()
	s.Equal(30*time.Second, c.HTTPTimeouts.ReadTimeout)
	s.Equal(30*time.Second, c.HTTPTimeouts.WriteTimeout)
	s.Equal(60*time.Second, c.HTTPTimeouts.IdleTimeout)
}

func TestPackages(t *testing.T) {
	suite.Run(t, &PackageSuite{})
}
