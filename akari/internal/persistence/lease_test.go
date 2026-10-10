package persistence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProcessLeaseFailuresAndRelease(t *testing.T) {
	t.Parallel()

	const canceledCaller = "canceled caller"

	for _, failure := range []string{canceledCaller, "invalid path", "none"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()

			file, err := New(t.TempDir())
			assertError(t, err, nil)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if failure == canceledCaller {
				cancel()
			}

			if failure == "invalid path" {
				err = os.Mkdir(filepath.Join(file.directory, "process.lock"), privateDirectory)
				assertError(t, err, nil)
			}

			release, err := file.Acquire(ctx)
			if failure != "none" {
				if err == nil || release != nil {
					t.Fatal("invalid lease acquisition succeeded")
				}

				return
			}

			assertError(t, err, nil)

			release()
			release()

			release, err = file.Acquire(ctx)
			assertError(t, err, nil)

			release()
		})
	}
}
