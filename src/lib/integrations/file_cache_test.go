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

func (s *FileCacheSuite) Test_DropDeploymentRemovesOnlyThatDeployment() {
	s.cache.put("10:/index.html", s.file(10))
	s.cache.put("10:/app.js", s.file(10))
	s.cache.put("11:/index.html", s.file(10))

	s.cache.dropDeployment("10")

	_, hasOld := s.cache.get("10:/index.html")
	_, hasOldTwo := s.cache.get("10:/app.js")
	_, hasOther := s.cache.get("11:/index.html")

	s.False(hasOld)
	s.False(hasOldTwo)
	s.True(hasOther, "a different deployment keeps its entries")
}

func (s *FileCacheSuite) Test_DropDeploymentReclaimsTheBudget() {
	s.cache.put("10:/index.html", s.file(30))
	s.cache.put("11:/index.html", s.file(30))

	s.cache.dropDeployment("10")

	s.Equal(int64(30), s.cache.used, "the freed bytes go back to the budget")
}

// Prefixes must not be confused: dropping 1 must not touch 10.
func (s *FileCacheSuite) Test_DropDeploymentDoesNotMatchIdPrefixes() {
	s.cache.put("1:/index.html", s.file(10))
	s.cache.put("10:/index.html", s.file(10))

	s.cache.dropDeployment("1")

	_, hasOne := s.cache.get("1:/index.html")
	_, hasTen := s.cache.get("10:/index.html")

	s.False(hasOne)
	s.True(hasTen, "deployment 10 is not deployment 1")
}

func TestFileCache(t *testing.T) {
	suite.Run(t, &FileCacheSuite{})
}
