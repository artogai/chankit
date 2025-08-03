package chankit

import (
	"cmp"
	"fmt"
	"testing"
)

func genInts(n int) []int {
	items := make([]int, n)
	for i := range items {
		items[i] = i
	}
	return items
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

type cmpAndOrd interface {
	comparable
	cmp.Ordered
}

func assertNoPipeError(t *testing.T, p *Pipeline) {
	t.Helper()
	if err := p.Wait(); err != nil {
		t.Fatalf("pipeline error: %v", err)
	}
}

func assertSameElementsAs[T cmpAndOrd](t *testing.T, want, got []T, previewCnt ...int) {
	t.Helper()
	n := 20
	if len(previewCnt) > 0 {
		n = previewCnt[0]
	}

	diff := make(map[T]int)
	for _, v := range got {
		diff[v]++
	}
	for _, v := range want {
		diff[v]--
	}

	var extras, missings []string
	for v, d := range diff {
		switch {
		case d > 0:
			extras = append(extras, fmt.Sprintf("%vx%d", v, d))
		case d < 0:
			missings = append(missings, fmt.Sprintf("%vx%d", v, -d))
		}
	}

	if len(extras) == 0 && len(missings) == 0 {
		return
	}

	if n > 0 {
		if len(extras) > n {
			extras = append(extras[:n], fmt.Sprintf("... +%d more", len(extras)-n))
		}
		if len(missings) > n {
			missings = append(missings[:n], fmt.Sprintf("... +%d more", len(missings)-n))
		}
	}

	t.Fatalf(
		"slices diff:\n extra in got: %v\n missing from got: %v\n got: %s\n want: %s",
		extras, missings, previewSlice(got, n), previewSlice(want, n),
	)
}

func assertSlicesEqual[T cmpAndOrd](t *testing.T, want, got []T, previewCnt ...int) {
	t.Helper()
	n := 10
	if len(previewCnt) > 0 {
		n = previewCnt[0]
	}

	if len(got) != len(want) {
		t.Fatalf(
			"length mismatch: got=%d want=%d\n got: %s\n want: %s",
			len(got), len(want), previewSlice(got, n), previewSlice(want, n),
		)
	}

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf(
				"slices differ at index %d: got=%v want=%v\n got: %s\n want: %s",
				i, got[i], want[i], previewSlice(got, n), previewSlice(want, n),
			)
		}
	}
}

func previewSlice[T any](s []T, n int) string {
	if n <= 0 || len(s) <= n {
		return fmt.Sprintf("%v (len=%d)", s, len(s))
	}
	return fmt.Sprintf("%v ...(showing %d/%d)", s[:n], n, len(s))
}
