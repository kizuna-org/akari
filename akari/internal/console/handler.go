// Package console provides runtime observation and developer-owned task management.
package console

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/development"
	"github.com/kizuna-org/akari/internal/docrepo"
	"github.com/kizuna-org/akari/internal/mcpserver"
)

//go:generate go run ../../tools/docBundle

//go:embed assets
var assets embed.FS

//go:embed documents.json
var documents []byte

const cookieName = "akari_console"
const schemeHTTPS = "https"

// Reader is management-only. Its raw export never crosses this console's HTTP boundary.
type Reader interface {
	Export() continuity.Frame
}

func NewHandler(token string, reader Reader, authDisabled bool) http.Handler {
	return newHandler(token, reader, authDisabled, workspace{tasks: nil, documents: nil})
}

type workspace struct {
	tasks     *development.Store
	documents *docrepo.Repository
}

func newHandler(token string, reader Reader, authDisabled bool, work workspace) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", asset("index.html", authDisabled))
	mux.HandleFunc("GET /assets/{name}", func(writer http.ResponseWriter, request *http.Request) {
		asset(request.PathValue("name"), authDisabled)(writer, request)
	})
	mux.HandleFunc("POST /api/session", login(token, authDisabled))
	mux.HandleFunc("DELETE /api/session", logout(authDisabled))

	docsHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		if work.documents != nil {
			library, err := work.documents.List(request.Context())
			if err != nil {
				http.Error(writer, "documentation unavailable", http.StatusServiceUnavailable)

				return
			}

			writeJSON(writer, library)

			return
		}

		_, _ = writer.Write(documents)
	})
	mux.Handle("GET /api/docs", authorize(token, authDisabled, docsHandler))

	mux.Handle("GET /api/status", authorize(token, authDisabled, statusHandler(reader)))
	taskRoutes(mux, token, authDisabled, work.tasks)

	if work.tasks != nil && work.documents != nil {
		mux.Handle("PUT /api/docs", authorize(token, authDisabled, documentWriter(work.documents)))
		mux.Handle("/mcp", authorize(token, authDisabled, mcpBoundary(authDisabled,
			mcpserver.New(work.tasks, work.documents))))
	}

	return headers(mux)
}

func statusHandler(reader Reader) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		data, err := json.Marshal(project(reader.Export()))
		if err != nil {
			http.Error(writer, "status unavailable", http.StatusInternalServerError)

			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(data)
	}
}

func asset(name string, authDisabled bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if strings.ContainsAny(name, "/\\") || strings.HasPrefix(name, ".") {
			http.NotFound(writer, request)

			return
		}

		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			http.NotFound(writer, request)

			return
		}

		if name == "index.html" {
			data = []byte(strings.ReplaceAll(string(data), `data-auth-disabled="false"`,
				`data-auth-disabled="`+strconv.FormatBool(authDisabled)+`"`))
		}

		switch {
		case strings.HasSuffix(name, ".js"):
			writer.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		case strings.HasSuffix(name, ".css"):
			writer.Header().Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(name, ".html"):
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		default:
			writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}

		_, _ = writer.Write(data)
	}
}

func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; "+
			"style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; "+
			"frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		next.ServeHTTP(writer, request)
	})
}

func authorize(token string, authDisabled bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if authDisabled {
			next.ServeHTTP(writer, request)

			return
		}

		provided := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")

		cookie, err := request.Cookie(cookieName)
		if err == nil {
			provided = cookie.Value
		}

		if token == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			http.Error(writer, "management authentication required", http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(writer, request)
	})
}

func login(token string, authDisabled bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if authDisabled {
			writer.WriteHeader(http.StatusNoContent)

			return
		}

		if !sameOrigin(request) {
			http.Error(writer, "same-origin request required", http.StatusForbidden)

			return
		}

		var input struct {
			Token string `json:"token"`
		}

		const maxBody = 1024

		err := json.NewDecoder(http.MaxBytesReader(writer, request.Body, maxBody)).Decode(&input)
		if err != nil || token == "" || subtle.ConstantTimeCompare([]byte(input.Token), []byte(token)) != 1 {
			http.Error(writer, "invalid management code", http.StatusUnauthorized)

			return
		}

		http.SetCookie(writer, sessionCookie(token, requestScheme(request) == schemeHTTPS, 0))
		writer.WriteHeader(http.StatusNoContent)
	}
}

func logout(authDisabled bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if authDisabled {
			writer.WriteHeader(http.StatusNoContent)

			return
		}

		if !sameOrigin(request) {
			http.Error(writer, "same-origin request required", http.StatusForbidden)

			return
		}

		http.SetCookie(writer, sessionCookie("", requestScheme(request) == schemeHTTPS, -1))
		writer.WriteHeader(http.StatusNoContent)
	}
}

func sameOrigin(request *http.Request) bool {
	return request.Header.Get("Origin") == requestScheme(request)+"://"+request.Host
}

func requestScheme(request *http.Request) string {
	if request.TLS != nil {
		return schemeHTTPS
	}
	// Reverse proxies terminate TLS before forwarding the request over HTTP.
	// Use the first forwarded value, following the standard proxy header format.
	forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")[0])
	if forwarded == schemeHTTPS {
		return schemeHTTPS
	}

	return "http"
}

func sessionCookie(token string, secure bool, maxAge int) *http.Cookie {
	//nolint:gosec // Local HTTP console; Secure is enabled on TLS, HttpOnly and SameSite are always enforced.
	cookie := new(http.Cookie)
	cookie.Name = cookieName
	cookie.Value = token
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.Secure = secure
	cookie.SameSite = http.SameSiteStrictMode
	cookie.MaxAge = maxAge

	return cookie
}
