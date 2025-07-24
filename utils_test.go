package chankit

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
