package chankit

import (
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

// should be rarely used
func WithReorderWindow(maxGap int) Option {
	return func(c *config) {
		c.parOpt.reorderWindow = max(maxGap, 0)
	}
}

type HaltStrategy int

const (
	HaltBoth   HaltStrategy = iota // wait for both sides (default)
	HaltLeft                       // stop when left side finishes
	HaltRight                      // stop when right side finishes
	HaltEither                     // stop when either side finishes
)

func (h HaltStrategy) String() string {
	switch h {
	case HaltBoth:
		return "HaltBoth"
	case HaltLeft:
		return "HaltLeft"
	case HaltRight:
		return "HaltRight"
	case HaltEither:
		return "HaltEither"
	default:
		return "UnknownHaltStrategy"
	}
}

func WithHaltStrategy(s HaltStrategy) Option {
	return func(c *config) {
		c.haltStrategy = s
	}
}

type parOpt struct {
	n             int
	unordered     bool
	reorderWindow int
}

type config struct {
	bufCap       int
	parOpt       parOpt
	haltStrategy HaltStrategy
}

func makeConfig(opts []Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
