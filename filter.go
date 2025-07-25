package chankit

import "context"

// Filter forwards elements from in that satisfy pred.
//
// It closes the returned channel after input is fully consumed.
// If ctx is canceled, it stops early and returns
func Filter[A any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	pred func(A) bool,
	opts ...Option,
) <-chan A {
	return FilterErrCtx(
		ctx,
		p,
		in,
		func(_ context.Context, a A) (bool, error) { return pred(a), nil },
		opts...)
}

// FilterErr forwards elements from in that satisfy pred.
//
// If pred returns an error, the pipeline fails and no more elements are processed.
// It closes the returned channel after input is fully consumed or on error.
// If ctx is canceled, it stops early and returns
func FilterErr[A any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	pred func(A) (bool, error),
	opts ...Option,
) <-chan A {
	return FilterErrCtx(
		ctx,
		p,
		in,
		func(_ context.Context, a A) (bool, error) { return pred(a) },
		opts...)
}

// FilterErrCtx forwards elements from in that satisfy pred.
//
// The predicate receives the context, which may include timeouts or cancellation.
// If pred returns an error, the pipeline fails and no more elements are processed.
// It closes the returned channel after input is fully consumed or on error.
// If ctx is canceled, it stops early and returns
func FilterErrCtx[A any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	pred func(context.Context, A) (bool, error),
	opts ...Option,
) <-chan A {
	cfg := makeConfig(opts)
	out := make(chan A, cfg.bufCap)

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

				p, err := pred(ctx, a)
				if err != nil {
					return err
				}

				if p {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case out <- a:
					}
				}
			}
		}
	})

	return out
}

// Take forwards at most n elements from in, then:
//
//   - if an upstream-cancel func was supplied via WithUpstreamCancel,
//     it calls that func so the producer branch stops immediately.
//     Only the producer branch stops; downstream stages continue to consume
//     the values that are already in flight.
//   - otherwise it drains in until the producer closes.
//
// In both cases it closes the returned channel. n <= 0 means "take none"
func Take[A any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	n int,
	opts ...Option,
) <-chan A {
	cfg := makeConfig(opts)
	out := make(chan A, cfg.bufCap)

	if n <= 0 {
		if cfg.upstreamCancel != nil {
			cfg.upstreamCancel()
		}
		close(out)
		return out
	}

	p.goSafe(func() error {
		defer close(out)

		taken := 0
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case v, ok := <-in:
				if !ok {
					return nil
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- v:
				}

				taken++

				if taken == n {
					if cfg.upstreamCancel != nil {
						cfg.upstreamCancel()
					} else {
						p.goSafe(func() error {
							for {
								select {
								case <-ctx.Done():
									return ctx.Err()
								case _, ok := <-in:
									if !ok {
										return nil
									}
								}
							}
						})
					}

					return nil
				}
			}
		}
	})

	return out
}
