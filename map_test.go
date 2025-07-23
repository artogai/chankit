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

		items := items(itemsN)
		opts := WithBuffer(buf)

		t.Run("Map", func(t *testing.T) {
			t.Parallel()
			p, ctx := NewPipeline(t.Context())
			out := Map(ctx, p, slice2chan(items), work, opts)
			got := chan2slice(out)
			check(t, p, items, got, true, mul)
		})

		t.Run("MapConcUnordered", func(t *testing.T) {
			t.Parallel()
			p, ctx := NewPipeline(t.Context())
			out := MapConcUnordered(ctx, p, slice2chan(items), parN, work, opts)
			got := chan2slice(out)
			check(t, p, items, got, false, mul)
		})

		t.Run("MapConcOrdered", func(t *testing.T) {
			t.Parallel()
			p, ctx := NewPipeline(t.Context())
			out := MapConcOrdered(ctx, p, slice2chan(items), parN, work, opts)
			got := chan2slice(out)
			check(t, p, items, got, true, mul)
		})

		t.Run("MapMultiStage", func(t *testing.T) {
			t.Parallel()
			p, ctx := NewPipeline(t.Context())
			out1 := Map(ctx, p, slice2chan(items), work, opts)
			out2 := MapConcOrdered(ctx, p, out1, parN, work, opts)
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

func items(n int) []int {
	items := make([]int, n)
	for i := range items {
		items[i] = i
	}
	return items
}

func transformed(in []int, mul int) []int {
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = v * mul
	}
	return out
}

func slice2chan[T any](in []T) <-chan T {
	out := make(chan T, len(in))
	go func() {
		for _, v := range in {
			out <- v
		}
		close(out)
	}()
	return out
}

func chan2slice[T any](ch <-chan T) []T {
	var out []T
	for v := range ch {
		out = append(out, v)
	}
	return out
}
