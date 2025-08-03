package chankit

import (
	"context"
	"sync/atomic"
	"time"
)

type ProducerOpt func(*producerOpts)

type producerOpts struct {
	Start       int
	Step        int
	Limit       int
	Delay       time.Duration
	ProducedCnt *int32
}

func defaultProducerOpts() producerOpts {
	return producerOpts{
		Start: 0, Step: 1, Limit: -1, Delay: 0,
	}
}

func WithStart(n int) ProducerOpt { return func(o *producerOpts) { o.Start = n } }

func WithStep(step int) ProducerOpt { return func(o *producerOpts) { o.Step = step } }

func WithLimit(n int) ProducerOpt { return func(o *producerOpts) { o.Limit = n } }

func WithDelay(d time.Duration) ProducerOpt { return func(o *producerOpts) { o.Delay = d } }

func WithCounter(cnt *int32) ProducerOpt { return func(o *producerOpts) { o.ProducedCnt = cnt } }

func CreateProducer(ctx context.Context, opts ...ProducerOpt) <-chan int {
	cfg := defaultProducerOpts()
	for _, opt := range opts {
		opt(&cfg)
	}

	out := make(chan int)

	go func() {
		defer close(out)
		val := cfg.Start
		sent := 0
		for cfg.Limit <= 0 || sent < cfg.Limit {
			select {
			case <-ctx.Done():
				return
			case out <- val:
				if cfg.ProducedCnt != nil {
					atomic.AddInt32(cfg.ProducedCnt, 1)
				}
				if cfg.Delay > 0 {
					time.Sleep(cfg.Delay)
				}
				val += cfg.Step
				sent++
			}
		}
	}()

	return out
}
