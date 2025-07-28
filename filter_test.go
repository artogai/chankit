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
		name          string
		withCancelOpt bool
		bufCap        int
	}{
		{"drain-mode-unbuff", false, 0},
		{"cancel-upstream-unbuff", true, 0},
		{"drain-mode-buff", false, 32},
		{"cancel-upstream-buff", true, 32},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			n := 100
			sent := int32(0)

			prodCtx := t.Context()
			opts := []Option{WithBuffer(tc.bufCap)}
			if tc.withCancelOpt {
				ctx, prodCancel := context.WithCancel(prodCtx)
				prodCtx = ctx
				opts = append(opts, WithUpstreamCancel(prodCancel))
			}

			in := countingProducer(prodCtx, !tc.withCancelOpt, n, &sent)

			p, ctx := NewPipeline(t.Context())
			out := Take(ctx, p, in, n, opts...)

			// drain output
			got := 0
			for range out {
				got++
			}
			if got != n {
				t.Fatalf("expected %d values, got %d", n, got)
			}

			// wait for pipeline to finish
			if err := p.Wait(); err != nil {
				t.Fatalf("pipeline error: %v", err)
			}

			if tc.withCancelOpt {
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

func countingProducer(ctx context.Context, finite bool, n int, counter *int32) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := 0; ; i++ {
			if finite && i > n*2 {
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
