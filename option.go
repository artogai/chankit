package chankit

import (
	"context"
	"runtime"
)

type Option func(*config)

func WithBuffer(n int) Option {
	return func(c *config) {
		c.bufCap = max(n, 0)
	}
}

func WithParallel(n ...int) Option {
	num := runtime.NumCPU()
	if len(n) > 0 {
		num = n[0]
	}

	return func(c *config) {
		c.parOpt.n = max(num, 0)
	}
}

func WithUnordered() Option {
	return func(c *config) {
		c.parOpt.unordered = true
	}
}

func WithUpstreamCancel(cf context.CancelFunc) Option {
	return func(c *config) { c.upstreamCancel = cf }
}

// should be rarely used
func WithReorderWindow(maxGap int) Option {
	return func(c *config) {
		c.parOpt.reorderWindow = max(maxGap, 0)
	}
}

type parOpt struct {
	n             int
	unordered     bool
	reorderWindow int
}

type config struct {
	bufCap         int
	parOpt         parOpt
	upstreamCancel context.CancelFunc
}

func makeConfig(opts []Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
