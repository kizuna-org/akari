package continuity

import (
	"context"
	"errors"
	"testing"
)

func TestOutcomeGateStopsOnDeadlineOrUnavailable(t *testing.T) {
	t.Parallel()

	for _, unavailable := range []bool{false, true} {
		t.Run(map[bool]string{false: "canceled wait", true: "failed storage"}[unavailable], func(t *testing.T) {
			t.Parallel()

			owner := new(Owner)
			owner.gate = make(chan struct{}, 1)
			owner.unavailable = unavailable

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			want := error(ErrUnavailable)

			if !unavailable {
				owner.gate <- struct{}{}

				cancel()

				want = context.Canceled
			}

			err := owner.enterWait(ctx)
			if !errors.Is(err, want) {
				t.Fatalf("outcome gate returned %v, want %v", err, want)
			}
		})
	}
}
