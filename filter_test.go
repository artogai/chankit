package chankit

import (
	"context"
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
		want := filterInts(items, pred)

		p, ctx := NewPipeline(t.Context())
		out := Filter(ctx, p, slice2chan(items), pred, WithBuffer(buf))
		got := chan2slice(out)

		assertNoPipeError(t, p)
		assertSlicesEqual(t, want, got)
	})
}

func TestTake(t *testing.T) {
	tests := []struct {
		name     string
		isFinite bool
		bufCap   int
	}{
		{"finite-unbuff", true, 0},
		{"finite-buff", true, 32},
		{"infinite-unbuff", false, 0},
		{"infinite-buff", false, 32},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			n := 100
			limit := 0
			if tc.isFinite {
				limit = 500
			}

			prodCtx, prodCancel := context.WithCancel(t.Context())
			defer prodCancel()
			in := CreateProducer(prodCtx, WithLimit(limit))

			p, ctx := NewPipeline(t.Context())
			out := Take(ctx, p, in, n, WithBuffer(tc.bufCap))
			got := chan2slice(out)

			if len(got) != n {
				t.Fatalf("want %d values, got %d", n, len(got))
			}

			if !tc.isFinite {
				prodCancel()
			}
			assertNoPipeError(t, p)
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

			assertNoPipeError(t, p)
			assertSlicesEqual(t, tc.want, got)
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
