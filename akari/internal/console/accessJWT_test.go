//nolint:wsl,wsl_v5 // Table-driven security cases are grouped for readability.
package console

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testAudienceValue = "aud"

func TestAccessJWTVerifier(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require(t, err)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require(t, err)

	const (
		audience = "akari-app-aud"
		keyID    = "first-key"
	)

	var mutex sync.RWMutex
	var keySet = accessTestJWKS(t, keyID, &key.PublicKey)
	fetches := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()

		fetches++
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(keySet)
	}))
	t.Cleanup(server.Close)
	issuer := server.URL

	verifier := &accessJWTVerifier{
		issuer:                server.URL,
		audience:              audience,
		client:                server.Client(),
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  make(map[string]*rsa.PublicKey),
		cacheTill:             time.Time{},
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Time{},
	}

	valid := accessTestToken(t, key, keyID, issuer, audience, time.Now().Add(time.Minute))
	verificationErr := verifier.verify(t.Context(), valid)
	if verificationErr != nil {
		t.Fatalf("valid Access assertion was rejected: %v", verificationErr)
	}

	for _, invalid := range []string{
		accessTestToken(t, key, keyID, "https://other.example", audience, time.Now().Add(time.Minute)),
		accessTestToken(t, key, keyID, issuer, "other-aud", time.Now().Add(time.Minute)),
		accessTestToken(t, key, keyID, issuer, audience, time.Now().Add(-time.Minute)),
		accessTestToken(t, otherKey, keyID, issuer, audience, time.Now().Add(time.Minute)),
	} {
		verificationErr = verifier.verify(t.Context(), invalid)
		if verificationErr == nil {
			t.Fatal("invalid Access assertion was accepted")
		}
	}

	mutex.Lock()
	fetchCount := fetches
	mutex.Unlock()
	if fetchCount != 1 {
		t.Fatalf("cached key set fetched %d times, want 1", fetchCount)
	}

	rotatedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require(t, err)
	mutex.Lock()
	keySet = accessTestJWKS(t, "rotated-key", &rotatedKey.PublicKey)
	mutex.Unlock()

	rotated := accessTestToken(t, rotatedKey, "rotated-key", issuer, audience, time.Now().Add(time.Minute))
	verificationErr = verifier.verify(t.Context(), rotated)
	if verificationErr != nil {
		t.Fatalf("assertion signed by a newly published key was rejected: %v", verificationErr)
	}

	mutex.Lock()
	fetchCount = fetches
	mutex.Unlock()
	if fetchCount != 2 {
		t.Fatalf("rotated key set fetched %d times, want 2", fetchCount)
	}

	unknown := accessTestToken(t, rotatedKey, "unknown-key", issuer, audience, time.Now().Add(time.Minute))
	verificationErr = verifier.verify(t.Context(), unknown)
	if verificationErr == nil {
		t.Fatal("assertion with unknown key ID was accepted")
	}

	mutex.Lock()
	fetchCount = fetches
	mutex.Unlock()
	if fetchCount != 2 {
		t.Fatalf("unknown key triggered an unthrottled refresh; got %d fetches", fetchCount)
	}
}

