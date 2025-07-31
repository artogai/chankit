package chankit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFold(t *testing.T) {
	tests := []struct {
		name     string
		expected int
		input    []int
	}{
		{"empty input", 0, nil},
		{"sum", 15, []int{0, 1, 2, 3, 4, 5}},
	}

	agg := func(acc, i int) int { return acc + i }

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, ctx := NewPipeline(t.Context())
			out := Fold(ctx, p, slice2chan(tc.input), 0, agg)

			v := <-out
			if v != tc.expected {
				t.Fatalf("expected %d, got %d", tc.expected, v)
			}

			if err := p.Wait(); err != nil {
				t.Fatalf("pipeline error: %v", err)
			}
		})
	}
}

func TestFoldErr(t *testing.T) {
	t.Run("error propagation", func(t *testing.T) {
		t.Parallel()

		f := func(acc, v int) (int, error) {
			if v < 0 {
				return 0, errors.New("negative input")
			}
			return acc + v, nil
		}

		p, ctx := NewPipeline(t.Context())
		in := slice2chan([]int{1, 2, -1, 4})
		_ = FoldErr(ctx, p, in, 0, f)

		if err := p.Wait(); err == nil {
			t.Fatal("error expected")
		}
	})

}

func TestFoldErrCtx(t *testing.T) {
	t.Run("cancellable fold", func(t *testing.T) {
		t.Parallel()

		p, ctx := NewPipeline(t.Context())

		stageCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()

		in := make(chan int)
		out := FoldErrCtx(stageCtx, p, in, 0, func(ctx context.Context, acc, v int) (int, error) {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
				return acc + v, nil
			}
		})

		go func() {
			defer close(in)
			for i := range 1000000 {
				in <- i
			}
		}()

		select {
		case <-out:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("timed out")
		}
	})
}
