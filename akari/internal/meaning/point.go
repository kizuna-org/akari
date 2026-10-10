// Package meaning carries semantic coordinates without selecting an embedding provider.
package meaning

import (
	"context"
	"math"
	"slices"
)

type fault string

func (err fault) Error() string { return string(err) }

const ErrPoint fault = "semantic point must have finite nonzero coordinates in a named space"

// Point names its encoder revision; equal dimensions alone do not imply compatibility.
type Point struct {
	Text   string    `json:"text"`
	Space  string    `json:"space"`
	Vector []float64 `json:"vector"`
}

// Encoder is provided by a selected adapter; core code does not invent embeddings.
type Encoder interface {
	Encode(ctx context.Context, text string) (Point, error)
}

func (point Point) Clone() Point {
	return Point{Text: point.Text, Space: point.Space, Vector: slices.Clone(point.Vector)}
}

func (point Point) Valid() bool {
	if point.Text == "" || point.Space == "" || len(point.Vector) == 0 {
		return false
	}

	norm := 0.0

	for _, coordinate := range point.Vector {
		if !Finite(coordinate) {
			return false
		}

		norm = math.Hypot(norm, coordinate)
	}

	return norm > 0 && Finite(norm)
}

func Similarity(left, right Point) (float64, error) {
	if !left.Valid() || !right.Valid() || left.Space != right.Space || len(left.Vector) != len(right.Vector) {
		return 0, ErrPoint
	}

	leftNorm, rightNorm := 0.0, 0.0

	for index := range left.Vector {
		leftNorm = math.Hypot(leftNorm, left.Vector[index])
		rightNorm = math.Hypot(rightNorm, right.Vector[index])
	}

	dot := 0.0

	for index := range left.Vector {
		dot += (left.Vector[index] / leftNorm) * (right.Vector[index] / rightNorm)
	}

	return math.Max(-1, math.Min(1, dot)), nil
}

func Finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func Unit(value float64) bool { return Finite(value) && value >= 0 && value <= 1 }
