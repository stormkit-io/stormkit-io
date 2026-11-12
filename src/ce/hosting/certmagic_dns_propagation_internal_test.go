package hosting

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/libdns"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/suite"
)

// fakeDNSProvider records what was added and can fail on demand.
type fakeDNSProvider struct {
	certmagic.DNSProvider

	appendErr error
	appended  []libdns.Record
}

func (f *fakeDNSProvider) AppendRecords(_ context.Context, _ string, recs []libdns.Record) ([]libdns.Record, error) {
	f.appended = append(f.appended, recs...)
	return recs, f.appendErr
}

// fakeNameservers serves a record on a nameserver once it has been asked a
// given number of times, mimicking nameservers that update one by one.
type fakeNameservers struct {
	mu        sync.Mutex
	servers   []string
	readyIn   map[string]int
	asked     map[string]int
	failFirst map[string]bool
	value     string
}

func (f *fakeNameservers) nameservers(_ context.Context, _ string) ([]string, error) {
	return f.servers, nil
}

func (f *fakeNameservers) txt(_ context.Context, q txtQuery) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.asked[q.Nameserver]++

	if f.failFirst[q.Nameserver] && f.asked[q.Nameserver] == 1 {
		return nil, errors.New("i/o timeout")
	}

	if f.asked[q.Nameserver] >= f.readyIn[q.Nameserver] {
		return []string{"unrelated", f.value}, nil
	}

	return []string{"unrelated"}, nil
}

type AllNameserversProviderSuite struct {
	suite.Suite
	inner *fakeDNSProvider
	ns    *fakeNameservers
}

func (s *AllNameserversProviderSuite) SetupTest() {
	s.inner = &fakeDNSProvider{}
	s.ns = &fakeNameservers{
		servers: []string{"hydrogen:53", "oxygen:53", "helium:53"},
		readyIn: map[string]int{"hydrogen:53": 1, "oxygen:53": 2, "helium:53": 4},
		asked:   map[string]int{},
		value:   "challenge-digest",
	}
}

func (s *AllNameserversProviderSuite) provider(timeout time.Duration) *allNameserversProvider {
	return &allNameserversProvider{
		DNSProvider: s.inner,
		querier:     s.ns,
		interval:    time.Millisecond,
		timeout:     timeout,
	}
}

func (s *AllNameserversProviderSuite) challenge() []libdns.Record {
	return []libdns.Record{libdns.TXT{Name: "_acme-challenge", Text: "challenge-digest"}}
}

// Test_WaitsForEveryNameserver verifies that the record must reach the slowest
// nameserver, not just the first one, before the challenge goes ahead.
func (s *AllNameserversProviderSuite) Test_WaitsForEveryNameserver() {
	added, err := s.provider(time.Minute).AppendRecords(context.Background(), "stormkit.dev.", s.challenge())

	s.NoError(err)
	s.Len(added, 1)
	s.Len(s.inner.appended, 1)
	s.Equal(4, s.ns.asked["helium:53"])
	s.Equal(1, s.ns.asked["hydrogen:53"])
}

// Test_RetriesNameserverThatFailsToAnswer verifies that a nameserver that
// doesn't answer is asked again rather than skipped.
func (s *AllNameserversProviderSuite) Test_RetriesNameserverThatFailsToAnswer() {
	s.ns.readyIn = map[string]int{"hydrogen:53": 1, "oxygen:53": 1, "helium:53": 1}
	s.ns.failFirst = map[string]bool{"oxygen:53": true}

	_, err := s.provider(time.Minute).AppendRecords(context.Background(), "stormkit.dev.", s.challenge())

	s.NoError(err)
	s.Equal(2, s.ns.asked["oxygen:53"])
}

// Test_GoesAheadAfterTimeout verifies that a record that never reaches one
// nameserver doesn't fail the append, so certmagic still tries the challenge
// and later deletes the record.
func (s *AllNameserversProviderSuite) Test_GoesAheadAfterTimeout() {
	s.ns.readyIn["helium:53"] = 1 << 30

	added, err := s.provider(20*time.Millisecond).AppendRecords(context.Background(), "stormkit.dev.", s.challenge())

	s.NoError(err)
	s.Len(added, 1)
	s.Greater(s.ns.asked["helium:53"], 1)
}

func (s *AllNameserversProviderSuite) Test_ReturnsProviderError() {
	s.inner.appendErr = errors.New("unauthorized")

	_, err := s.provider(time.Minute).AppendRecords(context.Background(), "stormkit.dev.", s.challenge())

	s.EqualError(err, "unauthorized")
	s.Empty(s.ns.asked)
}

func (s *AllNameserversProviderSuite) Test_SkipsNonTXTRecords() {
	recs := []libdns.Record{libdns.CNAME{Name: "_acme-challenge", Target: "elsewhere.example."}}

	_, err := s.provider(time.Minute).AppendRecords(context.Background(), "stormkit.dev.", recs)

	s.NoError(err)
	s.Empty(s.ns.asked)
}

// Test_AuthoritativeDNS_ReadsTXT queries a local DNS server to check the real
// query: no recursion, and multi-string TXT values joined.
func (s *AllNameserversProviderSuite) Test_AuthoritativeDNS_ReadsTXT() {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	s.Require().NoError(err)

	var recursionDesired atomic.Bool

	server := &dns.Server{PacketConn: conn, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		recursionDesired.Store(r.RecursionDesired)

		res := new(dns.Msg)
		res.SetReply(r)
		res.Answer = append(res.Answer, &dns.TXT{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
			Txt: []string{"challenge-", "digest"},
		})

		_ = w.WriteMsg(res)
	})}

	go func() { _ = server.ActivateAndServe() }()
	defer server.Shutdown()

	querier := authoritativeDNS{client: &dns.Client{Timeout: time.Second}}
	values, err := querier.txt(context.Background(), txtQuery{Nameserver: conn.LocalAddr().String(), FQDN: "_acme-challenge.stormkit.dev"})

	s.NoError(err)
	s.Equal([]string{"challenge-digest"}, values)
	s.False(recursionDesired.Load())
}

func TestAllNameserversProviderSuite(t *testing.T) {
	suite.Run(t, new(AllNameserversProviderSuite))
}
