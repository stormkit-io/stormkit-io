//go:build !imageopt

package hosting

import "github.com/stormkit-io/stormkit-io/src/lib/slog"

func init() {
	slog.Debug(slog.LogOpts{
		Msg:   "image optimization is disabled, returning original content. Check docs/IMAGE_OPTIMIZATION.md for enabling instructions.",
		Level: slog.DL1,
	})
}

// ImageOptimizer defines the interface for image optimization operations
type ImageOptimizer interface {
	Optimize(content []byte, width, height int, smart bool) ([]byte, error)
}

// NoopOptimizer is a no-op implementation of ImageOptimizer
type NoopOptimizer struct{}

// NewImageOptimizer creates a new image optimizer instance (no-op version)
func NewImageOptimizer() ImageOptimizer {
	return &NoopOptimizer{}
}

// Optimize returns an error indicating image optimization is disabled
func (o *NoopOptimizer) Optimize(content []byte, width, height int, smart bool) ([]byte, error) {
	return content, nil
}

// IsImageOptimizationEnabled returns false when image optimization is disabled
func IsImageOptimizationEnabled() bool {
	return false
}
