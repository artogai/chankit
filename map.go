package chankit

import (
	"context"
	"sync"
)

func Map[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	fn func(A) B,
	opts ...Option,
) <-chan B {
	return mapImpl(
		ctx,
		p,
		in,
		func(_ context.Context, a A) (B, error) { return fn(a), nil },
		opts...)
}

func MapErr[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	fn func(A) (B, error),
	opts ...Option,
) <-chan B {
	return mapImpl(ctx, p, in, func(_ context.Context, a A) (B, error) { return fn(a) }, opts...)
}

func MapErrCtx[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	fn func(context.Context, A) (B, error),
	opts ...Option,
) <-chan B {
	return mapImpl(ctx, p, in, fn, opts...)
}

func mapImpl[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	fn func(context.Context, A) (B, error),
	opts ...Option,
) <-chan B {
	cfg := makeConfig(opts)
	out := make(chan B, cfg.bufCap)

	switch {
	case cfg.parOpt.n < 0:
		panic("parallelism < 0")
	case cfg.parOpt.n == 0:
		sequentialMapImpl(ctx, p, in, out, fn)
	case cfg.parOpt.n == 1:
		concUnorderedMapImpl(ctx, p, in, out, fn, 1)
	default:
		if cfg.parOpt.unordered {
			concUnorderedMapImpl(ctx, p, in, out, fn, cfg.parOpt.n)
		} else {
			concOrderedMapImpl(ctx, p, in, out, fn, &cfg.parOpt)
		}
	}

	return out
}

func sequentialMapImpl[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	out chan<- B,
	fn func(context.Context, A) (B, error),
) {
	p.goSafe(func() error {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case a, ok := <-in:
				if !ok {
					return nil
				}

				b, err := fn(ctx, a)
				if err != nil {
					return err
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- b:
				}
			}
		}
	})
}

func concUnorderedMapImpl[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	out chan<- B,
	fn func(context.Context, A) (B, error),
	parN int,
) {
	var wg sync.WaitGroup
	for range parN {
		wg.Add(1)
		p.goSafe(func() error {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case a, ok := <-in:
					if !ok {
						return nil
					}

					b, err := fn(ctx, a)
					if err != nil {
						return err
					}

					select {
					case <-ctx.Done():
						return ctx.Err()
					case out <- b:
					}
				}
			}
		})
	}

	p.goSafe(func() error {
		wg.Wait()
		close(out)
		return nil
	})
}

func concOrderedMapImpl[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	out chan<- B,
	fn func(context.Context, A) (B, error),
	parOpt *parOpt,
) {
	type job struct {
		idx int64
		val A
	}

	type res struct {
		idx int64
		val B
	}

	parN := parOpt.n
	semCap := parOpt.reorderWindow
	if semCap <= 0 { // default
		semCap = 4 * max(parOpt.n, 1)
	}
	sem := make(chan struct{}, semCap)
	jobCh := make(chan job, parN)
	resCh := make(chan res, parN)

	p.goSafe(func() error {
		defer close(jobCh)
		for idx := int64(0); ; idx++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case a, ok := <-in:
				if !ok {
					return nil
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case jobCh <- job{idx, a}:
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case sem <- struct{}{}: // blocks when gap >= window
				}
			}
		}
	})

	var wg sync.WaitGroup
	for range parN {
		wg.Add(1)
		p.goSafe(func() error {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case job, ok := <-jobCh:
					if !ok {
						return nil
					}

					b, err := fn(ctx, job.val)
					if err != nil {
						return err
					}

					select {
					case <-ctx.Done():
						return ctx.Err()
					case resCh <- res{job.idx, b}:
					}
				}
			}
		})
	}

	p.goSafe(func() error {
		wg.Wait()
		close(resCh)
		return nil
	})

	p.goSafe(func() error {
		next := int64(0)
		buffer := make(map[int64]B, parN)

		defer close(out)
		defer func() {
			for k := range buffer {
				delete(buffer, k)
			}
		}()

		emit := func(ctx context.Context, v B) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- v:
				<-sem
				return nil
			}
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-resCh:
				if !ok {
					for {
						v, ok := buffer[next]
						if !ok {
							return nil
						}
						if err := emit(ctx, v); err != nil {
							return err
						}
						delete(buffer, next)
						next++
					}
				}
				if res.idx == next {
					if err := emit(ctx, res.val); err != nil {
						return err
					}
					next++
					for {
						v, ok := buffer[next]
						if !ok {
							break
						}
						if err := emit(ctx, v); err != nil {
							return err
						}
						delete(buffer, next)
						next++
					}
				} else {
					buffer[res.idx] = res.val
				}
			}
		}
	})
}
