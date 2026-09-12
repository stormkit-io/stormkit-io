package integrations

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FileCacheSuite struct {
	suite.Suite
	cache *fileCache
}

func (s *FileCacheSuite) SetupTest() {
	s.cache = &fileCache{
		budget:  100,
		maxItem: 40,
		entries: map[string]*list.Element{},
		order:   list.New(),
	}
}

func (s *FileCacheSuite) file(size int) *GetFileResult {
	return &GetFileResult{Content: make([]byte, size), Size: int64(size)}
}

func (s *FileCacheSuite) Test_StoresAndReturnsTheSameValue() {
	stored := s.file(10)
	s.cache.put("a", stored)

	got, ok := s.cache.get("a")

	s.True(ok)
	s.Same(stored, got, "callers share the entry rather than getting a copy")
}

func (s *FileCacheSuite) Test_MissIsNotAnError() {
	got, ok := s.cache.get("absent")

	s.False(ok)
	s.Nil(got)
}

func (s *FileCacheSuite) Test_RefusesFilesOverThePerItemCap() {
	s.cache.put("big", s.file(41))

	_, ok := s.cache.get("big")

	s.False(ok, "one large file must not be able to push out the working set")
}

func (s *FileCacheSuite) Test_StaysInsideTheBudget() {
	for _, key := range []string{"a", "b", "c", "d"} {
		s.cache.put(key, s.file(30))
	}

	s.LessOrEqual(s.cache.used, s.cache.budget)
	s.LessOrEqual(int64(len(s.cache.entries)), s.cache.budget/30)
}

func (s *FileCacheSuite) Test_EvictsLeastRecentlyUsed() {
	s.cache.put("a", s.file(30))
	s.cache.put("b", s.file(30))
	s.cache.put("c", s.file(30))

	// Touching "a" makes "b" the coldest.
	s.cache.get("a")
	s.cache.put("d", s.file(30))

	_, hasA := s.cache.get("a")
	_, hasB := s.cache.get("b")

	s.True(hasA, "recently used entries survive")
	s.False(hasB, "the coldest entry is the one evicted")
}

func (s *FileCacheSuite) Test_ReplacingAKeyDoesNotDoubleCount() {
	s.cache.put("a", s.file(30))
	s.cache.put("a", s.file(10))

	s.Equal(int64(10), s.cache.used)
}

func (s *FileCacheSuite) Test_ZeroBudgetDisablesIt() {
	disabled := &fileCache{entries: map[string]*list.Element{}, order: list.New()}

	disabled.put("a", s.file(1))
	_, ok := disabled.get("a")

	s.False(ok, "a zero budget is how the cache is turned off without a release")
}

func TestFileCache(t *testing.T) {
	suite.Run(t, &FileCacheSuite{})
}
