package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kizuna-org/akari/internal/development"
	"github.com/kizuna-org/akari/internal/docrepo"
)

func TestMCPClientControlsTheSharedWorkspace(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"streamable HTTP client"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store, err := development.Open(t.Context(), t.TempDir())

			must(t, err)

			defer func() { must(t, store.Close()) }()

			documents, err := docrepo.Open(t.Context(), filepath.Join(t.TempDir(), "docs"), true,
				[]docrepo.Document{{Path: "docs/README.md", Content: "# Initial\n", Revision: ""},
					{Path: "design/README.md", Content: "# Design\n", Revision: ""}})
			must(t, err)

			server := httptest.NewServer(New(store, documents))
			defer server.Close()

			implementation := new(mcp.Implementation)
			implementation.Name = "Akari integration test"
			client := mcp.NewClient(implementation, nil)
			transport := new(mcp.StreamableClientTransport)
			transport.Endpoint = server.URL
			transport.DisableStandaloneSSE = true
			session, err := client.Connect(t.Context(), transport, nil)

			must(t, err)

			defer func() { must(t, session.Close()) }()

			tools, err := session.ListTools(t.Context(), nil)
			must(t, err)
			check(t, len(tools.Tools) == 8, "MCP tool discovery failed")

			board := call[development.Board](t, session, "tasks_list", map[string]any{})
			check(t, len(board.Tasks) > 20, "seed tasks not available over MCP")
			filtered := call[development.Board](t, session, "tasks_list", map[string]any{"status": "decision",
				"priority": "high"})
			check(t, len(filtered.Tasks) > 0 && len(filtered.Tasks) < len(board.Tasks), "task filters did not apply")

			created := call[development.Board](t, session, "tasks_create", map[string]any{"title": "MCP controlled task"})
			task := findTitle(t, created, "MCP controlled task")
			check(t, task.Version == 1 && task.Status == "todo", "task defaults not applied")
			task.Status = "in_progress"
			task.Notes = "changed through MCP"
			_ = call[development.Board](t, session, "tasks_update", map[string]any{"task": task})
			failure(t, session, "tasks_update", map[string]any{"task": task})

			actual := call[development.Task](t, session, "tasks_get", map[string]any{"id": task.ID})
			check(t, actual.Version == 2 && actual.Notes == task.Notes, "MCP update was not saved")
			persisted, err := store.Snapshot(t.Context())
			must(t, err)
			check(t, findTitle(t, persisted, task.Title).Version == actual.Version, "MCP and console do not share the same DB")

			library := call[[]docrepo.Document](t, session, "documents_list", map[string]any{})
			check(t, len(library) == 2, "document listing failed")
			document := call[docrepo.Document](t, session, "documents_read", map[string]any{"path": "docs/README.md"})
			document.Content = "# Edited from MCP\n"
			updated := call[docrepo.Document](t, session, "documents_write", map[string]any{"document": document})
			check(t, updated.Revision != document.Revision, "MCP did not commit the Markdown edit")
			failure(t, session, "documents_write", map[string]any{"document": document})
			failure(t, session, "documents_read", map[string]any{"path": "../private.md"})

			_ = call[development.Board](t, session, "tasks_delete", map[string]any{
				"id": actual.ID, "expectedVersion": actual.Version,
			})

			failure(t, session, "tasks_get", map[string]any{"id": actual.ID})
		})
	}
}

func call[T any](t *testing.T, session *mcp.ClientSession, name string, input map[string]any) T {
	t.Helper()

	params := new(mcp.CallToolParams)
	params.Name = name
	params.Arguments = input
	result, err := session.CallTool(t.Context(), params)
	must(t, err)

	if result.IsError {
		t.Fatalf("%s returned a tool error: %v", name, result.Content)
	}

	data, err := json.Marshal(result.StructuredContent)
	must(t, err)

	var value T

	must(t, json.Unmarshal(data, &value))

	return value
}

func failure(t *testing.T, session *mcp.ClientSession, name string, input map[string]any) {
	t.Helper()

	params := new(mcp.CallToolParams)
	params.Name = name
	params.Arguments = input
	result, err := session.CallTool(t.Context(), params)
	must(t, err)
	check(t, result.IsError, "stale or unsafe tool input was accepted")
}

func findTitle(t *testing.T, board development.Board, title string) development.Task {
	t.Helper()

	for _, task := range board.Tasks {
		if task.Title == title {
			return task
		}
	}

	t.Fatal("task missing")

	var task development.Task

	return task
}

func check(t *testing.T, value bool, message string) {
	t.Helper()

	if !value {
		t.Fatal(message)
	}
}

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
