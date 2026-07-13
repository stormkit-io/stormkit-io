package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type Example struct {
	ID types.ID `json:"id"`
}

type TypeIDSuite struct {
	suite.Suite
}

func (s *TypeIDSuite) Test_Marshaling() {
	b, err := json.Marshal(Example{ID: 1})

	s.NoError(err)
	s.JSONEq(`{ "id": "1" }`, string(b))

	b, err = json.Marshal([]Example{
		{ID: 1},
		{ID: 2},
	})

	s.NoError(err)
	s.JSONEq(`[{ "id": "1"}, {"id": "2" }]`, string(b))
}

func (s *TypeIDSuite) Test_Unmarshaling() {
	examples := []Example{}
	err := json.Unmarshal([]byte(`[{ "id": "1"}, {"id": "2" }]`), &examples)

	s.NoError(err)
	s.Equal([]Example{{ID: 1}, {ID: 2}}, examples)

	example := Example{}
	err = json.Unmarshal([]byte(`{ "id": "1"}`), &example)

	s.NoError(err)
	s.Equal(Example{ID: 1}, example)

	example = Example{}
	err = json.Unmarshal([]byte(`{ "id": 1}`), &example)

	s.NoError(err)
	s.Equal(Example{ID: 1}, example)
}

func TestTypeID(t *testing.T) {
	suite.Run(t, &TypeIDSuite{})
}
