// Package console provides an authenticated, read-only administrative console.
package console

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kizuna-org/akari/internal/continuity"
)

//go:generate go run ../../tools/docBundle

//go:embed assets
var assets embed.FS

//go:embed documents.json
var documents []byte

const cookieName = "akari_console"

// Reader is management-only. Its raw export never crosses this console's HTTP boundary.
type Reader interface {
	Export() continuity.Frame
}

func NewHandler(token string, reader Reader) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", asset("index.html"))
	mux.HandleFunc("GET /assets/{name}", func(writer http.ResponseWriter, request *http.Request) {
		asset(request.PathValue("name"))(writer, request)
	})
	mux.HandleFunc("POST /api/session", login(token))
	mux.HandleFunc("DELETE /api/session", logout)
	mux.Handle("GET /api/docs", authorize(token, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(documents)
	})))
	mux.Handle("GET /api/status", authorize(token, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(project(reader.Export()))
	})))

	return headers(mux)
}

func asset(name string) http.HandlerFunc {
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

func authorize(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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

func login(token string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
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

		http.SetCookie(writer, sessionCookie(token, request.TLS != nil, 0))
		writer.WriteHeader(http.StatusNoContent)
	}
}

func logout(writer http.ResponseWriter, request *http.Request) {
	if !sameOrigin(request) {
		http.Error(writer, "same-origin request required", http.StatusForbidden)

		return
	}

	http.SetCookie(writer, sessionCookie("", request.TLS != nil, -1))
	writer.WriteHeader(http.StatusNoContent)
}

func sameOrigin(request *http.Request) bool {
	scheme := "http://"
	if request.TLS != nil {
		scheme = "https://"
	}

	return request.Header.Get("Origin") == scheme+request.Host
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