func TestAccessJWTVerifierBacksOffAfterConcurrentRefreshFailures(t *testing.T) {
	t.Parallel()

	var fetches atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		fetches.Add(1)
		http.Error(writer, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	verifier := &accessJWTVerifier{
		issuer:                server.URL,
		audience:              testAudienceValue,
		client:                server.Client(),
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  make(map[string]*rsa.PublicKey),
		cacheTill:             time.Time{},
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Time{},
	}

	const requests = 16
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(requests)
	for range requests {
		go func() {
			defer workers.Done()
			<-start
			_, err := verifier.publicKey(t.Context(), "access-key")
			if err == nil {
				t.Error("publicKey succeeded while the signing key endpoint was unavailable")
			}
		}()
	}
	close(start)
	workers.Wait()

	if got := fetches.Load(); got != 1 {
		t.Fatalf("concurrent failed requests fetched JWKS %d times, want 1", got)
	}
}

func TestAccessJWTVerifierServesCachedKeyDuringBlockedRefresh(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require(t, err)
	keySet := accessTestJWKS(t, "rotated-key", &key.PublicKey)
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseRefresh := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseRefresh()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		_, _ = writer.Write(keySet)
	}))
	t.Cleanup(server.Close)

	verifier := &accessJWTVerifier{
		issuer:                server.URL,
		audience:              testAudienceValue,
		client:                server.Client(),
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  map[string]*rsa.PublicKey{"known-key": &key.PublicKey},
		cacheTill:             time.Now().Add(time.Minute),
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Time{},
	}

	unknownResult := make(chan error, 2)
	go func() {
		_, keyErr := verifier.publicKey(t.Context(), "rotated-key")
		unknownResult <- keyErr
	}()
	<-started
	go func() {
		_, keyErr := verifier.publicKey(t.Context(), "rotated-key")
		unknownResult <- keyErr
	}()

	cachedResult := make(chan error, 1)
	go func() {
		_, keyErr := verifier.publicKey(t.Context(), "known-key")
		cachedResult <- keyErr
	}()
	select {
	case err := <-cachedResult:
		if err != nil {
			t.Fatalf("cached key lookup failed during refresh: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("cached key lookup blocked behind a signing-key refresh")
	}

	releaseRefresh()
	for range 2 {
		err = <-unknownResult
		if err != nil {
			t.Fatalf("concurrent rotated-key lookup failed: %v", err)
		}
	}
}

func TestAccessJWTVerifierCanceledRefreshDoesNotBackOffHealthyRequests(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require(t, err)
	keySet := accessTestJWKS(t, "new-key", &key.PublicKey)
	started := make(chan struct{})
	var fetches atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if fetches.Add(1) == 1 {
			close(started)
			<-request.Context().Done()

			return
		}
		_, _ = writer.Write(keySet)
	}))
	t.Cleanup(server.Close)
	verifier := &accessJWTVerifier{
		issuer:                server.URL,
		audience:              testAudienceValue,
		client:                server.Client(),
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  map[string]*rsa.PublicKey{"old-key": &key.PublicKey},
		cacheTill:             time.Now().Add(time.Minute),
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Time{},
	}

	requestCtx, cancel := context.WithCancel(t.Context())
	requestResult := make(chan error, 1)
	go func() {
		_, keyErr := verifier.publicKey(requestCtx, "new-key")
		requestResult <- keyErr
	}()
	<-started
	cancel()
	err = <-requestResult
	if err == nil {
		t.Fatal("canceled signing-key fetch unexpectedly succeeded")
	}

	_, err = verifier.publicKey(t.Context(), "new-key")
	if err != nil {
		t.Fatalf("healthy retry failed after canceled refresh: %v", err)
	}
	if got := fetches.Load(); got != 2 {
		t.Fatalf("signing keys fetched %d times after canceled request, want 2", got)
	}
}

func TestAccessJWTVerifierRechecksUnknownKeyCooldownAfterGateWait(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require(t, err)
	var fetches atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		fetches.Add(1)
	}))
	t.Cleanup(server.Close)
	verifier := &accessJWTVerifier{
		issuer:                server.URL,
		audience:              testAudienceValue,
		client:                server.Client(),
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  map[string]*rsa.PublicKey{"old-key": &key.PublicKey},
		cacheTill:             time.Now().Add(time.Minute),
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Now(),
	}
	verifier.refreshGate <- struct{}{}

	result := make(chan error, 1)
	go func() {
		_, keyErr := verifier.publicKey(t.Context(), "another-unknown-key")
		result <- keyErr
	}()
	<-verifier.refreshGate

	err = <-result
	if !errors.Is(err, errAccessJWTKeyMissing) {
		t.Fatalf("got error %v, want unknown-key error", err)
	}
	if got := fetches.Load(); got != 0 {
		t.Fatalf("cooldown lookup fetched JWKS %d times, want 0", got)
	}
}

func TestAccessJWTConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, issuer, audience string
		configured, wantError  bool
	}{
		{name: "disabled", issuer: "", audience: "", configured: false, wantError: false},
		{name: "incomplete issuer", issuer: "https://access.example", audience: "", configured: true,
			wantError: true},
		{name: "incomplete audience", issuer: "", audience: testAudienceValue,
			configured: true, wantError: true},
		{name: "invalid issuer", issuer: "http://access.example", audience: testAudienceValue, configured: true,
			wantError: true},
		{name: "configured", issuer: "https://access.example", audience: testAudienceValue,
			configured: true, wantError: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(accessIssuerEnv, test.issuer)
			t.Setenv(accessAudienceEnv, test.audience)

			verifier, configured, err := newAccessJWTVerifierFromEnvironment()
			if (err != nil) != test.wantError || configured != test.configured {
				t.Fatalf("got configured=%t error=%v", configured, err)
			}
			if test.configured && !test.wantError && verifier == nil {
				t.Fatal("valid Access verifier configuration returned nil")
			}
		})
	}
}

