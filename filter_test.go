package chankit

import (
	"slices"
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

func filterInts(in []int, pred func(x int) bool) []int {
	var out []int
	for v := range in {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}
