package action

import (
	"context"
	"errors"
	"testing"
)

func TestIdentifiedAdapterRequiresDurableIdentity(t *testing.T) {
	t.Parallel()

	for _, identified := range []bool{false, true} {
		t.Run(map[bool]string{false: "unidentified", true: "empty identity"}[identified], func(t *testing.T) {
			t.Parallel()

			gateway, err := New(map[string]Prepare{"keyed": func([]byte) (Operation, error) {
				return Operation{Impact: Reversible, Destination: "local", ConflictKey: "target", Call: nil,
					CallWithKey: func(context.Context, string) ([]byte, error) {
						t.Fatal("adapter called without a durable key")

						return nil, nil
					}}, nil
			}}, 1)
			if err != nil {
				t.Fatal(err)
			}

			pending, err := gateway.Open(false, Scope{AllowedDestinations: []string{"local"}, ReadyRecipients: nil}).
				Plan(t.Context(), "keyed", nil)
			if err != nil {
				t.Fatal(err)
			}

			if identified {
				_, err = pending.ExecuteIdentified(t.Context(), "")
			} else {
				_, err = pending.Execute(t.Context())
			}

			if !errors.Is(err, ErrOperation) {
				t.Fatal("missing identity accepted")
			}
		})
	}
}
