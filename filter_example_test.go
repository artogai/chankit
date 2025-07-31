package chankit_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/artogai/chankit"
)

func ExampleFilter() {
	p, ctx := chankit.NewPipeline(context.Background())

	// keeps even numbers
	filter := func(v int) bool {
		return v%2 == 0
	}

	in := slice2chan([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	out := chankit.Filter(ctx, p, in, filter)

	for v := range out {
		fmt.Println(v)
	}

	if err := p.Wait(); err != nil {
		panic(err)
	}

	// Output:
	// 0
	// 2
	// 4
	// 6
	// 8
}

func ExampleFilterErr() {
	p, ctx := chankit.NewPipeline(context.Background())

	// keeps even numbers, fails on 3
	filter := func(v int) (bool, error) {
		if v == 3 {
			return false, errors.New("fail on 3")
		}
		return v%2 == 0, nil
	}

	in := slice2chan([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	out := chankit.FilterErr(ctx, p, in, filter)

	for v := range out {
		fmt.Println(v)
	}

	err := p.Wait()
	fmt.Println("err:", err != nil)

	// Output:
	// 0
	// 2
	// err: true
}

func ExampleFilterErrCtx() {
	p, ctx := chankit.NewPipeline(context.Background())

	// simulates io, keeps even numbers
	filter := func(ctx context.Context, v int) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return v%2 == 0, nil
		}
	}

	in := slice2chan([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	out := chankit.FilterErrCtx(ctx, p, in, filter)

	for v := range out {
		fmt.Println(v)
	}

	if err := p.Wait(); err != nil {
		panic(err)
	}

	// Output:
	// 0
	// 2
	// 4
	// 6
	// 8
}

func ExampleTake_infiniteProducer() {
	parentCtx := context.Background()

	prodCtx, prodCancel := context.WithCancel(parentCtx)
	in := tickerProducer(prodCtx)

	p, ctx := chankit.NewPipeline(parentCtx)

	// takes 5 first elements, then cancels the producer, drains leftovers in the background
	out := chankit.Take(ctx, p, in, 5, chankit.WithUpstreamCancel(prodCancel))

	for v := range out {
		fmt.Println(v)
	}

	if err := p.Wait(); err != nil {
		panic(err)
	}

	// Output:
	// 0
	// 1
	// 2
	// 3
	// 4
}

func ExampleTake_finiteProducer() {
	p, ctx := chankit.NewPipeline(context.Background())

	in := slice2chan([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	// takes first 5 elements, then drains the producer in the background
	out := chankit.Take(ctx, p, in, 5)

	for v := range out {
		fmt.Println(v)
	}

	if err := p.Wait(); err != nil {
		panic(err)
	}

	// Output:
	// 0
	// 1
	// 2
	// 3
	// 4
}

func ExampleDrop() {
	p, ctx := chankit.NewPipeline(context.Background())

	in := slice2chan([]int{0, 1, 2, 3, 4, 5})
	// drops 3 first elements
	out := chankit.Drop(ctx, p, in, 3)

	for v := range out {
		fmt.Println(v)
	}

	if err := p.Wait(); err != nil {
		panic(err)
	}

	// Output:
	// 3
	// 4
	// 5
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

func tickerProducer(ctx context.Context) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				out <- i
			}
		}
	}()

	return out
}