func TestAccessJWTMCPBoundary(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, origin string
		want         int
	}{
		{name: "verified originless assertion", origin: "", want: http.StatusNoContent},
		{name: "hostile origin", origin: hostileOrigin, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", nil)
			request.Header.Set("Origin", test.origin)
			response := httptest.NewRecorder()

			mcpBoundary(false, true, next).ServeHTTP(response, request)

			if response.Code != test.want {
				t.Fatalf("got %d want %d", response.Code, test.want)
			}
		})
	}
}

func TestAccessJWTMCPIntegration(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require(t, err)

	const audience = "akari-app-aud"

	keys := accessTestJWKS(t, "access-key", &key.PublicKey)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(keys)
	}))
	t.Cleanup(server.Close)
	issuer := server.URL

	verifier := &accessJWTVerifier{
		issuer:                server.URL,
		audience:              audience,
		client:                server.Client(),
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  make(map[string]*rsa.PublicKey),
		cacheTill:             time.Time{},
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Time{},
	}

	for _, test := range []struct {
		name         string
		disabled     bool
		assertion    string
		origin       string
		wantResponse int
	}{
		{name: "verified with admin auth enabled", disabled: false,
			assertion: accessTestToken(t, key, "access-key", issuer, audience, time.Now().Add(time.Minute)),
			origin:    "", wantResponse: http.StatusOK},
		{name: "verified with admin auth disabled", disabled: true,
			assertion: accessTestToken(t, key, "access-key", issuer, audience, time.Now().Add(time.Minute)),
			origin:    "", wantResponse: http.StatusOK},
		{name: "missing assertion", disabled: false, assertion: "", origin: "",
			wantResponse: http.StatusUnauthorized},
		{name: "wrong audience", assertion: accessTestToken(t, key, "access-key",
			issuer, "other-aud", time.Now().Add(time.Minute)), disabled: false,
			origin: "", wantResponse: http.StatusUnauthorized},
		{name: "hostile origin", assertion: accessTestToken(t, key, "access-key",
			issuer, audience, time.Now().Add(time.Minute)), disabled: false,
			origin: hostileOrigin, wantResponse: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			body := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			request.Header.Set("Origin", test.origin)
			if test.assertion != "" {
				request.Header.Set("Cf-Access-Jwt-Assertion", test.assertion)
			}

			response := httptest.NewRecorder()
			handler := newHandlerWithAccess(testToken, staticReader{frame: emptyFrame()},
				test.disabled, testWorkspace(t), verifier)
			handler.ServeHTTP(response, request)

			if response.Code != test.wantResponse {
				t.Fatalf("got %d want %d", response.Code, test.wantResponse)
			}
		})
	}
}

func TestAccessJWTKeyFetchFailure(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require(t, err)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	issuer := server.URL

	verifier := &accessJWTVerifier{
		issuer:                server.URL,
		audience:              "akari-app-aud",
		client:                server.Client(),
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  make(map[string]*rsa.PublicKey),
		cacheTill:             time.Time{},
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Time{},
	}
	assertion := accessTestToken(t, key, "access-key", issuer,
		"akari-app-aud", time.Now().Add(time.Minute))

	err = verifier.verify(t.Context(), assertion)
	if err == nil {
		t.Fatal("assertion passed when Access signing keys could not be fetched")
	}
}

func TestAccessJWTMiddlewareRejectsMissingAssertion(t *testing.T) {
	t.Parallel()

	verifier := &accessJWTVerifier{
		issuer:                "",
		audience:              "",
		client:                nil,
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  nil,
		cacheTill:             time.Time{},
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Time{},
	}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", nil)
	response := httptest.NewRecorder()
	verifier.middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler ran without an Access assertion")
	})).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("got %d want %d", response.Code, http.StatusUnauthorized)
	}
}

func accessTestToken(t *testing.T, key *rsa.PrivateKey, keyID, issuer, audience string,
	expires time.Time) string {
	t.Helper()

	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   "",
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(expires),
		NotBefore: nil,
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		ID:        "",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	signed, err := token.SignedString(key)
	require(t, err)

	return signed
}

func accessTestJWKS(t *testing.T, keyID string, key *rsa.PublicKey) []byte {
	t.Helper()

	exponent := make([]byte, 0)
	for value := key.E; value > 0; value >>= accessBitsPerByte {
		exponent = append([]byte{byte(value)}, exponent...)
	}
	encoded, err := json.Marshal(map[string][]map[string]string{
		"keys": {{
			"kid": keyID,
			"kty": "RSA",
			"alg": "RS256",
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(exponent),
		}},
	})
	require(t, err)

	return encoded
}
