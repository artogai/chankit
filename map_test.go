package chankit

import (
	"context"
	"math/rand"
	"slices"
	"testing"
	"time"
)

const (
	maxSleep = 3 * time.Millisecond
	mul      = 3
)

func FuzzMapTest(f *testing.F) {
	f.Add(0, 1, 0)
	f.Add(0, 1, 32)
	f.Add(1, 1, 0)
	f.Add(1, 4, 32)
	f.Add(1000, 1, 0)
	f.Add(1000, 4, 0)
	f.Add(1000, 4, 16)

	work := randWork(maxSleep, mul)

	f.Fuzz(func(t *testing.T, itemsN, parN, buf int) {
		if itemsN < 0 || itemsN > 2000 || parN <= 0 || parN > 64 || buf > 1_000 {
			t.Skip()
		}

		items := genInts(itemsN)
		opts := WithBuffer(buf)

		t.Run("Map Serially", func(t *testing.T) {
			t.Parallel()
			p, ctx := NewPipeline(t.Context())
			out := MapErrCtx(ctx, p, slice2chan(items), work, opts)
			got := chan2slice(out)
			check(t, p, items, got, true, mul)
		})

		t.Run("Map Concurrent Ordered", func(t *testing.T) {
			t.Parallel()
			p, ctx := NewPipeline(t.Context())
			out := MapErrCtx(ctx, p, slice2chan(items), work, opts, WithParallel(parN))
			got := chan2slice(out)
			check(t, p, items, got, true, mul)
		})

		t.Run("Map Concurrent Unordered", func(t *testing.T) {
			t.Parallel()
			p, ctx := NewPipeline(t.Context())
			out := MapErrCtx(
				ctx,
				p,
				slice2chan(items),
				work,
				opts,
				WithParallel(parN),
				WithUnordered(),
				WithReorderWindow(parN),
			)
			got := chan2slice(out)
			check(t, p, items, got, false, mul)
		})

		t.Run("Map Multi-Stage", func(t *testing.T) {
			t.Parallel()
			p, ctx := NewPipeline(t.Context())
			out1 := MapErrCtx(ctx, p, slice2chan(items), work, opts)
			out2 := MapErrCtx(ctx, p, out1, work, opts, WithParallel(parN))
			got := chan2slice(out2)
			check(t, p, items, got, true, mul*mul)
		})
	})
}

func check(t *testing.T, p *Pipeline, items, got []int, ordered bool, mul int) {
	t.Helper()

	if err := p.Wait(); err != nil {
		t.Fatalf("pipeline failed %v", err)
	}

	want := transformed(items, mul)

	if ordered {
		if !slices.Equal(got, want) {
			t.Fatalf("ordered mismatch")
		}
	} else {
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("unordered mismatch")
		}
	}
}

func randWork(max time.Duration, mul int) func(context.Context, int) (int, error) {
	return func(_ context.Context, x int) (int, error) {
		if max > 0 {
			time.Sleep(time.Duration(rand.Int63n(int64(max))))
		}
		return x * mul, nil
	}
}

func transformed(in []int, mul int) []int {
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = v * mul
	}
	return out
}
