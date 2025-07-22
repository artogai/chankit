package chankit

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"testing"
	"time"
)

const (
	rounds   = 100
	itemsMax = 1000
	parNMax  = 12
	bufMax   = 64
	mul      = 10
	maxSleep = 0 * time.Millisecond
)

func TestMapsGen(t *testing.T) {
	cases := []struct {
		name   string
		items  []int
		parN   int
		bufCap int
	}{
		{"empty-unbuffered", nil, 1, 0},
		{"empty-buffered", nil, 1, 32},
		{"single-item-unbuffered", []int{1}, 1, 0},
		{"single-item-buffered", []int{1}, 4, 32},
	}

	for _, c := range cases {
		t.Run("Corner/"+c.name, func(t *testing.T) {
			t.Parallel()
			checkAllVariants(t, c.items, c.parN, c.bufCap)
		})
	}

	runRandom := func(tag string, bufSelector func() int) {
		for i := range rounds {
			items := randItems(itemsMax)
			parN := rand.Intn(parNMax) + 1
			bufCap := bufSelector()

			t.Run(fmt.Sprintf("%s/%d", tag, i), func(t *testing.T) {
				t.Parallel()
				checkAllVariants(t, items, parN, bufCap)
			})
		}
	}

	runRandom("rng-buffered", func() int { return 0 })
	runRandom("rng-unbuffered", func() int { return rand.Intn(bufMax) + 1 })
}

func checkAllVariants(t *testing.T, items []int, parN, buf int) {
	work := randWork(maxSleep, mul)

	check := func(name string, got []int, ordered bool, iterMul int) {
		t.Helper()
		want := transformed(items, iterMul)
		if ordered {
			if !slices.Equal(got, want) {
				t.Fatalf("%s: ordered mismatch", name)
			}
		} else {
			slices.Sort(got)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("%s: unordered mismatch", name)
			}
		}
	}

	// Map
	func() {
		p, ctx := NewPipeline(context.Background())
		out := Map(ctx, p, slice2chan(items), work, WithBuffer(buf))
		got := chan2slice(out)
		if err := p.Wait(); err != nil {
			t.Fatalf("Map Wait(): %v", err)
		}
		check("Map", got, true, mul)
	}()

	// MapConcOrdered
	func() {
		p, ctx := NewPipeline(context.Background())
		out := MapConcOrdered(ctx, p, slice2chan(items), parN, work, WithBuffer(buf))
		got := chan2slice(out)
		if err := p.Wait(); err != nil {
			t.Fatalf("MapConcOrdered Wait(): %v", err)
		}
		check("MapConcOrdered", got, true, mul)
	}()

	// MapConcUnordered
	func() {
		p, ctx := NewPipeline(context.Background())
		out := MapConcUnordered(ctx, p, slice2chan(items), parN, work, WithBuffer(buf))
		got := chan2slice(out)
		if err := p.Wait(); err != nil {
			t.Fatalf("MapConcUnordered Wait(): %v", err)
		}
		check("MapConcUnordered", got, false, mul)
	}()

	// MultiStage
	func() {
		p, ctx := NewPipeline(context.Background())
		stage1 := Map(ctx, p, slice2chan(items), work, WithBuffer(buf))
		stage2 := MapConcOrdered(ctx, p, stage1, parN, work, WithBuffer(buf))
		got := chan2slice(stage2)
		if err := p.Wait(); err != nil {
			t.Fatalf("MultiStage: %v", err)
		}
		check("MultiStage", got, true, mul*mul)
	}()
}

func randWork(max time.Duration, mul int) func(context.Context, int) (int, error) {
	return func(_ context.Context, x int) (int, error) {
		if max > 0 {
			time.Sleep(time.Duration(rand.Int63n(int64(max))))
		}
		return x * mul, nil
	}
}

func randItems(max int) []int {
	n := rand.Intn(max) + 1
	items := make([]int, n)
	for i := range items {
		items[i] = rand.Intn(10_000)
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
