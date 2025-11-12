//go:build imageopt

package hosting

import (
	"github.com/h2non/bimg"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

func init() {
	slog.Debug(slog.LogOpts{
		Msg:   "image optimization is enabled",
		Level: slog.DL1,
	})
}

// ImageOptimizer defines the interface for image optimization operations
type ImageOptimizer interface {
	Optimize(content []byte, width, height int, smart bool) ([]byte, error)
}

// BimgOptimizer is the bimg-based implementation of ImageOptimizer
type BimgOptimizer struct{}

// NewImageOptimizer creates a new image optimizer instance
func NewImageOptimizer() ImageOptimizer {
	return &BimgOptimizer{}
}

// Optimize optimizes an image using bimg
func (o *BimgOptimizer) Optimize(content []byte, width, height int, smart bool) ([]byte, error) {
	image := bimg.NewImage(content)

	var optimized []byte
	var err error

	if smart {
		optimized, err = image.SmartCrop(width, height)
	} else {
		optimized, err = image.ForceResize(width, height)
	}

	return optimized, err
}

// IsImageOptimizationEnabled returns true if image optimization is enabled
func IsImageOptimizationEnabled() bool {
	return true
}
