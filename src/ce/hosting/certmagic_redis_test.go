package hosting_test

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

type CertmagicRedisSuite struct {
	suite.Suite
	storage *hosting.RedisStorage
	ctx     context.Context
}

func (s *CertmagicRedisSuite) SetupSuite() {
	// Create a new RedisStorage with test configuration
	s.storage = hosting.NewRedisStorage(nil)
	s.storage.KeyPrefix = "test-certmagic"
	s.storage.SetClient(rediscache.Client())
	s.ctx = context.Background()
}

func (s *CertmagicRedisSuite) TearDownSuite() {
	// Clean up all test keys
	if s.storage != nil {
		keys, err := rediscache.Client().Keys(s.ctx, s.storage.KeyPrefix+"*").Result()

		if err == nil && len(keys) > 0 {
			rediscache.Client().Del(s.ctx, keys...)
		}
	}
}

func (s *CertmagicRedisSuite) BeforeTest(_, _ string) {
	// Clean up keys before each test
	keys, err := rediscache.Client().Keys(s.ctx, s.storage.KeyPrefix+"*").Result()

	if err == nil && len(keys) > 0 {
		rediscache.Client().Del(s.ctx, keys...)
	}
}

// Test_Store tests storing data in Redis
func (s *CertmagicRedisSuite) Test_Store() {
	key := "certs/example.com/cert.pem"
	value := []byte("-----BEGIN CERTIFICATE-----\ntest certificate data\n-----END CERTIFICATE-----")

	err := s.storage.Store(s.ctx, key, value)
	s.NoError(err)

	// Verify the key exists
	s.True(s.storage.Exists(s.ctx, key))
}

// Test_Store_EmptyValue tests storing empty data
func (s *CertmagicRedisSuite) Test_Store_EmptyValue() {
	key := "certs/empty/file.txt"
	value := []byte("")

	err := s.storage.Store(s.ctx, key, value)
	s.NoError(err)
	s.True(s.storage.Exists(s.ctx, key))
}

// Test_Load tests loading data from Redis
func (s *CertmagicRedisSuite) Test_Load() {
	key := "certs/example.com/key.pem"
	expectedValue := []byte("-----BEGIN PRIVATE KEY-----\ntest private key\n-----END PRIVATE KEY-----")

	// Store data first
	err := s.storage.Store(s.ctx, key, expectedValue)
	s.NoError(err)

	// Load it back
	actualValue, err := s.storage.Load(s.ctx, key)
	s.NoError(err)
	s.Equal(expectedValue, actualValue)
}

// Test_Load_NotFound tests loading non-existent key
func (s *CertmagicRedisSuite) Test_Load_NotFound() {
	key := "nonexistent/key.pem"

	_, err := s.storage.Load(s.ctx, key)
	s.Error(err)
	s.ErrorIs(err, fs.ErrNotExist)
}

// Test_Delete tests deleting data from Redis
func (s *CertmagicRedisSuite) Test_Delete() {
	key := "certs/todelete/cert.pem"
	value := []byte("certificate to delete")

	// Store data first
	err := s.storage.Store(s.ctx, key, value)
	s.NoError(err)
	s.True(s.storage.Exists(s.ctx, key))

	// Delete it
	err = s.storage.Delete(s.ctx, key)
	s.NoError(err)

	// Verify it's gone
	s.False(s.storage.Exists(s.ctx, key))
}

// Test_Delete_NonExistent tests deleting a key that doesn't exist
func (s *CertmagicRedisSuite) Test_Delete_NonExistent() {
	key := "nonexistent/key.pem"

	// Should not error when deleting non-existent key
	err := s.storage.Delete(s.ctx, key)
	s.NoError(err)
}

// Test_Exists tests checking if keys exist
func (s *CertmagicRedisSuite) Test_Exists() {
	key := "certs/exists/cert.pem"

	// Should not exist initially
	s.False(s.storage.Exists(s.ctx, key))

	// Store data
	err := s.storage.Store(s.ctx, key, []byte("data"))
	s.NoError(err)

	// Should exist now
	s.True(s.storage.Exists(s.ctx, key))

	// Delete it
	err = s.storage.Delete(s.ctx, key)
	s.NoError(err)

	// Should not exist anymore
	s.False(s.storage.Exists(s.ctx, key))
}

// Test_List_NonRecursive tests listing directory contents non-recursively
func (s *CertmagicRedisSuite) Test_List_NonRecursive() {
	// Store test files in multiple subdirectories
	files := map[string][]byte{
		"certs/site1.com/cert.pem": []byte("cert1"),
		"certs/site1.com/key.pem":  []byte("key1"),
		"certs/site2.com/cert.pem": []byte("cert2"),
		"certs/site2.com/key.pem":  []byte("key2"),
	}

	for key, value := range files {
		err := s.storage.Store(s.ctx, key, value)
		s.NoError(err)
	}

	// List files in certs directory (non-recursive)
	keys, err := s.storage.List(s.ctx, "certs", false)
	s.NoError(err)

	// Should see directory entries, not individual files
	s.Contains(keys, "certs/site1.com")
	s.Contains(keys, "certs/site2.com")
}

