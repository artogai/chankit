package chankit

import (
	"context"
	"log"

	"golang.org/x/sync/errgroup"
)

func Map[A, B any](ctx context.Context, eg *errgroup.Group, in <-chan A, fn func(A) B) <-chan B {
	return MapErrCtx(ctx, eg, in, func(ctx context.Context, a A) (B, error) {
		return fn(a), nil
	})
}

func MapErr[A, B any](ctx context.Context, eg *errgroup.Group, in <-chan A, fn func(A) (B, error)) <-chan B {
	return MapErrCtx(ctx, eg, in, func(ctx context.Context, a A) (B, error) {
		return fn(a)
	})
}

func MapErrCtx[A, B any](
	ctx context.Context,
	eg *errgroup.Group,
	in <-chan A,
	fn func(context.Context, A) (B, error),
	bufSize ...int,
) <-chan B {
	size := 0
	if len(bufSize) > 0 {
		size = bufSize[0]
	}
	out := make(chan B, size)

	eg.Go(func() error {
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
				case out <- b:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	})

	return out
}

func test(ctx context.Context, in <-chan int) (<-chan int, *errgroup.Group) {
	eg, ctx := errgroup.WithContext(ctx)
	out1 := Map(ctx, eg, in, func(i int) int { return i + 1 })
	out2 := MapErr(ctx, eg, out1, func(i int) (int, error) { return i, nil })
	out3 := MapErrCtx(ctx, eg, out2, func(ctx context.Context, i int) (int, error) { return i, nil })
	return out3, eg
}

func main() {
	ctx := context.Background()
	var in <-chan int
	out, eg := test(ctx, in)

	for o := range out {
		println(o)
	}

	if err := eg.Wait(); err != nil {
		log.Fatal("error: ", err)
	}
}
