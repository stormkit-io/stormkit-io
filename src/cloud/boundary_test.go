package cloud_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

const cloudPkg = "github.com/stormkit-io/stormkit-io/src/cloud"

// BoundarySuite keeps Stormkit Cloud's control-plane code out of the product:
// self-hosted builds (and the customer instances Cloud provisions, which run
// them) must never compile in anything under src/cloud.
type BoundarySuite struct {
	suite.Suite
}

func (s *BoundarySuite) Test_ProductDoesNotImportCloud() {
	gomod, err := exec.Command("go", "env", "GOMOD").Output()
	s.Require().NoError(err)

	cmd := exec.Command("go", "list", "-tags=imageopt,alibaba", "-deps", "-f", "{{.ImportPath}}",
		"./src/ce/...", "./src/ee/...", "./src/lib/...", "./src/migrations/...")
	cmd.Dir = filepath.Dir(strings.TrimSpace(string(gomod)))

	out, err := cmd.Output()
	s.Require().NoError(err)

	for pkg := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		s.False(pkg == cloudPkg || strings.HasPrefix(pkg, cloudPkg+"/"), "product code depends on %s", pkg)
	}
}

func TestBoundarySuite(t *testing.T) {
	suite.Run(t, new(BoundarySuite))
}
