package development

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Registers the pure-Go SQLite driver; CGO remains disabled.
)

//go:embed taskSeed.json
var taskSeed []byte

type Store struct{ database *sql.DB }

const databasePermission = 0o600

func Open(ctx context.Context, directory string) (*Store, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(absolute, "development.db")

	err = prepareDatabase(path)
	if err != nil {
		return nil, err
	}

	location := new(url.URL)
	location.Scheme = "file"
	location.Path = path
	location.RawQuery = "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

	database, err := sql.Open("sqlite", location.String())
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(1)

	store := &Store{database: database}

	err = store.initialize(ctx)
	if err == nil {
		_, err = store.Snapshot(ctx)
	}

	if err != nil {
		_ = database.Close()

		return nil, err
	}

	return store, nil
}

func (store *Store) Close() error { return store.database.Close() }

func (store *Store) Snapshot(ctx context.Context) (Board, error) {
	transaction, err := store.database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: 0})
	if err != nil {
		return Board{}, err
	}

	defer func() { _ = transaction.Rollback() }()

	return readBoard(ctx, transaction)
}

func (store *Store) Change(ctx context.Context, mutation Mutation) (Board, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return Board{}, err
	}

	defer func() { _ = transaction.Rollback() }()

	board, err := readBoard(ctx, transaction)
	if err != nil {
		return Board{}, err
	}

	if board.Revision >= math.MaxInt64 {
		return Board{}, ErrConflict
	}

	err = applyMutation(&board, mutation)
	if err != nil {
		return Board{}, err
	}

	board.Revision++
	board.UpdatedAt = time.Now().UTC()

	err = saveMutation(ctx, transaction, board, mutation)
	if err != nil {
		return Board{}, err
	}

	return board, transaction.Commit()
}

func prepareDatabase(path string) error {
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return ErrInvalid
		}

		return os.Chmod(path, databasePermission)
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	//nolint:gosec // Fixed installation-owned path; exclusive creation rejects replacement symlinks.
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, databasePermission)
	if err != nil {
		return err
	}

	return file.Close()
}

func (store *Store) initialize(ctx context.Context) error {
	var version int

	err := store.database.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err != nil {
		return err
	}

	if version < 0 || version > 1 {
		return ErrInvalid
	}

	if version == 1 {
		return nil
	}

	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() { _ = transaction.Rollback() }()

	_, err = transaction.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS metadata (
 id INTEGER PRIMARY KEY CHECK(id = 1), revision INTEGER NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tasks (
 id TEXT PRIMARY KEY, version INTEGER NOT NULL, title TEXT NOT NULL, description TEXT NOT NULL,
 status TEXT NOT NULL, priority TEXT NOT NULL, category TEXT NOT NULL, notes TEXT NOT NULL, refs TEXT NOT NULL
);
PRAGMA user_version = 1;`)
	if err != nil {
		return err
	}

	if version == 0 {
		board, seedErr := seedBoard()
		if seedErr != nil {
			return seedErr
		}

		board.UpdatedAt = time.Now().UTC()

		saveErr := saveBoard(ctx, transaction, board)
		if saveErr != nil {
			return saveErr
		}
	}

	return transaction.Commit()
}

func readBoard(ctx context.Context, transaction *sql.Tx) (Board, error) {
	board := Board{Schema: 1, Revision: 0, UpdatedAt: time.Time{}, Tasks: []Task{}}

	var updated string

	err := transaction.QueryRowContext(ctx, "SELECT revision, updated_at FROM metadata WHERE id = 1").Scan(
		&board.Revision, &updated)
	if err != nil {
		return Board{}, err
	}

	if board.Revision == 0 {
		return Board{}, ErrInvalid
	}

	board.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return Board{}, err
	}

	rows, err := transaction.QueryContext(ctx,
		"SELECT id, version, title, description, status, priority, category, notes, refs FROM tasks ORDER BY id")
	if err != nil {
		return Board{}, err
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			task       Task
			references string
		)

		err = rows.Scan(&task.ID, &task.Version, &task.Title, &task.Description,
			&task.Status, &task.Priority, &task.Category, &task.Notes, &references)
		if err != nil {
			return Board{}, err
		}

		if decodeTaskRecord(&task, references) != nil || len(board.Tasks) >= TaskLimit {
			return Board{}, ErrInvalid
		}

		board.Tasks = append(board.Tasks, task)
	}

	return board, rows.Err()
}

func decodeTaskRecord(task *Task, references string) error {
	if json.Unmarshal([]byte(references), &task.References) != nil || !validTask(*task) || task.Version == 0 {
		return ErrInvalid
	}

	if task.References == nil {
		task.References = []string{}
	}

	return nil
}

func saveBoard(ctx context.Context, transaction *sql.Tx, board Board) error {
	for _, task := range board.Tasks {
		err := saveTask(ctx, transaction, task)
		if err != nil {
			return err
		}
	}

	return saveMetadata(ctx, transaction, board)
}

func saveMutation(ctx context.Context, transaction *sql.Tx, board Board, mutation Mutation) error {
	err := saveRecord(ctx, transaction, board, mutation)
	if err != nil {
		return err
	}

	return saveMetadata(ctx, transaction, board)
}

func saveRecord(ctx context.Context, transaction *sql.Tx, board Board, mutation Mutation) error {
	if mutation.Operation == Delete {
		_, err := transaction.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", mutation.ID)

		return err
	}

	for _, task := range board.Tasks {
		created := mutation.Operation == Create && task.ID == board.Tasks[len(board.Tasks)-1].ID
		if task.ID == mutation.ID || created {
			return saveTask(ctx, transaction, task)
		}
	}

	return ErrAbsent
}

func saveTask(ctx context.Context, transaction *sql.Tx, task Task) error {
	references, err := json.Marshal(task.References)
	if err != nil {
		return err
	}

	_, err = transaction.ExecContext(ctx, `INSERT INTO tasks
(id,version,title,description,status,priority,category,notes,refs) VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET version=excluded.version, title=excluded.title, description=excluded.description,
status=excluded.status, priority=excluded.priority, category=excluded.category,
notes=excluded.notes, refs=excluded.refs`,
		task.ID, task.Version, task.Title, task.Description, task.Status, task.Priority, task.Category, task.Notes,
		string(references))

	return err
}

func saveMetadata(ctx context.Context, transaction *sql.Tx, board Board) error {
	_, err := transaction.ExecContext(ctx, `INSERT INTO metadata (id,revision,updated_at) VALUES(1,?,?)
ON CONFLICT(id) DO UPDATE SET revision=excluded.revision, updated_at=excluded.updated_at`,
		board.Revision, board.UpdatedAt.Format(time.RFC3339Nano))

	return err
}
