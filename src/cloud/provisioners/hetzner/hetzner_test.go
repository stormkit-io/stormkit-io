package hetzner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
	"github.com/stormkit-io/stormkit-io/src/cloud/provisioners"
	"github.com/stretchr/testify/suite"
)

// fakeHetzner is a minimal Hetzner Cloud API: server types, datacenters with
// stock, server create (optionally failing per type@location) and delete.
type fakeHetzner struct {
	mu sync.Mutex

	serverTypes map[int64]string

	// stock lists, per location name, the in-stock server type names.
	stock map[string][]string

	// createErrors maps "type@location" to the Hetzner error code to fail with.
	createErrors map[string]string

	created []schema.ServerCreateRequest
	deleted []int64
}

func (f *fakeHetzner) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /server_types", func(w http.ResponseWriter, r *http.Request) {
		types := []schema.ServerType{}

		for id, name := range f.serverTypes {
			types = append(types, schema.ServerType{ID: id, Name: name})
		}

		f.json(w, http.StatusOK, schema.ServerTypeListResponse{ServerTypes: types})
	})

	mux.HandleFunc("GET /datacenters", func(w http.ResponseWriter, r *http.Request) {
		dcs := []schema.Datacenter{}
		dcID := int64(1)

		for location, names := range f.stock {
			available := []int64{}

			for _, name := range names {
				available = append(available, f.typeID(name))
			}

			dcs = append(dcs, schema.Datacenter{
				ID:          dcID,
				Name:        location + "-dc1",
				Location:    schema.Location{ID: dcID, Name: location},
				ServerTypes: schema.DatacenterServerTypes{Available: available},
			})

			dcID++
		}

		f.json(w, http.StatusOK, schema.DatacenterListResponse{Datacenters: dcs})
	})

	mux.HandleFunc("POST /servers", func(w http.ResponseWriter, r *http.Request) {
		req := schema.ServerCreateRequest{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		f.mu.Lock()
		f.created = append(f.created, req)
		attempt := len(f.created)
		f.mu.Unlock()

		serverType := req.ServerType.Name

		if code := f.createErrors[serverType+"@"+req.Location]; code != "" {
			f.json(w, http.StatusConflict, schema.ErrorResponse{Error: schema.Error{Code: code, Message: code}})
			return
		}

		server := schema.Server{
			ID:   int64(1000 + attempt),
			Name: req.Name,
			PublicNet: schema.ServerPublicNet{
				IPv4: schema.ServerPublicNetIPv4{IP: "203.0.113.7"},
				IPv6: schema.ServerPublicNetIPv6{IP: "2001:db8::/64"},
			},
		}

		f.json(w, http.StatusCreated, schema.ServerCreateResponse{Server: server, Action: schema.Action{ID: 1, Status: "running"}})
	})

	mux.HandleFunc("DELETE /servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

		f.mu.Lock()
		f.deleted = append(f.deleted, id)
		f.mu.Unlock()

		f.json(w, http.StatusOK, schema.ServerDeleteResponse{Action: schema.Action{ID: 2, Status: "running"}})
	})

	return mux
}

func (f *fakeHetzner) typeID(name string) int64 {
	for id, n := range f.serverTypes {
		if n == name {
			return id
		}
	}

	return 0
}

func (f *fakeHetzner) json(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// attempts returns the "type@location" of every create request, in order.
func (f *fakeHetzner) attempts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := []string{}

	for _, req := range f.created {
		serverType := req.ServerType.Name
		out = append(out, serverType+"@"+req.Location)
	}

	return out
}

type ProvisionerSuite struct {
	suite.Suite

	hetzner     *fakeHetzner
	hetznerSrv  *httptest.Server
	instanceSrv *httptest.Server

	// readyAfter is the number of probes the fake instance fails before
	// answering 200. Negative means it never becomes ready.
	readyAfter int32
	probes     atomic.Int32
	probeKeys  sync.Map
}

