package lifecycle

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/persistence"
)

const (
	crashDirectory = "AKARI_TEST_SESSION_CRASH_DIRECTORY"
	beforeEffect   = "before effect"
)

func TestKilledProcessDoesNotRedispatch(t *testing.T) {
	t.Parallel()

	for _, phase := range []string{beforeEffect, "after effect"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			//nolint:gosec // Re-execute this test binary with a constant test selector, without a shell.
			command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCrashChild$")

			command.Env = append(os.Environ(), crashDirectory+"="+directory, "AKARI_TEST_CRASH_PHASE="+phase)
			output, err := command.StdoutPipe()
			testError(t, err, nil)
			testError(t, command.Start(), nil)
			t.Cleanup(func() { _ = command.Process.Kill() })

			scanner := bufio.NewScanner(output)
			ready := false

			for scanner.Scan() {
				if scanner.Text() == "crash-boundary-ready" {
					ready = true

					break
				}
			}

			if !ready {
				t.Fatal("child did not reach persisted dispatch boundary")
			}

			testError(t, command.Process.Kill(), nil)

			err = command.Wait()
			if err == nil {
				t.Fatal("child exited without a process crash")
			}

			store, err := persistence.New(directory)
			testError(t, err, nil)
			gateway := crashGateway(t, directory, "parent")

			owner, err := Open(t.Context(), store, gateway, configuration())

			testError(t, err, nil)

			defer func() { testError(t, owner.Close(t.Context()), nil) }()

			if owner.Export().Outbox[actionID].Stage != continuity.Unknown {
				t.Fatal("interrupted dispatch was not recovered as unknown")
			}

			_, err = owner.Dispatch(t.Context(), actionID)
			testError(t, err, continuity.ErrAction)

			checkEffect(t, directory, phase)
		})
	}
}

func TestCrashChild(t *testing.T) {
	t.Parallel()

	directory := os.Getenv(crashDirectory)
	if directory == "" {
		t.Skip("subprocess helper")
	}

	for _, phase := range []string{os.Getenv("AKARI_TEST_CRASH_PHASE")} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()

			store, err := persistence.New(directory)
			testError(t, err, nil)
			owner, err := Open(t.Context(), store, crashGateway(t, directory, phase), configuration())
			testError(t, err, nil)
			_, err = owner.Adopt(t.Context(), proposal(), intents())
			testError(t, err, nil)
			_, err = owner.Dispatch(t.Context(), actionID)
			testError(t, err, nil)
		})
	}
}

func crashGateway(t *testing.T, directory, phase string) *action.Gateway {
	t.Helper()

	gateway, err := action.New(map[string]action.Prepare{toolName: func(arguments []byte) (action.Operation, error) {
		return action.Operation{Impact: action.Reversible, Destination: destination, ConflictKey: string(arguments),
			Call: nil, CallWithKey: func(ctx context.Context, _ string) ([]byte, error) {
				if phase != beforeEffect {
					//nolint:gosec // The parent creates this private temporary directory for the test subprocess.
					testError(t, os.WriteFile(filepath.Join(directory, "effect"), []byte("effect"), 0o600), nil)
				}

				_, err := os.Stdout.WriteString("crash-boundary-ready\n")
				testError(t, err, nil)
				<-ctx.Done()

				return nil, ctx.Err()
			}}, nil
	}}, 1)
	testError(t, err, nil)

	return gateway
}

func checkEffect(t *testing.T, directory, phase string) {
	t.Helper()

	//nolint:gosec // This path belongs to the isolated temporary directory created by the test.
	data, err := os.ReadFile(filepath.Join(directory, "effect"))
	if phase == beforeEffect {
		if !os.IsNotExist(err) {
			t.Fatal("operation unexpectedly executed")
		}
	} else if err != nil || strings.Count(string(data), "effect") != 1 {
		t.Fatal("external test effect was lost or duplicated")
	}
}
