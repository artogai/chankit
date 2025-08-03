package chankit

import (
	"context"
	"errors"
	"testing"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name  string
		halt  HaltStrategy
		left  <-chan int
		right <-chan int
		want  []int
	}{
		{
			"halt-both",
			HaltBoth,
			slice2chan([]int{0, 1, 2, 3}),
			slice2chan([]int{10, 11, 12}),
			[]int{0, 1, 2, 3, 10, 11, 12},
		},
		{"halt-left", HaltLeft, slice2chan([]int{0, 1, 2}), make(chan int), []int{0, 1, 2}},
		{"halt-right", HaltRight, make(chan int), slice2chan([]int{0, 1, 2}), []int{0, 1, 2}},
		{
			"halt-either-left",
			HaltEither,
			slice2chan([]int{0, 1, 2}),
			make(chan int),
			[]int{0, 1, 2},
		},
		{
			"halt-either-right",
			HaltEither,
			make(chan int),
			slice2chan([]int{0, 1, 2}),
			[]int{0, 1, 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, ctx := NewPipeline(t.Context())

			out := Merge(ctx, p, tc.left, tc.right, WithHaltStrategy(tc.halt))
			got := chan2slice(out)

			assertSameElementsAs(t, tc.want, got)
			if err := p.Wait(); err != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("pipeline error: %v", err)
			}
		})
	}
}

func FuzzMerge_EquivalentToConcat(f *testing.F) {
	f.Add(100, 200)
	f.Fuzz(func(t *testing.T, n, m int) {
		if n <= 0 || n > 1000 || m <= 0 || m > 1000 {
			return
		}

		leftVals := genInts(n)
		rightVals := genInts(m)
		want := append(leftVals, rightVals...)

		left := slice2chan(leftVals)
		right := slice2chan(rightVals)

		p, pctx := NewPipeline(t.Context())
		out := Merge(pctx, p, left, right, WithHaltStrategy(HaltBoth))
		got := chan2slice(out)

		assertSameElementsAs(t, want, got)
	})
}
