package chankit

import (
	"context"
	"sync"
)

func Map[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	fn func(context.Context, A) (B, error),
	opts ...Option,
) <-chan B {
	cfg := makeConfig(opts)
	out := make(chan B, cfg.bufCap)

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

	return out
}

func MapConcUnordered[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	parN int,
	fn func(context.Context, A) (B, error),
	opts ...Option,
) <-chan B {
	if parN <= 0 {
		panic("parN must be > 0")
	}

	cfg := makeConfig(opts)
	out := make(chan B, cfg.bufCap)

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

	return out
}

func MapConcOrdered[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	parN int,
	fn func(context.Context, A) (B, error),
	opts ...Option,
) <-chan B {
	if parN <= 0 {
		panic("parN must be > 0")
	}

	cfg := makeConfig(opts)
	out := make(chan B, cfg.bufCap)

	type job struct {
		idx int64
		val A
	}

	type res struct {
		idx int64
		val B
	}

	sem := make(chan struct{}, parN*4)
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
		defer close(out)

		next := int64(0)
		buffer := make(map[int64]B, parN)

		emit := func(v B) {
			out <- v
			<-sem
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
						emit(v)
						delete(buffer, next)
						next++
					}
				}
				if res.idx == next {
					emit(res.val)
					next++
					for {
						v, ok := buffer[next]
						if !ok {
							break
						}
						emit(v)
						delete(buffer, next)
						next++
					}
				} else {
					buffer[res.idx] = res.val
				}
			}
		}
	})

	return out
}
