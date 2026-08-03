package sysstats

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

type StoreSuite struct {
	suite.Suite
	store  *Store
	target string
	ctx    context.Context
}

func (s *StoreSuite) SetupTest() {
	s.ctx = context.Background()
	s.target = fmt.Sprintf("test-target-%d", time.Now().UnixNano())
	s.store = NewStore(NewStoreParams{})
}

func (s *StoreSuite) TearDownTest() {
	_ = s.store.Drop(s.ctx, s.target)
}

func (s *StoreSuite) sample(ts time.Time, cpu float64) *Sample {
	return &Sample{
		Target:     s.target,
		Timestamp:  ts.Unix(),
		Reachable:  true,
		CPUPercent: cpu,
		CPUValid:   true,
		Filesystems: []Filesystem{
			{Mountpoint: "/", SizeBytes: 100, AvailBytes: 40},
		},
	}
}

func (s *StoreSuite) Test_AppendAndRead() {
	now := time.Now()

	s.Require().NoError(s.store.Append(s.ctx, s.sample(now.Add(-2*time.Minute), 10)))
	s.Require().NoError(s.store.Append(s.ctx, s.sample(now.Add(-time.Minute), 20)))

	samples, err := s.store.Read(s.ctx, s.target, now.Add(-time.Hour))
	s.Require().NoError(err)
	s.Require().Len(samples, 2)

	s.Equal(float64(10), samples[0].CPUPercent, "oldest first")
	s.Equal(float64(20), samples[1].CPUPercent)
	s.Equal("/", samples[0].Filesystems[0].Mountpoint, "filesystems survive the round trip")
}

func (s *StoreSuite) Test_Read_HonoursSince() {
	now := time.Now()

	s.Require().NoError(s.store.Append(s.ctx, s.sample(now.Add(-10*time.Minute), 10)))
	s.Require().NoError(s.store.Append(s.ctx, s.sample(now, 20)))

	samples, err := s.store.Read(s.ctx, s.target, now.Add(-5*time.Minute))
	s.Require().NoError(err)
	s.Require().Len(samples, 1)
	s.Equal(float64(20), samples[0].CPUPercent)
}

// Anything past the retention window is dropped on the next write, so the key
// cannot grow without bound.
func (s *StoreSuite) Test_Append_TrimsBeyondRetention() {
	store := NewStore(NewStoreParams{Retention: time.Minute})
	now := time.Now()

	s.Require().NoError(store.Append(s.ctx, s.sample(now.Add(-10*time.Minute), 10)))
	s.Require().NoError(store.Append(s.ctx, s.sample(now, 20)))

	samples, err := store.Read(s.ctx, s.target, now.Add(-time.Hour))
	s.Require().NoError(err)
	s.Require().Len(samples, 1, "the sample outside the window was trimmed")
	s.Equal(float64(20), samples[0].CPUPercent)
}

func (s *StoreSuite) Test_Append_SetsExpiry() {
	s.Require().NoError(s.store.Append(s.ctx, s.sample(time.Now(), 10)))

	ttl, err := rediscache.Client().TTL(s.ctx, historyKey(s.target)).Result()
	s.Require().NoError(err)
	s.Positive(ttl, "a vanished machine must clean up after itself")
	s.LessOrEqual(ttl, keyTTL)
}

func (s *StoreSuite) Test_Latest() {
	now := time.Now()

	s.Require().NoError(s.store.Append(s.ctx, s.sample(now.Add(-time.Minute), 10)))
	s.Require().NoError(s.store.Append(s.ctx, s.sample(now, 99)))

	latest, err := s.store.Latest(s.ctx, s.target)
	s.Require().NoError(err)
	s.Require().NotNil(latest)
	s.Equal(float64(99), latest.CPUPercent)
}

func (s *StoreSuite) Test_Latest_NoSamplesYet() {
	latest, err := s.store.Latest(s.ctx, "target-that-never-reported")

	s.NoError(err, "a target with no history is not an error")
	s.Nil(latest)
}

func (s *StoreSuite) Test_Append_RejectsSampleWithoutTarget() {
	s.Error(s.store.Append(s.ctx, &Sample{Timestamp: time.Now().Unix()}))
}

// An unreachable machine is a fact worth keeping, not a reason to skip writing.
func (s *StoreSuite) Test_Append_StoresUnreachableSample() {
	s.Require().NoError(s.store.Append(s.ctx, &Sample{
		Target:    s.target,
		Timestamp: time.Now().Unix(),
		Reachable: false,
		Error:     "connection refused",
	}))

	latest, err := s.store.Latest(s.ctx, s.target)
	s.Require().NoError(err)
	s.Require().NotNil(latest)
	s.False(latest.Reachable)
	s.Equal("connection refused", latest.Error)
}

func (s *StoreSuite) Test_Drop() {
	s.Require().NoError(s.store.Append(s.ctx, s.sample(time.Now(), 10)))
	s.Require().NoError(s.store.Drop(s.ctx, s.target))

	latest, err := s.store.Latest(s.ctx, s.target)
	s.Require().NoError(err)
	s.Nil(latest)
}

func TestStoreSuite(t *testing.T) {
	suite.Run(t, new(StoreSuite))
}
