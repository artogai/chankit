package chankit

import "context"

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
