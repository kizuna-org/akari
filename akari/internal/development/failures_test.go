package development

import (
	"errors"
	"testing"
)

func TestDatabaseRollbackAndCorruption(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		sql  string
	}{
		{name: "task write failure", sql: `CREATE TRIGGER reject_task BEFORE UPDATE ON tasks
BEGIN SELECT RAISE(FAIL, 'task write failed'); END`},
		{name: "metadata failure after task write", sql: `CREATE TRIGGER reject_metadata BEFORE UPDATE ON metadata
BEGIN SELECT RAISE(FAIL, 'metadata write failed'); END`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			store, err := Open(t.Context(), t.TempDir())

			must(t, err)

			defer func() { must(t, store.Close()) }()

			before, err := store.Snapshot(t.Context())
			must(t, err)
			_, err = store.database.ExecContext(t.Context(), test.sql)
			must(t, err)

			task := before.Tasks[0]
			task.Notes = "must not survive failed transaction"

			_, err = store.Change(t.Context(), Mutation{Operation: Update, ID: task.ID, Task: task,
				ExpectedRevision: 0, ExpectedVersion: task.Version})
			if err == nil {
				t.Fatal("failed storage was reported as saved")
			}

			after, err := store.Snapshot(t.Context())
			must(t, err)
			actual := taskByID(t, after, task.ID)

			if after.Revision != before.Revision || actual.Version != task.Version || actual.Notes == task.Notes {
				t.Fatal("partial task write escaped its failed transaction")
			}
		})
	}
}

func TestCorruptDatabaseIsNotRepairedByReseeding(t *testing.T) {
	t.Parallel()

	for _, sql := range []string{
		"PRAGMA user_version = -1",
		"DROP TABLE tasks",
		"DELETE FROM metadata",
		"UPDATE metadata SET updated_at = 'invalid'",
		"UPDATE metadata SET revision = 0",
		"UPDATE tasks SET refs = 'invalid' WHERE id = 'BASE-01'",
		"UPDATE tasks SET version = 0 WHERE id = 'BASE-01'",
	} {
		t.Run(sql, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			store, err := Open(t.Context(), directory)
			must(t, err)
			_, err = store.database.ExecContext(t.Context(), sql)
			must(t, err)
			must(t, store.Close())
			opened, err := Open(t.Context(), directory)

			if err == nil || opened != nil {
				t.Fatal("corrupted task database was treated as a new installation")
			}
		})
	}
}

func TestTaskVersionAndCapacityBoundaries(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		edit func(*Board, *Mutation)
		want error
	}{
		{name: "no version", edit: func(_ *Board, m *Mutation) { m.ExpectedVersion = 0 }, want: ErrConflict},
		{name: "wrong identity", edit: func(_ *Board, m *Mutation) { m.Task.ID = "another" }, want: ErrInvalid},
		{name: "blank edit", edit: func(_ *Board, m *Mutation) { m.Task.Title = "" }, want: ErrInvalid},
		{name: "too many references", edit: func(_ *Board, m *Mutation) {
			m.Task.References = make([]string, 9)
		}, want: ErrInvalid},
		{name: "full board", edit: func(board *Board, m *Mutation) {
			board.Tasks = make([]Task, TaskLimit)
			m.Operation = Create
			m.Task.ID = ""
		}, want: ErrInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			board, err := seedBoard()
			must(t, err)

			task := board.Tasks[0]
			mutation := Mutation{Operation: Update, Task: task, ID: task.ID,
				ExpectedRevision: 0, ExpectedVersion: task.Version}
			test.edit(&board, &mutation)

			err = applyMutation(&board, mutation)
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}
