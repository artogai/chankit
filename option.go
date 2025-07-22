package chankit

type Option func(*config)

func WithBuffer(n int) Option {
	return func(c *config) {
		c.bufCap = max(n, 0)
	}
}

type config struct {
	bufCap int
}

func defaultConfig() config {
	return config{
		bufCap: 0,
	}
}

func makeConfig(opts []Option) config {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