// Test_List_Recursive tests listing directory contents recursively
func (s *CertmagicRedisSuite) Test_List_Recursive() {
	// Store test files in multiple subdirectories
	files := map[string][]byte{
		"certs/site1.com/cert.pem": []byte("cert1"),
		"certs/site1.com/key.pem":  []byte("key1"),
		"certs/site2.com/cert.pem": []byte("cert2"),
	}

	for key, value := range files {
		err := s.storage.Store(s.ctx, key, value)
		s.NoError(err)
	}

	// List files in certs directory (recursive)
	keys, err := s.storage.List(s.ctx, "certs", true)
	s.NoError(err)
	s.Len(keys, 3)

	// Should see all individual files
	s.Contains(keys, "certs/site1.com/cert.pem")
	s.Contains(keys, "certs/site1.com/key.pem")
	s.Contains(keys, "certs/site2.com/cert.pem")
}

// Test_List_EmptyDirectory tests listing an empty directory
func (s *CertmagicRedisSuite) Test_List_EmptyDirectory() {
	keys, err := s.storage.List(s.ctx, "empty/dir", false)
	s.NoError(err)
	s.Empty(keys)
}

// Test_Stat tests getting file metadata
func (s *CertmagicRedisSuite) Test_Stat() {
	key := "certs/stat-test/cert.pem"
	value := []byte("certificate data for stat test")

	// Store data first
	beforeStore := time.Now()
	err := s.storage.Store(s.ctx, key, value)
	s.NoError(err)
	afterStore := time.Now()

	// Get stat info
	info, err := s.storage.Stat(s.ctx, key)
	s.NoError(err)

	s.Equal(key, info.Key)
	s.Equal(int64(len(value)), info.Size)
	s.True(info.IsTerminal)
	s.False(info.Modified.IsZero())

	// Modified time should be between before and after store
	s.True(info.Modified.After(beforeStore.Add(-time.Second)))
	s.True(info.Modified.Before(afterStore.Add(time.Second)))
}

// Test_Stat_NotFound tests stat on non-existent key
func (s *CertmagicRedisSuite) Test_Stat_NotFound() {
	key := "nonexistent/stat/key"

	_, err := s.storage.Stat(s.ctx, key)
	s.Error(err)
	s.ErrorIs(err, fs.ErrNotExist)
}

// Test_Lock_And_Unlock tests basic lock and unlock
func (s *CertmagicRedisSuite) Test_Lock_And_Unlock() {
	name := "test-lock"

	// Acquire lock
	err := s.storage.Lock(s.ctx, name)
	s.NoError(err)

	// Release lock
	err = s.storage.Unlock(s.ctx, name)
	s.NoError(err)
}

// Test_Lock_Multiple tests locking different resources
func (s *CertmagicRedisSuite) Test_Lock_Multiple() {
	lock1 := "lock-1"
	lock2 := "lock-2"

	// Acquire both locks
	err := s.storage.Lock(s.ctx, lock1)
	s.NoError(err)

	err = s.storage.Lock(s.ctx, lock2)
	s.NoError(err)

	// Release both locks
	err = s.storage.Unlock(s.ctx, lock1)
	s.NoError(err)

	err = s.storage.Unlock(s.ctx, lock2)
	s.NoError(err)
}

// Test_Lock_Concurrent tests concurrent lock acquisition
func (s *CertmagicRedisSuite) Test_Lock_Concurrent() {
	name := "concurrent-lock"

	// First goroutine acquires lock
	err := s.storage.Lock(s.ctx, name)
	s.NoError(err)

	// Second goroutine tries to acquire same lock with timeout
	ctx2, cancel := context.WithTimeout(s.ctx, 500*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.storage.Lock(ctx2, name)
	}()

	// Wait a bit to ensure second goroutine is blocked
	time.Sleep(100 * time.Millisecond)

	// Second goroutine should still be waiting
	select {
	case <-done:
		s.Fail("Second lock should not have been acquired yet")
	default:
		// Expected - lock is still held
	}

	// Release first lock
	err = s.storage.Unlock(s.ctx, name)
	s.NoError(err)

	// Now second goroutine should timeout (context expired)
	err = <-done
	s.Error(err)
	s.ErrorIs(err, context.DeadlineExceeded)
}

