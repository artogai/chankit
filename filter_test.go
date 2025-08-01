package chankit

import (
	"context"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"
)

func FuzzFilter(f *testing.F) {
	f.Add(100, 1, 0)
	f.Add(1000, 3, 16)

	f.Fuzz(func(t *testing.T, itemsN, keepMod, buf int) {
		if itemsN < 0 || itemsN > 2_000 || keepMod <= 0 || keepMod > 50 || buf > 1_000 {
			t.Skip()
		}

		items := genInts(itemsN)
		pred := func(x int) bool { return x%keepMod == 0 }
		opts := WithBuffer(buf)

		p, ctx := NewPipeline(t.Context())
		outCh := Filter(ctx, p, slice2chan(items), pred, opts)
		got := chan2slice(outCh)

		if err := p.Wait(); err != nil {
			t.Fatalf("pipeline failed: %v", err)
		}
		want := filterInts(items, pred)

		if !slices.Equal(got, want) {
			t.Fatalf("ordered mismatch")
		}
	})
}

func TestTake(t *testing.T) {
	tests := []struct {
		name     string
		isFinite bool
		bufCap   int
	}{
		{"finite-unbuff", true, 0},
		{"infinite-unbuff", false, 0},
		{"finite-buff", true, 32},
		{"infinite-buff", false, 32},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			n := 100
			sent := int32(0)

			prodCtx, prodCancel := context.WithCancel(t.Context())
			defer prodCancel()
			in := countingProducer(prodCtx, tc.isFinite, n, &sent)

			p, ctx := NewPipeline(t.Context())
			out := Take(ctx, p, in, n, WithBuffer(tc.bufCap))

			// drain output
			got := 0
			for range out {
				got++
			}
			if got != n {
				t.Fatalf("expected %d values, got %d", n, got)
			}

			if !tc.isFinite {
				prodCancel()
			}

			if err := p.Wait(); err != nil {
				t.Fatalf("pipeline error: %v", err)
			}

			if !tc.isFinite {
				if atomic.LoadInt32(&sent) > int32(n) {
					t.Fatalf("producer kept running after cancel, sent=%d", sent)
				}
			}
		})
	}
}

func TestDrop(t *testing.T) {
	tests := []struct {
		name string
		n    int
		in   []int
		want []int
	}{
		{"passthrough", 0, []int{1, 2, 3}, []int{1, 2, 3}},
		{"normal drop", 3, []int{0, 1, 2, 3, 4}, []int{3, 4}},
		{"drop more than len", 5, []int{7, 8}, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, ctx := NewPipeline(t.Context())

			in := slice2chan(tc.in)
			out := Drop(ctx, p, in, tc.n)
			got := chan2slice(out)

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}

			if err := p.Wait(); err != nil {
				t.Fatalf("pipeline error: %v", err)
			}
		})
	}
}

func filterInts(in []int, pred func(x int) bool) []int {
	var out []int
	for v := range in {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

func countingProducer(ctx context.Context, isFinite bool, n int, counter *int32) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := 0; ; i++ {
			if isFinite && i > n*2 {
				return
			}
			select {
			case <-ctx.Done():
				return
			case out <- i:
				atomic.AddInt32(counter, 1)
			}
		}
	}()

	return out
}
