package console

import (
	"errors"
	"net/http"

	"github.com/kizuna-org/akari/internal/development"
	"github.com/kizuna-org/akari/internal/docrepo"
)

func documentWriter(repository *docrepo.Repository) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if !sameOrigin(request) {
			http.Error(writer, "same-origin request required", http.StatusForbidden)

			return
		}

		var document docrepo.Document

		const maxBody = 2 << 20

		if development.Decode(http.MaxBytesReader(writer, request.Body, maxBody), &document) != nil {
			http.Error(writer, "invalid Markdown document", http.StatusBadRequest)

			return
		}

		updated, err := repository.Write(request.Context(), document)
		if err != nil {
			code := http.StatusServiceUnavailable
			if errors.Is(err, docrepo.ErrConflict) {
				code = http.StatusConflict
			}

			if errors.Is(err, docrepo.ErrInvalid) {
				code = http.StatusBadRequest
			}

			http.Error(writer, "document update failed; read current revision before retrying", code)

			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writeJSON(writer, updated)
	}
}

func mcpBoundary(authDisabled, accessAuthenticated bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if (origin != "" && !sameOrigin(request)) ||
			(origin == "" && !authDisabled && !accessAuthenticated &&
				request.Header.Get("Authorization") == "") {
			http.Error(writer, "MCP requires bearer authentication or same-origin console access", http.StatusForbidden)

			return
		}

		const maxBody = 2 << 20

		request.Body = http.MaxBytesReader(writer, request.Body, maxBody)
		next.ServeHTTP(writer, request)
	})
}
