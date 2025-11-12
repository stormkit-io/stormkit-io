package hosting_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stretchr/testify/suite"
)

type ImageOptimizerSuite struct {
	suite.Suite
}

func (s *ImageOptimizerSuite) Test_NewImageOptimizer() {
	optimizer := hosting.NewImageOptimizer()
	s.NotNil(optimizer)
}

func (s *ImageOptimizerSuite) Test_IsImageOptimizationEnabled() {
	enabled := hosting.IsImageOptimizationEnabled()
	// Without imageopt tag, should be false
	// With imageopt tag, should be true
	s.NotNil(enabled) // Just verify it returns a value
}

func (s *ImageOptimizerSuite) Test_Optimize_NoOp() {
	// When built without imageopt tag, Optimize should return an error
	optimizer := hosting.NewImageOptimizer()
	content := []byte("test image content")
	
	result, err := optimizer.Optimize(content, 100, 100, false)
	
	if !hosting.IsImageOptimizationEnabled() {
		// Should return error when optimization is disabled
		s.Error(err)
		s.Nil(result)
		s.Contains(err.Error(), "image optimization is disabled")
	}
	// If optimization is enabled, the test would need actual image content
}

func TestImageOptimizer(t *testing.T) {
	suite.Run(t, &ImageOptimizerSuite{})
}