// Test_Lock_Sequential tests sequential lock acquisition
func (s *CertmagicRedisSuite) Test_Lock_Sequential() {
	name := "sequential-lock"

	// Acquire and release lock first time
	err := s.storage.Lock(s.ctx, name)
	s.NoError(err)

	err = s.storage.Unlock(s.ctx, name)
	s.NoError(err)

	// Should be able to acquire again
	err = s.storage.Lock(s.ctx, name)
	s.NoError(err)

	err = s.storage.Unlock(s.ctx, name)
	s.NoError(err)
}

// Test_Unlock_NonExistent tests unlocking a lock that was never acquired
func (s *CertmagicRedisSuite) Test_Unlock_NonExistent() {
	name := "never-locked"

	// Should not error when unlocking non-existent lock
	err := s.storage.Unlock(s.ctx, name)
	s.NoError(err)
}

// Test_Repair tests the repair functionality
func (s *CertmagicRedisSuite) Test_Repair() {
	// Store some test data
	files := map[string][]byte{
		"certs/repair1/cert.pem": []byte("cert1"),
		"certs/repair2/cert.pem": []byte("cert2"),
		"certs/repair2/key.pem":  []byte("key2"),
	}

	for key, value := range files {
		err := s.storage.Store(s.ctx, key, value)
		s.NoError(err)
	}

	// Run repair on subdirectory
	err := s.storage.Repair(s.ctx, "certs")
	s.NoError(err)

	// Verify data is still accessible
	for key := range files {
		s.True(s.storage.Exists(s.ctx, key))
	}

	// Verify list still works
	keys, err := s.storage.List(s.ctx, "certs", true)
	s.NoError(err)
	s.Len(keys, 3)
}

// Test_Repair_Root tests repair from root directory
func (s *CertmagicRedisSuite) Test_Repair_Root() {
	// Store some test data
	files := map[string][]byte{
		"root/level1/file1.pem": []byte("data1"),
		"root/level1/file2.pem": []byte("data2"),
	}

	for key, value := range files {
		err := s.storage.Store(s.ctx, key, value)
		s.NoError(err)
	}

	// Run repair from root (empty string)
	err := s.storage.Repair(s.ctx, "")
	s.NoError(err)

	// Verify data is still accessible
	for key := range files {
		s.True(s.storage.Exists(s.ctx, key))
	}
}

// Test_StoreAndLoad_LargeData tests storing and loading large data
func (s *CertmagicRedisSuite) Test_StoreAndLoad_LargeData() {
	key := "certs/large/data.pem"

	// Create a large value (1MB)
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	// Store large data
	err := s.storage.Store(s.ctx, key, largeValue)
	s.NoError(err)

	// Load it back
	loadedValue, err := s.storage.Load(s.ctx, key)
	s.NoError(err)
	s.Equal(largeValue, loadedValue)

	// Check size via Stat
	info, err := s.storage.Stat(s.ctx, key)
	s.NoError(err)
	s.Equal(int64(len(largeValue)), info.Size)
}

// Test_StoreAndList_NestedDirectories tests deeply nested directory structures
func (s *CertmagicRedisSuite) Test_StoreAndList_NestedDirectories() {
	files := map[string][]byte{
		"deep/level1/level2/level3/level4/file.pem": []byte("deeply nested"),
		"deep/level1/level2/file2.pem":              []byte("less nested"),
		"deep/level1/file1.pem":                     []byte("shallow"),
	}

	for key, value := range files {
		err := s.storage.Store(s.ctx, key, value)
		s.NoError(err)
	}

	// List recursively from root
	keys, err := s.storage.List(s.ctx, "deep", true)
	s.NoError(err)
	s.Len(keys, 3)

	// List non-recursively at different levels
	keys, err = s.storage.List(s.ctx, "deep/level1", false)
	s.NoError(err)
	s.Contains(keys, "deep/level1/file1.pem")
	s.Contains(keys, "deep/level1/level2")
}

// Test_StoreAndDelete_MultipleTimes tests storing and deleting the same key multiple times
func (s *CertmagicRedisSuite) Test_StoreAndDelete_MultipleTimes() {
	key := "certs/repeated/file.pem"

	for i := 0; i < 3; i++ {
		value := []byte("iteration " + string(rune('0'+i)))

		// Store
		err := s.storage.Store(s.ctx, key, value)
		s.NoError(err)
		s.True(s.storage.Exists(s.ctx, key))

		// Load and verify
		loaded, err := s.storage.Load(s.ctx, key)
		s.NoError(err)
		s.Equal(value, loaded)

		// Delete
		err = s.storage.Delete(s.ctx, key)
		s.NoError(err)
		s.False(s.storage.Exists(s.ctx, key))
	}
}

func TestCertmagicRedisSuite(t *testing.T) {
	suite.Run(t, &CertmagicRedisSuite{})
}
