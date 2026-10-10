package meaning

import (
	"errors"
	"math"
	"testing"
)

const testSpace = "test"

func TestSemanticDistance(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		vector []float64
		space  string
		want   float64
		err    error
	}{
		{name: "same", vector: []float64{2, 0}, space: testSpace, want: 1, err: nil},
		{name: "opposite", vector: []float64{-2, 0}, space: testSpace, want: -1, err: nil},
		{name: "orthogonal", vector: []float64{0, 2}, space: testSpace, want: 0, err: nil},
		{name: "wrong space", vector: []float64{2, 0}, space: "other", want: 0, err: ErrPoint},
		{name: "wrong dimensions", vector: []float64{1}, space: testSpace, want: 0, err: ErrPoint},
		{name: "zero", vector: []float64{0, 0}, space: testSpace, want: 0, err: ErrPoint},
		{name: "empty", vector: nil, space: testSpace, want: 0, err: ErrPoint},
		{name: "NaN", vector: []float64{math.NaN(), 0}, space: testSpace, want: 0, err: ErrPoint},
		{name: "overflowing norm",
			vector: []float64{math.MaxFloat64, math.MaxFloat64}, space: testSpace, want: 0, err: ErrPoint},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			left := Point{Text: "same label", Space: testSpace, Vector: []float64{1, 0}}
			right := Point{Text: left.Text, Space: test.space, Vector: test.vector}

			got, err := Similarity(left, right)
			if got != test.want || !errors.Is(err, test.err) {
				t.Fatalf("distance = %v, %v", got, err)
			}

			clone := left.Clone()
			clone.Vector[0] = 0

			if !left.Valid() || clone.Valid() || Unit(-1) || Unit(math.Inf(1)) {
				t.Fatal("invalid or aliased coordinates")
			}
		})
	}
}
