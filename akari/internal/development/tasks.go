// Package development owns developer task metadata, separate from Akari's mind.
package development

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"time"
)

type taskError string

func (err taskError) Error() string { return string(err) }

const (
	ErrInvalid  taskError = "development: invalid task or database schema"
	ErrConflict taskError = "development: task changed; read the current version before retrying"
	ErrAbsent   taskError = "development: task not found"
	TaskLimit             = 256
	Create                = "create"
	Update                = "update"
	Delete                = "delete"
	InProgress            = "in_progress"
)

type Task struct {
	ID          string   `json:"id"`
	Version     uint64   `json:"version"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Category    string   `json:"category"`
	Notes       string   `json:"notes"`
	References  []string `json:"references"`
}

type Board struct {
	Schema    int       `json:"schema"`
	Revision  uint64    `json:"revision"`
	UpdatedAt time.Time `json:"updatedAt"`
	Tasks     []Task    `json:"tasks"`
}

// Mutation uses ExpectedRevision to guard board edits. ExpectedVersion guards
// one task for API/MCP clients, allowing unrelated task edits without conflict.
type Mutation struct {
	Operation        string
	ID               string
	Task             Task
	ExpectedRevision uint64
	ExpectedVersion  uint64
}

func Decode(input io.Reader, value any) error {
	decoder := json.NewDecoder(input)

	var raw json.RawMessage

	if decoder.Decode(&raw) != nil || len(raw) == 0 || raw[0] != '{' {
		return ErrInvalid
	}

	var trailing any

	if !errors.Is(decoder.Decode(&trailing), io.EOF) {
		return ErrInvalid
	}

	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()

	if strict.Decode(value) != nil {
		return ErrInvalid
	}

	return nil
}

func validTask(task Task) bool {
	return validTaskText(task) && validTaskChoices(task) && validReferences(task.References)
}

func validTaskText(task Task) bool {
	return task.ID != "" && len(task.ID) <= 64 && !strings.ContainsAny(task.ID, "/\\?# \t\n") &&
		strings.TrimSpace(task.Title) != "" && len(task.Title) <= 480 && len(task.Description) <= 12000 &&
		len(task.Notes) <= 24000
}

func validTaskChoices(task Task) bool {
	return slices.Contains([]string{"todo", InProgress, "blocked", "decision", "done"}, task.Status) &&
		slices.Contains([]string{"high", "medium", "low"}, task.Priority) &&
		slices.Contains([]string{"runtime", "model", "tools", "inner", "console", "operations", "decision"}, task.Category)
}

func validReferences(references []string) bool {
	const maxReferences = 8

	if len(references) > maxReferences {
		return false
	}

	for _, reference := range references {
		path, _, _ := strings.Cut(reference, "#")
		parts := strings.Split(path, "/")

		if len(reference) > 512 || len(parts) != 2 || !slices.Contains([]string{"docs", "design"}, parts[0]) ||
			!strings.HasSuffix(parts[1], ".md") || strings.ContainsAny(parts[1], "\\?#") || parts[1] == ".md" {
			return false
		}
	}

	return true
}

func applyMutation(board *Board, mutation Mutation) error {
	index := slices.IndexFunc(board.Tasks, func(task Task) bool { return task.ID == mutation.ID })

	if mutation.ExpectedRevision != 0 && mutation.ExpectedRevision != board.Revision {
		return ErrConflict
	}

	if mutation.Operation == Create {
		return addTask(board, mutation.Task)
	}

	if index < 0 {
		return ErrAbsent
	}

	current := board.Tasks[index]
	if !matchingVersion(current, mutation) {
		return ErrConflict
	}

	switch mutation.Operation {
	case Delete:
		board.Tasks = slices.Delete(board.Tasks, index, index+1)
	case Update:
		return replaceTask(board, index, mutation)
	default:
		return ErrInvalid
	}

	return nil
}

func replaceTask(board *Board, index int, mutation Mutation) error {
	if mutation.Task.ID != mutation.ID || !validTask(mutation.Task) {
		return ErrInvalid
	}

	mutation.Task.Version = board.Tasks[index].Version + 1
	board.Tasks[index] = mutation.Task

	return nil
}

func addTask(board *Board, task Task) error {
	if task.ID != "" || len(board.Tasks) >= TaskLimit {
		return ErrInvalid
	}

	task.ID = rand.Text()

	task.Version = 1

	if task.References == nil {
		task.References = []string{}
	}

	if !validTask(task) {
		return ErrInvalid
	}

	board.Tasks = append(board.Tasks, task)

	return nil
}

func seedBoard() (Board, error) {
	var board Board

	err := Decode(bytes.NewReader(taskSeed), &board)
	if err != nil {
		return Board{}, err
	}

	for index := range board.Tasks {
		board.Tasks[index].Version = 1

		if !validTask(board.Tasks[index]) {
			return Board{}, ErrInvalid
		}
	}

	return board, nil
}

func matchingVersion(current Task, mutation Mutation) bool {
	return mutation.ExpectedRevision != 0 || (mutation.ExpectedVersion > 0 && mutation.ExpectedVersion == current.Version)
}
