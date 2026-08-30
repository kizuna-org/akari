package app

import (
	"testing"

	"go.uber.org/fx"
)

// TestOptionsResolve checks that every dependency the program needs can be
// satisfied by something the program provides.
//
// This is the check that stands in for booting the container: a graph with a
// missing or mistyped provider fails here rather than crash-looping on a
// machine somewhere. It does not touch the database, because validation builds
// the graph without running the constructors.
func TestOptionsResolve(t *testing.T) {
	t.Parallel()

	err := fx.ValidateApp(Options())
	if err != nil {
		t.Fatalf("fx.ValidateApp() error = %v, want the wiring to resolve", err)
	}
}

// TestNewBuildsAnApp checks the program can be constructed at all.
func TestNewBuildsAnApp(t *testing.T) {
	t.Parallel()

	if New() == nil {
		t.Fatal("New() = nil, want an application")
	}
}
