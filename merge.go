package chankit

import (
	"context"
)

func Merge[A any](
	ctx context.Context,
	p *Pipeline,
	leftIn <-chan A,
	rightIn <-chan A,
	opts ...Option,
) <-chan A {
	if leftIn == nil || rightIn == nil {
		panic("Merge: input channels must not be nil")
	}

	cfg := makeConfig(opts)
	out := make(chan A, cfg.bufCap)

	var leftDone, rightDone bool

	shouldStop := func() (bool, error) {
		switch cfg.haltStrategy {
		case HaltLeft:
			return leftDone, nil
		case HaltRight:
			return rightDone, nil
		case HaltEither:
			return leftDone || rightDone, nil
		case HaltBoth:
			return leftDone && rightDone, nil
		default:
			return true, ErrUnknownHaltStrategy
		}
	}

	p.goSafe(func() error {
		defer close(out)

		for {
			stop, err := shouldStop()
			if err != nil {
				return err
			}
			if stop {
				return nil
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case la, ok := <-leftIn:
				if !ok {
					leftDone = true
					leftIn = nil
					continue
				}
				select {
				case <-ctx.Done():
				case out <- la:
				}
			case ra, ok := <-rightIn:
				if !ok {
					rightDone = true
					rightIn = nil
					continue
				}
				select {
				case <-ctx.Done():
				case out <- ra:
				}
			}
		}
	})

	return out
}
