package runner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stretchr/testify/suite"
)

type ReporterSuite struct {
	suite.Suite
}

// Test_SignUploads verifies that the runner asks the deploy callback for the
// upload forms and reads them from the answer.
func (s *ReporterSuite) Test_SignUploads() {
	received := map[string]any{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)

		_ = json.NewEncoder(w).Encode(integrations.SignedUploads{
			Artifacts: map[string]integrations.SignedArtifact{"sk-client.zip": {Location: "hcloud:artifacts/1/2/sk-client.zip"}},
		})
	}))

	defer server.Close()

	uploads, err := NewReporter(server.URL).SignUploads()

	s.Require().NoError(err)
	s.Equal(true, received["signUploads"])
	s.Equal("hcloud:artifacts/1/2/sk-client.zip", uploads.Artifacts["sk-client.zip"].Location)
}

func (s *ReporterSuite) Test_SignUploads_NoCallback() {
	_, err := NewReporter("").SignUploads()

	s.ErrorContains(err, "no deployment callback")
}

func TestReporterSuite(t *testing.T) {
	suite.Run(t, new(ReporterSuite))
}
