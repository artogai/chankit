package chankit

import (
	"context"
	"sync"
)

type Pipeline struct {
	cancel context.CancelFunc

	wg  sync.WaitGroup
	err error
	mu  sync.Mutex
}

func NewPipeline(ctx context.Context) (*Pipeline, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Pipeline{cancel: cancel}, ctx
}

func (p *Pipeline) Wait() error {
	p.wg.Wait()
	p.cancel()
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

func (p *Pipeline) goSafe(fn func() error) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := fn(); err != nil {
			p.mu.Lock()
			if p.err == nil { // first one wins
				p.err = err
				p.cancel()
			}
			p.mu.Unlock()
		}
	}()
}
