package chankit_test

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/artogai/chankit"
)

func ExampleFold() {
	p, ctx := chankit.NewPipeline(context.Background())

	// formats int and accumulates string
	accumulateStr := func(strAcc string, i int) string {
		return strAcc + fmt.Sprintf("%d", i)
	}

	in := slice2chan([]int{0, 1, 2, 3, 4, 5})
	out := chankit.Fold(ctx, p, in, "", accumulateStr)

	v := <-out
	fmt.Println(v)

	if err := p.Wait(); err != nil {
		panic(err)
	}

	// Output:
	// 012345
}

func ExampleFoldErr() {
	p, ctx := chankit.NewPipeline(context.Background())

	// parses string and accumulates sum of ints
	accumulateInt := func(acc int64, s string) (int64, error) {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, err
		}
		return acc + i, nil
	}

	in := slice2chan([]string{"0", "1", "2", "3", "4", "5"})
	out := chankit.FoldErr(ctx, p, in, 0, accumulateInt)

	v := <-out
	fmt.Println(v)

	if err := p.Wait(); err != nil {
		panic(err)
	}

	// Output:
	// 15
}

func ExampleFoldErrCtx() {
	p, ctx := chankit.NewPipeline(context.Background())

	// simulates io, parses string and accumulates sum
	accumulateIO := func(ctx context.Context, acc int64, s string) (int64, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(10 * time.Millisecond):
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return 0, err
			}
			return acc + i, nil
		}
	}

	in := slice2chan([]string{"0", "1", "2", "3", "4", "5"})
	out := chankit.FoldErrCtx(ctx, p, in, 0, accumulateIO)

	v := <-out
	fmt.Println(v)

	if err := p.Wait(); err != nil {
		panic(err)
	}

	// Output:
	// 15
}
