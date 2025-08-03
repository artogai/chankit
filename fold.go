package chankit

import "context"

// Fold reduces elements from `in“ into an accumulator using `f“.
//
// When the input channel closes without error, the final accumulator is sent
// and the returned channel is closed.
// If ctx is canceled, it stops early and returns.
func Fold[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	init B,
	f func(B, A) B,
) <-chan B {
	return FoldErrCtx(
		ctx,
		p,
		in,
		init,
		func(_ context.Context, acc B, a A) (B, error) { return f(acc, a), nil },
	)
}

// FoldErr reduces elements from `in“ into an accumulator using `f“.
//
// When the input channel closes without error, the final accumulator is sent
// and the returned channel is closed.
// If f returns an error, the pipeline fails immediately: no accumulator is
// emitted and the returned channel is closed.
// If ctx is canceled, it stops early and returns.
func FoldErr[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	init B,
	f func(B, A) (B, error),
) <-chan B {
	return FoldErrCtx(
		ctx,
		p,
		in,
		init,
		func(_ context.Context, acc B, a A) (B, error) { return f(acc, a) },
	)
}

// FoldErrCtx reduces elements from `in“ into an accumulator using `f“.
//
// When the input channel closes without error, the final accumulator is sent
// and the returned channel is closed.
// If f returns an error, the pipeline fails immediately: no accumulator is
// emitted and the returned channel is closed.
// If ctx is canceled, it stops early and returns.
func FoldErrCtx[A, B any](
	ctx context.Context,
	p *Pipeline,
	in <-chan A,
	init B,
	f func(context.Context, B, A) (B, error),
) <-chan B {
	out := make(chan B, 1)

	p.goSafe(func() error {
		defer close(out)
		acc := init

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case a, ok := <-in:
				if !ok {
					select {
					case <-ctx.Done():
					case out <- acc:
					}
					return nil
				}

				var err error
				acc, err = f(ctx, acc, a)
				if err != nil {
					return err
				}
			}
		}
	})

	return out
}
