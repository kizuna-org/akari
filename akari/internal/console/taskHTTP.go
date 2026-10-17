package console

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/kizuna-org/akari/internal/development"
)

const taskAPI = "/api/development/tasks"

func taskRoutes(mux *http.ServeMux, token string, authDisabled bool, store *development.Store) {
	for _, pattern := range []string{
		"GET " + taskAPI, "GET " + taskAPI + "/{id}", "POST " + taskAPI,
		"PUT " + taskAPI + "/{id}", "DELETE " + taskAPI + "/{id}",
	} {
		mux.Handle(pattern, authorize(token, authDisabled, taskHandler(store)))
	}
}

func taskHandler(store *development.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if store == nil {
			http.Error(writer, "task store unavailable", http.StatusServiceUnavailable)

			return
		}

		if request.Method != http.MethodGet && !sameOrigin(request) {
			http.Error(writer, "same-origin request required", http.StatusForbidden)

			return
		}

		board, err := taskOperation(writer, request, store)
		if err != nil {
			taskHTTPError(writer, err)

			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("ETag", `"`+strconv.FormatUint(board.Revision, 10)+`"`)

		if request.Method == http.MethodGet && request.PathValue("id") != "" {
			for _, task := range board.Tasks {
				if task.ID == request.PathValue("id") {
					writer.Header().Set("ETag", `"`+strconv.FormatUint(task.Version, 10)+`"`)
					writeJSON(writer, task)

					return
				}
			}

			taskHTTPError(writer, development.ErrAbsent)

			return
		}

		writeJSON(writer, board)
	}
}

func taskOperation(writer http.ResponseWriter, request *http.Request, store *development.Store) (
	development.Board, error,
) {
	if request.Method == http.MethodGet {
		return store.Snapshot(request.Context())
	}

	expected, err := strconv.ParseUint(strings.Trim(request.Header.Get("If-Match"), `"`), 10, 64)
	if err != nil || expected == 0 {
		return development.Board{}, development.ErrConflict
	}

	var task development.Task

	if request.Method != http.MethodDelete {
		const maxBody = 64 << 10

		mediaType, _, parseErr := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if parseErr != nil || mediaType != "application/json" ||
			development.Decode(http.MaxBytesReader(writer, request.Body, maxBody), &task) != nil {
			return development.Board{}, development.ErrInvalid
		}
	}

	operation := map[string]string{http.MethodPost: development.Create, http.MethodPut: development.Update,
		http.MethodDelete: development.Delete}[request.Method]

	mutation := development.Mutation{Operation: operation, ID: request.PathValue("id"), Task: task,
		ExpectedRevision: 0, ExpectedVersion: expected}
	if request.Method == http.MethodPost {
		mutation.ExpectedRevision = expected
		mutation.ExpectedVersion = 0
	}

	return store.Change(request.Context(), mutation)
}

func taskHTTPError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, development.ErrConflict):
		http.Error(writer, "task conflict: read the current version before retrying", http.StatusConflict)
	case errors.Is(err, development.ErrInvalid):
		http.Error(writer, "invalid task fields or document references", http.StatusBadRequest)
	case errors.Is(err, development.ErrAbsent):
		http.Error(writer, "task not found", http.StatusNotFound)
	default:
		http.Error(writer, "task storage unavailable; reload to verify the saved state", http.StatusServiceUnavailable)
	}
}