func (s *ProvisionerSuite) SetupTest() {
	s.hetzner = &fakeHetzner{
		serverTypes: map[int64]string{1: "cx23", 2: "cx33"},
		stock: map[string][]string{
			"nbg1": {"cx23", "cx33"},
			"fsn1": {"cx23", "cx33"},
			"hel1": {"cx23", "cx33"},
		},
		createErrors: map[string]string{},
	}

	s.hetznerSrv = httptest.NewServer(s.hetzner.handler())
	s.readyAfter = 0
	s.probes.Store(0)
	s.probeKeys = sync.Map{}

	s.instanceSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := s.probes.Add(1)
		s.probeKeys.Store(r.Header.Get("Authorization"), true)

		if r.URL.Path != "/v1/mcp" || r.Method != http.MethodPost || s.readyAfter < 0 || n <= s.readyAfter {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
}

func (s *ProvisionerSuite) TearDownTest() {
	s.hetznerSrv.Close()
	s.instanceSrv.Close()
}

// provisioner returns a Provisioner wired to the fake Hetzner API and the
// fake instance, polling fast enough for tests.
func (s *ProvisionerSuite) provisioner() *Provisioner {
	p := New(Config{
		Token:            "test-token",
		ServerTypes:      []string{"cx23", "cx33"},
		Locations:        []string{"nbg1", "fsn1", "hel1"},
		Image:            "ubuntu-24.04",
		FirewallID:       42,
		SSHKeyIDs:        []int64{7, 8},
		InstallScriptURL: "https://www.stormkit.io/install.sh",
		ReadyTimeout:     100 * time.Millisecond,
	})

	p.hcloud = hcloud.NewClient(
		hcloud.WithToken("test-token"),
		hcloud.WithEndpoint(s.hetznerSrv.URL),
		hcloud.WithRetryOpts(hcloud.RetryOpts{MaxRetries: 0}),
	)

	p.readiness.PollInterval = 5 * time.Millisecond
	p.readiness.APIURL = func(string) string { return s.instanceSrv.URL }

	return p
}

func (s *ProvisionerSuite) create(params provisioners.CreateInstanceParams) (*provisioners.Instance, error) {
	return s.provisioner().CreateInstance(context.Background(), params)
}

func (s *ProvisionerSuite) Test_CreateInstance_FirstChoice() {
	instance, err := s.create(provisioners.CreateInstanceParams{Name: "acme", AdminEmail: "owner@acme.com"})

	s.Require().NoError(err)
	s.Equal([]string{"cx23@nbg1"}, s.hetzner.attempts())
	s.Equal("hetzner", instance.Provider)
	s.Equal(int64(1001), instance.ServerID)
	s.Equal("cx23", instance.ServerType)
	s.Equal("nbg1", instance.Location)
	s.Equal("203.0.113.7", instance.IPv4)
	s.Equal("203-0-113-7.sslip.io", instance.Domain)
	s.Regexp(`^SK_[0-9a-f]{62}$`, instance.APIKey)

	_, probedWithKey := s.probeKeys.Load("Bearer " + instance.APIKey)
	s.True(probedWithKey)

	req := s.hetzner.created[0]
	s.Equal("stormkit-acme", req.Name)
	s.Equal("ubuntu-24.04", req.Image.Name)
	s.Equal(map[string]string{instanceLabel: "acme"}, *req.Labels)
	s.Equal([]schema.ServerCreateFirewalls{{Firewall: 42}}, req.Firewalls)
	s.Equal([]int64{7, 8}, req.SSHKeys)
	s.Contains(req.UserData, "--api-key '"+instance.APIKey+"'")
	s.Contains(req.UserData, "--email 'owner@acme.com'")
	s.NotContains(req.UserData, "--domain")
	s.Empty(s.hetzner.deleted)
}

// Test_CreateInstance_SkipsOutOfStock verifies that a location without the
// server type in stock is never attempted.
func (s *ProvisionerSuite) Test_CreateInstance_SkipsOutOfStock() {
	s.hetzner.stock["nbg1"] = []string{"cx33"}

	instance, err := s.create(provisioners.CreateInstanceParams{Name: "acme"})

	s.Require().NoError(err)
	s.Equal([]string{"cx23@fsn1"}, s.hetzner.attempts())
	s.Equal("fsn1", instance.Location)
}

// Test_CreateInstance_CapacityErrorTriesNext verifies that stock reported as
// available but refused at create time moves on to the next option.
func (s *ProvisionerSuite) Test_CreateInstance_CapacityErrorTriesNext() {
	s.hetzner.createErrors["cx23@nbg1"] = string(hcloud.ErrorCodeResourceUnavailable)
	s.hetzner.createErrors["cx23@fsn1"] = string(hcloud.ErrorCodePlacementError)

	instance, err := s.create(provisioners.CreateInstanceParams{Name: "acme"})

	s.Require().NoError(err)
	s.Equal([]string{"cx23@nbg1", "cx23@fsn1", "cx23@hel1"}, s.hetzner.attempts())
	s.Equal("hel1", instance.Location)
}

func (s *ProvisionerSuite) Test_CreateInstance_FallsBackToNextServerType() {
	for location := range s.hetzner.stock {
		s.hetzner.stock[location] = []string{"cx33"}
	}

	instance, err := s.create(provisioners.CreateInstanceParams{Name: "acme"})

	s.Require().NoError(err)
	s.Equal([]string{"cx33@nbg1"}, s.hetzner.attempts())
	s.Equal("cx33", instance.ServerType)
}

func (s *ProvisionerSuite) Test_CreateInstance_NoCapacity() {
	s.hetzner.stock = map[string][]string{"ash": {"cx23"}}

	instance, err := s.create(provisioners.CreateInstanceParams{Name: "acme"})

	s.ErrorIs(err, provisioners.ErrNoCapacity)
	s.Nil(instance)
	s.Empty(s.hetzner.attempts())
}

// Test_CreateInstance_OtherErrorStops verifies that a non-capacity error is
// returned right away instead of being retried in every location.
func (s *ProvisionerSuite) Test_CreateInstance_OtherErrorStops() {
	s.hetzner.createErrors["cx23@nbg1"] = string(hcloud.ErrorCodeUniquenessError)

	instance, err := s.create(provisioners.CreateInstanceParams{Name: "acme"})

	s.True(hcloud.IsError(err, hcloud.ErrorCodeUniquenessError))
	s.Nil(instance)
	s.Equal([]string{"cx23@nbg1"}, s.hetzner.attempts())
}

func (s *ProvisionerSuite) Test_CreateInstance_WaitsUntilReady() {
	s.readyAfter = 3

	instance, err := s.create(provisioners.CreateInstanceParams{Name: "acme", Domain: "acme.stormkit.dev"})

	s.Require().NoError(err)
	s.Equal("acme.stormkit.dev", instance.Domain)
	s.Equal(int32(4), s.probes.Load())
	s.Contains(s.hetzner.created[0].UserData, "--domain 'acme.stormkit.dev'")
	s.Empty(s.hetzner.deleted)
}

// Test_CreateInstance_DeletesServerWhenNeverReady verifies that a server that
// never accepts its key is deleted, so nothing keeps billing.
func (s *ProvisionerSuite) Test_CreateInstance_DeletesServerWhenNeverReady() {
	s.readyAfter = -1

	instance, err := s.create(provisioners.CreateInstanceParams{Name: "acme"})

	s.ErrorIs(err, provisioners.ErrNotReady)
	s.Contains(err.Error(), "HTTP 502")
	s.Nil(instance)
	s.Equal([]int64{1001}, s.hetzner.deleted)
}

func (s *ProvisionerSuite) Test_CreateInstance_InvalidParams() {
	_, err := s.create(provisioners.CreateInstanceParams{Name: "Acme_Corp"})

	s.Require().Error(err)
	s.Empty(s.hetzner.attempts())
}

func (s *ProvisionerSuite) Test_DeleteInstance() {
	s.Require().NoError(s.provisioner().DeleteInstance(context.Background(), 1234))
	s.Equal([]int64{1234}, s.hetzner.deleted)
}

func TestProvisionerSuite(t *testing.T) {
	suite.Run(t, new(ProvisionerSuite))
}

type ConfigSuite struct {
	suite.Suite
}

func (s *ConfigSuite) Test_Defaults() {
	s.T().Setenv("HETZNER_API_KEY", "token")

	cfg, err := ConfigFromEnv()

	s.Require().NoError(err)
	s.Equal([]string{"cx23", "cx33"}, cfg.ServerTypes)
	s.Equal([]string{"nbg1", "fsn1", "hel1"}, cfg.Locations)
	s.Equal("ubuntu-24.04", cfg.Image)
	s.Equal("https://www.stormkit.io/install.sh", cfg.InstallScriptURL)
	s.Equal(20*time.Minute, cfg.ReadyTimeout)
	s.Zero(cfg.FirewallID)
	s.Empty(cfg.SSHKeyIDs)
}

func (s *ConfigSuite) Test_Overrides() {
	s.T().Setenv("HETZNER_API_KEY", "token")
	s.T().Setenv("HETZNER_SERVER_TYPES", " cx33 , cpx22,")
	s.T().Setenv("HETZNER_LOCATIONS", "hel1")
	s.T().Setenv("HETZNER_FIREWALL_ID", "99")
	s.T().Setenv("HETZNER_SSH_KEY_IDS", "1, 2")
	s.T().Setenv("PROVISIONER_READY_TIMEOUT", "5m")

	cfg, err := ConfigFromEnv()

	s.Require().NoError(err)
	s.Equal([]string{"cx33", "cpx22"}, cfg.ServerTypes)
	s.Equal([]string{"hel1"}, cfg.Locations)
	s.Equal(int64(99), cfg.FirewallID)
	s.Equal([]int64{1, 2}, cfg.SSHKeyIDs)
	s.Equal(5*time.Minute, cfg.ReadyTimeout)
}

// Test_ReportsAllErrors verifies that every misconfiguration is reported at once.
func (s *ConfigSuite) Test_ReportsAllErrors() {
	s.T().Setenv("HETZNER_API_KEY", "")
	s.T().Setenv("HETZNER_FIREWALL_ID", "abc")
	s.T().Setenv("HETZNER_SSH_KEY_IDS", "1,x")
	s.T().Setenv("PROVISIONER_READY_TIMEOUT", "soon")

	_, err := ConfigFromEnv()

	s.Require().Error(err)

	for _, want := range []string{"HETZNER_API_KEY", "HETZNER_FIREWALL_ID", "HETZNER_SSH_KEY_IDS", "PROVISIONER_READY_TIMEOUT"} {
		s.Contains(err.Error(), want)
	}

}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
