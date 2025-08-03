package chankit

import (
	"context"
	"sync"
	"sync/atomic"
)

func Merge[A any](
	ctx context.Context,
	p *Pipeline,
	leftIn <-chan A,
	rightIn <-chan A,
	opts ...Option,
) <-chan A {
	cfg := makeConfig(opts)
	out := make(chan A, cfg.bufCap)

	mergeCtx, mergeCancel := context.WithCancel(ctx)

	var wg sync.WaitGroup

	var leftDone, rightDone atomic.Bool

	shouldStop := func() bool {
		switch cfg.haltStrategy {
		case HaltLeft:
			return leftDone.Load()
		case HaltRight:
			return rightDone.Load()
		case HaltEither:
			return leftDone.Load() || rightDone.Load()
		case HaltBoth:
			return leftDone.Load() && rightDone.Load()
		default:
			return false
		}
	}

	mergeOne := func(isLeft bool, in <-chan A) {
		wg.Add(1)
		p.goSafe(func() error {
			defer wg.Done()

			for {
				select {
				case <-mergeCtx.Done():
					return mergeCtx.Err()
				case a, ok := <-in:
					if !ok {
						if isLeft {
							leftDone.Store(true)
						} else {
							rightDone.Store(true)
						}
						if shouldStop() {
							mergeCancel() // signal to other goroutine to stop
						}
						return nil
					}

					select {
					case <-mergeCtx.Done():
						return mergeCtx.Err()
					case out <- a:
					}
				}
			}
		})
	}

	mergeOne(true, leftIn)
	mergeOne(false, rightIn)

	go func() {
		wg.Wait()
		mergeCancel()
		close(out)
	}()

	return out
}
