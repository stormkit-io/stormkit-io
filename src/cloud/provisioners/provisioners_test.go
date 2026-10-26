package provisioners_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/stormkit-io/stormkit-io/src/cloud/provisioners"
	"github.com/stretchr/testify/suite"
)

type CreateInstanceParamsSuite struct {
	suite.Suite
}

func (s *CreateInstanceParamsSuite) Test_Valid() {
	s.NoError(provisioners.CreateInstanceParams{Name: "acme"}.Validate())
	s.NoError(provisioners.CreateInstanceParams{Name: "a", AdminEmail: "owner@acme.com"}.Validate())
}

// Test_ReportsAllErrors verifies that every invalid field is reported at once.
func (s *CreateInstanceParamsSuite) Test_ReportsAllErrors() {
	err := provisioners.CreateInstanceParams{
		Name:       "Acme_Corp",
		AdminEmail: "Owner <owner@acme.com>",
	}.Validate()

	s.Require().Error(err)
	s.Contains(err.Error(), `name "Acme_Corp"`)
	s.Contains(err.Error(), "admin email")
}

func (s *CreateInstanceParamsSuite) Test_RejectsBadNames() {
	for _, name := range []string{"", "-acme", "acme-", "ac me", "acme.io", strings.Repeat("a", 43)} {
		s.Error(provisioners.CreateInstanceParams{Name: name}.Validate(), name)
	}
}

func TestCreateInstanceParamsSuite(t *testing.T) {
	suite.Run(t, new(CreateInstanceParamsSuite))
}

type CloudInitSuite struct {
	suite.Suite
}

func (s *CloudInitSuite) Test_Render() {
	out, err := provisioners.CloudInit{
		InstallScriptURL: "https://www.stormkit.io/install.sh",
		APIKey:           "SK_abc",
		AdminEmail:       "o'brien@acme.com",
	}.Render()

	s.Require().NoError(err)
	s.True(strings.HasPrefix(out, "#cloud-config\n"))
	s.Equal([]string{
		"mkdir -p /opt/stormkit",
		`cd /opt/stormkit && curl -fsSL 'https://www.stormkit.io/install.sh' | sh -s -- --agent --api-key 'SK_abc' --email 'o'\''brien@acme.com'`,
	}, s.doc(out).Runcmd)
}

// Test_Render_BlocksMetadataFromContainers verifies that containers on the
// instance cannot read the user-data, and its owner key, from the metadata
// service, on every boot.
func (s *CloudInitSuite) Test_Render_BlocksMetadataFromContainers() {
	out, err := provisioners.CloudInit{InstallScriptURL: "https://x.test/install.sh", APIKey: "SK_abc"}.Render()

	s.Require().NoError(err)
	s.Equal([]string{
		"iptables -N DOCKER-USER 2>/dev/null || true",
		"iptables -C DOCKER-USER -d 169.254.169.254 -j DROP 2>/dev/null || iptables -I DOCKER-USER -d 169.254.169.254 -j DROP",
	}, s.doc(out).Bootcmd)
}

type cloudConfig struct {
	Bootcmd []string `yaml:"bootcmd"`
	Runcmd  []string `yaml:"runcmd"`
}

// doc parses a rendered cloud-config.
func (s *CloudInitSuite) doc(out string) cloudConfig {
	doc := cloudConfig{}

	s.Require().NoError(yaml.Unmarshal([]byte(out), &doc))

	return doc
}

func (s *CloudInitSuite) Test_Render_OmitsOptionalFlags() {
	out, err := provisioners.CloudInit{InstallScriptURL: "https://x.test/install.sh", APIKey: "SK_abc"}.Render()

	s.Require().NoError(err)
	s.Equal(`cd /opt/stormkit && curl -fsSL 'https://x.test/install.sh' | sh -s -- --agent --api-key 'SK_abc'`, s.doc(out).Runcmd[1])
}

func TestCloudInitSuite(t *testing.T) {
	suite.Run(t, new(CloudInitSuite))
}

type ReadinessSuite struct {
	suite.Suite

	srv *httptest.Server

	// readyAfter is the number of probes that fail before the fake instance
	// answers 200. Negative means it never becomes ready.
	readyAfter int32
	probes     atomic.Int32
	lastAuth   atomic.Value
}

func (s *ReadinessSuite) SetupTest() {
	s.readyAfter = 0
	s.probes.Store(0)

	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := s.probes.Add(1)
		s.lastAuth.Store(r.Header.Get("Authorization"))

		if r.URL.Path != "/v1/mcp" || r.Method != http.MethodPost || s.readyAfter < 0 || n <= s.readyAfter {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
}

func (s *ReadinessSuite) TearDownTest() {
	s.srv.Close()
}

func (s *ReadinessSuite) wait() error {
	return s.waitCtx(context.Background())
}

func (s *ReadinessSuite) waitCtx(ctx context.Context) error {
	r := provisioners.NewReadiness(200 * time.Millisecond)
	r.PollInterval = 5 * time.Millisecond
	r.APIURL = func(domain string) string { return s.srv.URL + "/" }

	return r.Wait(ctx, &provisioners.Instance{Domain: "acme.stormkit.dev", APIKey: "SK_abc"})
}

// Test_RetriesUntilReady verifies that early 401s, before bootstrap stores
// the key, are retried rather than treated as final.
func (s *ReadinessSuite) Test_RetriesUntilReady() {
	s.readyAfter = 2

	s.NoError(s.wait())
	s.Equal(int32(3), s.probes.Load())
	s.Equal("Bearer SK_abc", s.lastAuth.Load())
}

func (s *ReadinessSuite) Test_TimesOut() {
	s.readyAfter = -1

	err := s.wait()

	s.ErrorIs(err, provisioners.ErrNotReady)
	s.Contains(err.Error(), "HTTP 401")
}

// Test_CallerCancelIsNotATimeout verifies that giving up early is reported
// as a cancellation rather than as an instance that never became ready.
func (s *ReadinessSuite) Test_CallerCancelIsNotATimeout() {
	s.readyAfter = -1

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := s.waitCtx(ctx)

	s.Require().Error(err)
	s.NotErrorIs(err, provisioners.ErrNotReady)
	s.ErrorIs(err, context.DeadlineExceeded)
}

func (s *ReadinessSuite) Test_DefaultAPIURL() {
	s.Equal("https://api.acme.stormkit.dev", provisioners.NewReadiness(time.Minute).APIURL("acme.stormkit.dev"))
}

func TestReadinessSuite(t *testing.T) {
	suite.Run(t, new(ReadinessSuite))
}
