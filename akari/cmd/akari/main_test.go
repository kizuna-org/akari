package main

import "testing"

func TestMain(t *testing.T) { //nolint:paralleltest
	called := false
	originalRun := run

	t.Cleanup(func() {
		run = originalRun
	})

	run = func() {
		called = true
	}

	main()

	if !called {
		t.Fatal("main() did not run application")
	}
}
