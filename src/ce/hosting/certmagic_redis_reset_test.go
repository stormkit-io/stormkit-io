package hosting_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

// CertmagicRedisResetSuite covers the trap that recovering from a failover
// used to set: the certificate store took its connection once at boot, and the
// recovery closes that exact client. Certificates would then fail for the life
// of the process, which is worse than the failover it was recovering from.
type CertmagicRedisResetSuite struct {
	suite.Suite
	storage *hosting.RedisStorage
	ctx     context.Context
}

func (s *CertmagicRedisResetSuite) SetupTest() {
	s.ctx = context.Background()
	s.storage = hosting.NewRedisStorage(nil)
	s.storage.KeyPrefix = "test-certmagic-reset"
	s.storage.SetClientFunc(rediscache.UniversalClient)
}

func (s *CertmagicRedisResetSuite) TearDownTest() {
	keys, err := rediscache.Client().Keys(s.ctx, s.storage.KeyPrefix+"*")

	if err == nil && len(keys) > 0 {
		rediscache.Client().Del(s.ctx, keys...)
	}
}

func (s *CertmagicRedisResetSuite) Test_StoreSurvivesAClientReset() {
	key := "survives/cert.pem"

	s.Require().NoError(s.storage.Store(s.ctx, key, []byte("before")))

	// Exactly what a recovered failover does to the shared client.
	rediscache.Client().Reset()

	value, err := s.storage.Load(s.ctx, key)

	s.Require().NoError(err, "certificates must still be readable after a reset")
	s.Equal([]byte("before"), value)

	s.NoError(s.storage.Store(s.ctx, key+".2", []byte("after")), "and still writable")
}

func (s *CertmagicRedisResetSuite) Test_LockingSurvivesAClientReset() {
	s.Require().NoError(s.storage.Lock(s.ctx, "survives-lock"))
	s.Require().NoError(s.storage.Unlock(s.ctx, "survives-lock"))

	rediscache.Client().Reset()

	s.NoError(s.storage.Lock(s.ctx, "survives-lock"), "the lock client must follow the reset too")
	s.NoError(s.storage.Unlock(s.ctx, "survives-lock"))
}

func TestCertmagicRedisReset(t *testing.T) {
	suite.Run(t, &CertmagicRedisResetSuite{})
}
