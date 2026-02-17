package hosting_test

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HostSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HostSuite) SetupSuite() {
	s.conn = databasetest.InitTx("host_suite")
	s.Factory = factory.New(s.conn)
}

func (s *HostSuite) TearDownSuite() {
	s.conn.CloseTx()
}

func (s *HostSuite) host() *hosting.Host {
	return &hosting.Host{Request: shttp.NewRequestContext(nil), Name: "test-domain"}
}

func (s *HostSuite) Test_RequestConfig_InvalidDisplayName() {
	usr := s.MockUser()
	app := s.MockApp(usr, map[string]any{"DisplayName": "my-app"})
	env := s.MockEnv(app)

	_ = s.MockDeployment(env)

	conf, err := hosting.FetchAppConf("asdfijasdiofjsiodfjasio--1.stormkit:8888")
	s.NoError(err)
	s.Len(conf, 0)
}

func (s *HostSuite) Test_RequestConfig_CaseInsensitivy() {
	usr := s.MockUser()
	app := s.MockApp(usr, map[string]any{"DisplayName": "my-APP"})
	env := s.MockEnv(app)
	dep := s.MockDeployment(env, map[string]any{
		"Published": deploy.PublishedInfo{
			{EnvID: env.ID, Percentage: 100},
		},
	})

	conf, err := hosting.FetchAppConf("my-app.stormkit:8888")
	s.NoError(err)
	s.Len(conf, 1)
	s.Equal(conf[0].DeploymentID, dep.ID)
}

func (s *HostSuite) Test_ChooseVersion_MultipleVersions() {
	h := s.host()

	confs := []*appconf.Config{
		{Percentage: 100, DeploymentID: 1},
		{Percentage: 0, DeploymentID: 2},
	}

	s.Equal(confs[0], h.ChooseVersion(confs))
}

func (s *HostSuite) Test_ChooseVersion_MultipleVersionsWithVersionCookie() {
	req := &http.Request{
		Header: map[string][]string{
			"Cookie": {
				fmt.Sprintf("%s=3", hosting.VersionCookieName),
			},
		},
	}

	h := &hosting.Host{Request: shttp.NewRequestContext(req)}

	confs := []*appconf.Config{
		{Percentage: 25, DeploymentID: 1},
		{Percentage: 65, DeploymentID: 2},
		{Percentage: 10, DeploymentID: 3},
	}

	s.Equal(confs[2], h.ChooseVersion(confs))
}

func (s *HostSuite) Test_ChooseVersion_SingleConf() {
	h := s.host()

	confs := []*appconf.Config{
		{Percentage: 25, DeploymentID: 1},
	}

	s.Equal(confs[0], h.ChooseVersion(confs))
}

func (s *HostSuite) Test_ChooseVersion_NoConf() {
	h := s.host()
	s.Nil(h.ChooseVersion([]*appconf.Config{}))
}

func (s *HostSuite) Test_HostNameIdentifier() {
	s.Equal("my-app--12345", hosting.HostNameIdentifier("my-app--12345.stormkit:8888"))
	s.Equal("my-app--staging", hosting.HostNameIdentifier("my-app--staging.stormkit:8888"))
	s.Equal("my-app.com", hosting.HostNameIdentifier("my-app.com"))
}

// This is a test for proxying the URL, which is used in host_shttp.go file.
func (s *HostSuite) Test_ModifyingURL() {
	u1, err := url.Parse("https://stormkit:8888/my-app?id=12345#section")
	s.NoError(err)
	u2, err := url.Parse("http://abc.com:3000")
	s.NoError(err)
	u2.RawQuery = u1.RawQuery
	u2.Fragment = u1.Fragment
	u2.Path = u1.Path
	s.Equal("http://abc.com:3000/my-app?id=12345#section", u2.String())
}

func TestHostModel(t *testing.T) {
	suite.Run(t, &HostSuite{})
}

var benchDBOnce sync.Once
var benchDB databasetest.TestDB

func setupBenchDB() {
	benchDBOnce.Do(func() {
		benchDB = databasetest.InitTx("fetch_app_conf_bench")
	})
}

// BenchmarkFetchAppConfCacheMiss simulates concurrent requests for unique hostnames,
// which forces cache misses and tests lock contention under load.
func BenchmarkFetchAppConfCacheMiss(b *testing.B) {
	setupBenchDB()

	var counter int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Unique hostname each time = guaranteed cache miss
			i := atomic.AddInt64(&counter, 1)
			hostname := fmt.Sprintf("bench-test-%d.example.com", i)
			hosting.FetchAppConf(hostname)
		}
	})
}

// BenchmarkFetchAppConfCacheHit simulates concurrent requests for the same hostname,
// which tests cache hit performance.
func BenchmarkFetchAppConfCacheHit(b *testing.B) {
	setupBenchDB()

	// Prime the cache
	hosting.FetchAppConf("cached-host.example.com")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			hosting.FetchAppConf("cached-host.example.com")
		}
	})
}
