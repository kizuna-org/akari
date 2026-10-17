package development

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestTaskDatabaseLifecycle(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"create update reopen delete"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			store, err := Open(t.Context(), directory)
			must(t, err)
			t.Cleanup(func() { must(t, store.Close()) })
			board, err := store.Snapshot(t.Context())
			must(t, err)

			if len(board.Tasks) < 20 || board.Revision != 1 {
				t.Fatal("initial backlog was not seeded")
			}

			info, err := os.Stat(filepath.Join(directory, "development.db"))
			must(t, err)

			if info.Mode().Perm() != databasePermission {
				t.Fatal("task database is not private")
			}

			task := exampleTask()
			created, err := store.Change(t.Context(), Mutation{Operation: Create, Task: task,
				ID: "", ExpectedRevision: board.Revision, ExpectedVersion: 0})
			must(t, err)

			task = created.Tasks[len(created.Tasks)-1]
			task.Status = InProgress
			task.Notes = "saved from API"
			updated, err := store.Change(t.Context(), Mutation{Operation: Update, Task: task,
				ID: task.ID, ExpectedRevision: 0, ExpectedVersion: task.Version})
			must(t, err)

			if updated.Revision != created.Revision+1 {
				t.Fatal("task update did not advance board revision")
			}

			must(t, store.Close())
			reopened, err := Open(t.Context(), directory)

			must(t, err)

			defer func() { must(t, reopened.Close()) }()

			restored, err := reopened.Snapshot(t.Context())
			must(t, err)

			found := taskByID(t, restored, task.ID)

			if found.Notes != task.Notes || found.Status != task.Status || found.Version != task.Version+1 {
				t.Fatal("database did not preserve the edited task")
			}

			deleted, err := reopened.Change(t.Context(), Mutation{Operation: Delete, ID: task.ID,
				Task: task, ExpectedRevision: 0, ExpectedVersion: found.Version})
			must(t, err)

			if len(deleted.Tasks) != len(board.Tasks) {
				t.Fatal("deleted task remains in database")
			}
		})
	}
}

func TestConcurrentTaskVersions(t *testing.T) {
	t.Parallel()

	for _, sameTask := range []bool{true, false} {
		names := map[bool]string{true: "same task conflicts", false: "independent tasks both succeed"}
		t.Run(names[sameTask], func(t *testing.T) {
			t.Parallel()

			store, err := Open(t.Context(), t.TempDir())

			must(t, err)

			defer func() { must(t, store.Close()) }()

			board, err := store.Snapshot(t.Context())
			must(t, err)

			first := board.Tasks[0]
			second := board.Tasks[1]

			if sameTask {
				second = first
			}

			start := make(chan struct{})
			results := make(chan error, 2)
			group := new(sync.WaitGroup)

			for _, task := range []Task{first, second} {
				group.Go(func() {
					<-start

					task.Status = InProgress

					_, changeErr := store.Change(t.Context(), Mutation{Operation: Update, ID: task.ID, Task: task,
						ExpectedRevision: 0, ExpectedVersion: task.Version})
					results <- changeErr
				})
			}

			close(start)
			group.Wait()
			close(results)

			successes := 0

			for changeErr := range results {
				if changeErr == nil {
					successes++
				} else if !errors.Is(changeErr, ErrConflict) {
					t.Fatal(changeErr)
				}
			}

			want := 2
			if sameTask {
				want = 1
			}

			if successes != want {
				t.Fatalf("got %d successful writers, want %d", successes, want)
			}
		})
	}
}

func TestRejectedTaskChangesAreAtomic(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		change func(*Mutation)
		want   error
	}{
		{name: "stale board", change: func(m *Mutation) { m.ExpectedRevision = 999 }, want: ErrConflict},
		{name: "invalid title", change: func(m *Mutation) { m.Task.Title = " " }, want: ErrInvalid},
		{name: "invalid status", change: func(m *Mutation) { m.Task.Status = "started" }, want: ErrInvalid},
		{name: "invalid priority", change: func(m *Mutation) { m.Task.Priority = "urgent" }, want: ErrInvalid},
		{name: "invalid category", change: func(m *Mutation) { m.Task.Category = "other" }, want: ErrInvalid},
		{name: "external reference",
			change: func(m *Mutation) { m.Task.References = []string{"https://example.com"} }, want: ErrInvalid},
		{name: "path traversal",
			change: func(m *Mutation) { m.Task.References = []string{"docs/../private.md"} }, want: ErrInvalid},
		{name: "caller assigned ID", change: func(m *Mutation) { m.Task.ID = "assigned" }, want: ErrInvalid},
		{name: "missing task", change: func(m *Mutation) { m.Operation = Delete; m.ID = "absent" }, want: ErrAbsent},
		{name: "unknown operation", change: func(m *Mutation) { m.Operation = "PATCH"; m.ID = "BASE-01" }, want: ErrInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			store, err := Open(t.Context(), t.TempDir())

			must(t, err)

			defer func() { must(t, store.Close()) }()

			mutation := Mutation{Operation: Create, ID: "", Task: exampleTask(), ExpectedRevision: 1, ExpectedVersion: 0}
			test.change(&mutation)
			_, err = store.Change(t.Context(), mutation)

			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}

			board, err := store.Snapshot(t.Context())
			must(t, err)

			if board.Revision != 1 {
				t.Fatal("rejected edit changed the board")
			}
		})
	}
}

func TestDatabaseRejectsBadStorage(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"corrupt", "symlink", "missing directory", "canceled", "future schema"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			switch name {
			case "corrupt":
				must(t, os.WriteFile(filepath.Join(directory, "development.db"), []byte("not sqlite"), databasePermission))
			case "symlink":
				must(t, os.Symlink("elsewhere", filepath.Join(directory, "development.db")))
			case "missing directory":
				directory = filepath.Join(directory, "missing")
			case "canceled":
				cancel()
			case "future schema":
				store, err := Open(t.Context(), directory)
				must(t, err)
				_, err = store.database.ExecContext(t.Context(), "PRAGMA user_version = 99")
				must(t, err)
				must(t, store.Close())
			}

			store, err := Open(ctx, directory)
			if err == nil || store != nil {
				t.Fatal("invalid task storage was silently replaced")
			}
		})
	}
}

func TestStrictTaskJSON(t *testing.T) {
	t.Parallel()

	for _, input := range []string{`{"unknown":true}`, `{`, `{} {}`, "null"} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			var task Task

			if Decode(strings.NewReader(input), &task) == nil {
				t.Fatal("malformed or trailing JSON was accepted")
			}
		})
	}
}

func exampleTask() Task {
	return Task{ID: "", Version: 0, Title: "Development task", Description: "An API managed task",
		Status: "todo", Priority: "high", Category: "runtime", Notes: "", References: []string{"design/05-delivery.md"}}
}

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func taskByID(t *testing.T, board Board, identity string) Task {
	t.Helper()

	for _, task := range board.Tasks {
		if task.ID == identity {
			return task
		}
	}

	t.Fatal("task missing")

	var missing Task

	return missing
}
