package hosting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ParseAcceptBoundsSuite covers the caps that keep a hostile Accept header from
// turning one request into tens of milliseconds of CPU and tens of megabytes of
// allocation on a shared edge (issue #478). parseAccept is unexported, so these
// live in an internal test.
type ParseAcceptBoundsSuite struct {
	suite.Suite
}

func (s *ParseAcceptBoundsSuite) Test_NormalHeaderParsesEveryEntry() {
	pref := parseAccept("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	s.False(pref.empty)
	s.Len(pref.types, 4)
}

func (s *ParseAcceptBoundsSuite) Test_OversizedHeaderIsTreatedAsAbsent() {
	pref := parseAccept(strings.Repeat("*/*,", (1<<20)/4))

	s.True(pref.empty, "an oversized header must degrade to no preference")
	s.Empty(pref.types)
	// The safe default: everything matches, so HTML is served.
	s.False(pref.prefersMarkdown())
}

func (s *ParseAcceptBoundsSuite) Test_CommaBombIsTreatedAsAbsent() {
	pref := parseAccept(strings.Repeat(",", 1<<20))

	s.True(pref.empty)
	s.Empty(pref.types)
}

func (s *ParseAcceptBoundsSuite) Test_EntryCountIsCappedBelowTheLengthLimit() {
	// Many short entries stay under maxAcceptHeaderLen but must still not parse
	// past maxAcceptEntries.
	header := strings.TrimSuffix(strings.Repeat("a/b,", maxAcceptEntries+50), ",")
	s.LessOrEqual(len(header), maxAcceptHeaderLen)

	pref := parseAccept(header)

	s.False(pref.empty)
	s.Len(pref.types, maxAcceptEntries)
}

func (s *ParseAcceptBoundsSuite) Test_HeaderAtTheLengthLimitStillParses() {
	one := "text/html,"
	repeats := maxAcceptHeaderLen / len(one)
	header := strings.TrimSuffix(strings.Repeat(one, repeats), ",")
	s.LessOrEqual(len(header), maxAcceptHeaderLen)

	pref := parseAccept(header)

	s.False(pref.empty)
}

func TestParseAcceptBoundsSuite(t *testing.T) {
	suite.Run(t, new(ParseAcceptBoundsSuite))
}
