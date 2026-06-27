package hosting

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type CollectSanitizeSuite struct {
	suite.Suite
}

func (s *CollectSanitizeSuite) Test_stripNullChars() {
	s.Equal("abc", stripNullChars("abc"))
	s.Equal("ab", stripNullChars(string([]byte{'a', 0, 'b'})))
	s.Equal("ab", stripNullChars(string([]byte{'a', 'b', 0})))
	s.Equal("", stripNullChars(string([]byte{0})))
}

func (s *CollectSanitizeSuite) Test_sanitizeRequestID() {
	valid := "11111111-1111-1111-1111-111111111111"

	s.Equal(null.StringFrom(valid), sanitizeRequestID(valid))
	s.False(sanitizeRequestID("").Valid)
	s.False(sanitizeRequestID("not-a-uuid").Valid)
	s.False(sanitizeRequestID("'; DROP TABLE analytics_events;--").Valid)
}

func (s *CollectSanitizeSuite) Test_sanitizeMetadata() {
	valid := sanitizeMetadata(json.RawMessage(`{"price": 42}`))
	s.True(valid.Valid)
	s.JSONEq(`{"price": 42}`, valid.String)

	s.False(sanitizeMetadata(nil).Valid)
	s.False(sanitizeMetadata(json.RawMessage{}).Valid)

	// A JSON unicode NUL escape is valid JSON but rejected by Postgres jsonb.
	nullEscape := json.RawMessage(append([]byte{'"'}, append([]byte{'\\', 'u', '0', '0', '0', '0'}, '"')...))
	s.False(sanitizeMetadata(nullEscape).Valid)

	// A raw NUL byte cannot be stored in jsonb either.
	rawNull := json.RawMessage([]byte{'"', 'a', 0, 'b', '"'})
	s.False(sanitizeMetadata(rawNull).Valid)
}

func TestCollectSanitizeSuite(t *testing.T) {
	suite.Run(t, new(CollectSanitizeSuite))
}
