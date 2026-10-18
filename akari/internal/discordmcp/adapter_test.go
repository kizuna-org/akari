package discordmcp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/action"
)

const (
	testSecret         = "secret"
	subscribeMethod    = "events/subscribe"
	testSubscription   = "sub1"
	testResultKey      = "result"
	testDiscoverMethod = "server/discover"
	testMentionArg     = "direct_bot_mention"
	testToolCallMethod = "tools/call"
	testUnsubscribe    = "events/unsubscribe"
	oldTestSession     = "old-session"
	newTestSession     = "new-session"
)

func newClient(endpoint string) *Client {
	return &Client{
		Endpoint: endpoint, Token: "", HTTP: nil,
		mu: sync.Mutex{}, session: "", next: atomic.Uint64{}, Timeout: 0,
	}
}

func newAdapter(
	client *Client,
	callbackURL, secret string,
	handler func(context.Context, Event) error,
	now func() time.Time,
) *Adapter {
	return &Adapter{
		Client: client, CallbackURL: callbackURL, Secret: secret,
		Handler: handler, Now: now,
	}
}

func TestVerifyWebhook(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_800_000_000, 0)
	tests := []struct {
		name, timestamp, signature string
		valid                      bool
	}{
		{name: "valid", timestamp: strconv.FormatInt(now.Unix(), 10), signature: "", valid: true},
		{name: "bad timestamp", timestamp: "invalid", signature: "", valid: false},
		{name: "stale", timestamp: strconv.FormatInt(now.Add(-6*time.Minute).Unix(), 10), signature: "", valid: false},
		{name: "tampered", timestamp: strconv.FormatInt(now.Unix(), 10), signature: "", valid: false},
	}
	body := []byte(`{"eventId":"e1"}`)

	for i := range tests {
		test := &tests[i]
		if test.valid {
			mac := hmac.New(sha256.New, []byte(testSecret))
			_, _ = mac.Write([]byte("id." + test.timestamp + "." + string(body)))
			test.signature = "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := VerifyWebhook(testSecret, "id", test.timestamp, test.signature, body, now)
			if got != test.valid {
				t.Fatalf("got %v, want %v", got, test.valid)
			}
		})
	}
}

func TestServeHTTPVerificationAndEvent(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name, body string
		want       int
		handlerErr bool
	}{
		{name: "challenge", body: `{"type":"verification","challenge":"nonce"}`,
			want: http.StatusOK, handlerErr: false},
		{name: "message", body: `{"eventId":"e1","name":"discord.message.created","data":{"message_id":"m1"}}`,
			want: http.StatusNoContent, handlerErr: false},
		{name: "handler retry", body: `{"eventId":"e2","name":"discord.message.created","data":{}}`,
			want: http.StatusServiceUnavailable, handlerErr: true},
		{name: "reject unrelated event", body: `{"eventId":"e3","name":"discord.channel.created","data":{}}`,
			want: http.StatusBadRequest, handlerErr: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			adapter := newAdapter(nil, "", testSecret, func(_ context.Context, e Event) error {
				if e.Name != "discord.message.created" {
					t.Errorf("unexpected event %q", e.Name)
				}

				if test.handlerErr {
					return context.Canceled
				}

				return nil
			}, func() time.Time { return now })
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/events", strings.NewReader(test.body))
			webhookTimestamp := strconv.FormatInt(now.Unix(), 10)

			req.Header.Set("Webhook-Id", "w1")
			req.Header.Set("Webhook-Timestamp", webhookTimestamp)

			mac := hmac.New(sha256.New, []byte(testSecret))
			_, _ = mac.Write([]byte("w1." + webhookTimestamp + "." + test.body))
			req.Header.Set("Webhook-Signature", "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))

			rec := httptest.NewRecorder()
			adapter.ServeHTTP(rec, req)

			if rec.Code != test.want {
				t.Fatalf("status %d, want %d (%s)", rec.Code, test.want, rec.Body.String())
			}

			if test.name == "challenge" {
				var got map[string]string

				err := json.Unmarshal(rec.Body.Bytes(), &got)
				if err != nil || got["challenge"] != "nonce" {
					t.Fatalf("challenge response %v, %v", got, err)
				}
			}
		})
	}
}

func TestClientEnforcesSelectedTools(t *testing.T) {
	t.Parallel()

	client := newClient("http://127.0.0.1")

	for _, name := range []string{"discord_voice_join", "discord_list_members", "unknown"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := client.CallTool(context.Background(), name, nil)
			if err == nil {
				t.Errorf("CallTool(%q) unexpectedly allowed", name)
			}
		})
	}
}

func TestClientFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{name: "http status", status: http.StatusUnauthorized, body: `{}`, want: errHTTPStatus},
		{name: "rpc error", status: http.StatusOK, body: `{"error":{"code":-32011,"message":"unknown event"}}`, want: errRPC},
		{name: "malformed response", status: http.StatusOK, body: `not json`, want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			t.Cleanup(server.Close)

			client := newClient(server.URL)

			err := client.Discover(context.Background())
			if err == nil {
				t.Fatal("expected request error")
			}

			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("error %v does not wrap %v", err, test.want)
			}
		})
	}

	t.Run("missing endpoint", func(t *testing.T) {
		t.Parallel()

		err := newClient("").Discover(context.Background())
		if !errors.Is(err, errEndpointRequired) {
			t.Fatalf("error %v", err)
		}
	})
}

//nolint:lll,wsl,wsl_v5 // Keep the compact test matrix and fake response handling readable.
func TestClientToolErrorPreservesPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		result     string
		wantErr    bool
		wantResult string
	}{
		{name: "successful tool result", result: `{"content":[{"type":"text","text":"ok"}]}`,
			wantErr: false, wantResult: `{"content":[{"type":"text","text":"ok"}]}`},
		{name: "tool failure", result: `{"isError":true,"content":[{"type":"text","text":"denied"}]}`, wantErr: true, wantResult: `{"isError":true,"content":[{"type":"text","text":"denied"}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(`{"result":` + test.result + `}`))
			}))
			t.Cleanup(server.Close)

			got, err := newClient(server.URL).CallTool(context.Background(), replyToolName, nil)
			if (err != nil) != test.wantErr {
				t.Fatalf("error %v, wantErr %v", err, test.wantErr)
			}
			if string(got) != test.wantResult {
				t.Fatalf("payload %s, want %s", got, test.wantResult)
			}
		})
	}
}

//nolint:wsl,wsl_v5 // This is an end-to-end Gateway assertion around one fake MCP response.
func TestGatewayReportsMCPToolFailure(t *testing.T) {
	t.Parallel()

	const payload = `{"isError":true,"content":[{"type":"text","text":"Discord rejected message"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"result":` + payload + `}`))
	}))
	t.Cleanup(server.Close)

	adapter := newAdapter(newClient(server.URL), "", "", nil, nil)
	gateway, err := action.New(adapter.Tools(), 1)
	if err != nil {
		t.Fatal(err)
	}
	destination := "discord:channel-1"
	session := gateway.Open(false, action.Scope{AllowedDestinations: []string{destination},
		ReadyRecipients: []string{destination}})
	pending, err := session.Plan(context.Background(), sendToolName,
		[]byte(`{"channel_id":"channel-1","content":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	observed, err := pending.Execute(context.Background())
	if err == nil {
		t.Fatal("tool failure reported success")
	}
	if observed.Status == action.Succeeded {
		t.Fatalf("status %s", observed.Status)
	}
	if string(observed.Data) != payload {
		t.Fatalf("payload %s, want %s", observed.Data, payload)
	}
}

//nolint:wsl,wsl_v5 // Keep the session-recovery scenario in one fake-server test.
func TestClientClearsRejectedSessionWithoutReplayingToolCall(t *testing.T) {
	t.Parallel()

	var methods []string
	var sessions []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		methods = append(methods, request.Header.Get("Mcp-Method"))
		sessions = append(sessions, request.Header.Get("Mcp-Session-Id"))
		if request.Header.Get("Mcp-Method") == testToolCallMethod {
			writer.WriteHeader(http.StatusNotFound)

			return
		}
		writer.Header().Set("Mcp-Session-Id", "fresh")
		_, _ = writer.Write([]byte(`{"result":{}}`))
	}))
	t.Cleanup(server.Close)

	client := newClient(server.URL)
	client.session = "expired"
	_, err := client.CallTool(context.Background(), replyToolName, nil)
	if !errors.Is(err, errHTTPStatus) {
		t.Fatalf("tool call error %v", err)
	}
	if got := strings.Join(methods, ","); got != testToolCallMethod {
		t.Fatalf("tool call replayed: %s", got)
	}
	if client.session != "" {
		t.Fatalf("stale session remains: %q", client.session)
	}
	err = client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(sessions, ","); got != "expired," {
		t.Fatalf("session headers %q", got)
	}
	if client.session != "fresh" {
		t.Fatalf("recovery session %q", client.session)
	}
}

func TestClientClearsOnlyMatchingSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestSession string
		currentSession string
		wantSession    string
	}{
		{name: "matching failed session", requestSession: oldTestSession, currentSession: oldTestSession, wantSession: ""},
		{name: "newer session retained", requestSession: oldTestSession,
			currentSession: newTestSession, wantSession: newTestSession},
		{name: "no session to clear", requestSession: "", currentSession: newTestSession, wantSession: newTestSession},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := newClient("endpoint")
			client.session = test.currentSession
			client.clearSessionIfCurrent(test.requestSession)

			if client.session != test.wantSession {
				t.Fatalf("session %q, want %q", client.session, test.wantSession)
			}
		})
	}
}

//nolint:wsl,wsl_v5 // Keep the response-ordering race scenario together.
func TestClientDelayedSuccessDoesNotOverwriteRecoveredSession(t *testing.T) {
	t.Parallel()

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var toolRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Header.Get("Mcp-Method") {
		case testToolCallMethod:
			if toolRequests.Add(1) == 1 {
				close(firstStarted)
				<-releaseFirst
				writer.Header().Set("Mcp-Session-Id", oldTestSession)
				_, _ = writer.Write([]byte(`{"result":{}}`))

				return
			}

			writer.WriteHeader(http.StatusNotFound)
		case testDiscoverMethod:
			writer.Header().Set("Mcp-Session-Id", newTestSession)
			_, _ = writer.Write([]byte(`{"result":{}}`))
		}
	}))
	t.Cleanup(server.Close)

	client := newClient(server.URL)
	client.session = oldTestSession
	firstDone := make(chan error, 1)

	go func() {
		_, err := client.CallTool(context.Background(), replyToolName, nil)
		firstDone <- err
	}()
	<-firstStarted
	_, err := client.CallTool(context.Background(), replyToolName, nil)
	if !errors.Is(err, errHTTPStatus) {
		t.Fatalf("failed call error %v", err)
	}
	err = client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if client.session != newTestSession {
		t.Fatalf("recovered session %q, want %q", client.session, newTestSession)
	}
	close(releaseFirst)
	err = <-firstDone
	if err != nil {
		t.Fatal(err)
	}
	if client.session != newTestSession {
		t.Fatalf("delayed response overwrote recovered session: %q", client.session)
	}
	if got := toolRequests.Load(); got != 2 {
		t.Fatalf("tool calls %d, want failed call not replayed (2 explicit requests)", got)
	}
}

func TestDiscoveryAndSubscriptionRequestsHaveDeadlines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call func(*Client) error
	}{
		{name: "discovery", call: func(client *Client) error { return client.Discover(context.Background()) }},
		{name: "subscription", call: func(client *Client) error {
			_, err := client.Subscribe(context.Background(), "callback", testSecret, nil)

			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				<-time.After(100 * time.Millisecond)

				_, _ = writer.Write([]byte(`{"result":{}}`))
			}))
			t.Cleanup(server.Close)

			client := newClient(server.URL)
			client.Timeout = 10 * time.Millisecond
			err := test.call(client)

			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("request error %v, want deadline", err)
			}
		})
	}
}

//nolint:wsl,wsl_v5 // Verify the deferred cleanup with one canceled lifecycle context.
func TestRunUnsubscribesOnceWhenAlreadyCanceled(t *testing.T) {
	t.Parallel()

	var unsubscribeCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Mcp-Method") == testUnsubscribe {
			unsubscribeCount.Add(1)
		}
		_, _ = writer.Write([]byte(`{"result":{}}`))
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	adapter := newAdapter(newClient(server.URL), "callback", testSecret, nil, nil)
	err := adapter.Run(ctx, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := unsubscribeCount.Load(); got != 1 {
		t.Fatalf("unsubscribe count %d, want 1", got)
	}
}

func TestClientRequestAndSubscription(t *testing.T) {
	t.Parallel()

	var (
		methods   []string
		methodsMu sync.Mutex
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method := request.Header.Get("Mcp-Method")

		methodsMu.Lock()

		methods = append(methods, method)
		methodsMu.Unlock()

		var req struct {
			ID uint64 `json:"id"`
		}

		err := json.NewDecoder(request.Body).Decode(&req)
		if err != nil {
			t.Error(err)
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Mcp-Session-Id", "session1")

		result := any(map[string]any{"ok": true})
		if method == subscribeMethod {
			result = map[string]any{"id": testSubscription, "refreshBefore": time.Now().Add(time.Hour)}
		}

		encodeErr := json.NewEncoder(writer).Encode(map[string]any{
			jsonRPCKey:    jsonRPCVersion,
			"id":          req.ID,
			testResultKey: result,
		})
		if encodeErr != nil {
			t.Error(encodeErr)
		}
	}))
	t.Cleanup(server.Close)

	client := newClient(server.URL)
	client.HTTP = server.Client()
	client.Token = "fake-token"

	err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.CallTool(context.Background(), replyToolName, map[string]any{"message_id": "m1", "content": "hello"})
	if err != nil {
		t.Fatal(err)
	}

	args := map[string]any{testMentionArg: true}

	sub, err := client.Subscribe(context.Background(), "https://callback.example/events", testSecret, args)

	if err != nil || sub.ID != testSubscription {
		t.Fatalf("subscription %#v, %v", sub, err)
	}

	want := []string{testDiscoverMethod, "tools/call", subscribeMethod}

	methodsMu.Lock()
	defer methodsMu.Unlock()

	if strings.Join(methods, ",") != strings.Join(want, ",") {
		t.Fatalf("methods %v", methods)
	}
}

//nolint:cyclop // The test keeps gateway registration, allowed calls, and rejected arguments together.
func TestGatewayCapabilities(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var call struct {
			Params struct {
				Name      string            `json:"name"`
				Arguments map[string]string `json:"arguments"`
			} `json:"params"`
		}

		err := json.NewDecoder(request.Body).Decode(&call)
		if err != nil {
			t.Error(err)
		}

		encodeErr := json.NewEncoder(writer).Encode(map[string]any{
			jsonRPCKey: jsonRPCVersion, "id": 1,
			"result": map[string]any{"tool": call.Params.Name, "args": call.Params.Arguments},
		})
		if encodeErr != nil {
			t.Error(encodeErr)
		}
	}))
	t.Cleanup(server.Close)

	adapter := newAdapter(newClient(server.URL), "", "", nil, nil)

	gateway, err := action.New(adapter.Tools(), 1)
	if err != nil {
		t.Fatal(err)
	}

	allowedDestinations := []string{"discord:channel-1", "discord:message-1"}
	session := gateway.Open(false, action.Scope{
		AllowedDestinations: allowedDestinations,
		ReadyRecipients:     allowedDestinations,
	})

	tests := []struct{ name, tool, args string }{
		{name: "reply", tool: "discord_reply", args: `{"message_id":"message-1","content":"thanks"}`},
		{name: "new post", tool: "discord_send_message", args: `{"channel_id":"channel-1","content":"done"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			encoded := []byte(test.args)

			pending, err := session.Plan(context.Background(), test.tool, encoded)
			if err != nil {
				t.Fatal(err)
			}

			observed, err := pending.Execute(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			if observed.Status != action.Succeeded {
				t.Fatalf("status %s", observed.Status)
			}
		})
	}

	if _, ok := adapter.Tools()["discord_voice_join"]; ok {
		t.Fatal("unselected voice tool registered")
	}

	for _, test := range []struct{ name, args string }{
		{name: "unknown field", args: `{"channel_id":"channel-1","content":"hi","guild_id":"guild-1"}`},
		{name: "missing destination", args: `{"content":"hi"}`},
		{name: "missing content", args: `{"channel_id":"channel-1"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := adapter.Tools()[sendToolName]([]byte(test.args))
			if err == nil {
				t.Fatal("invalid arguments accepted")
			}
		})
	}
}

func TestRunRefreshesAndCancellationUnsubscribes(t *testing.T) {
	t.Parallel()

	methods := make(chan string, 8)
	subscriptionReady := make(chan struct{}, 2)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method := request.Header.Get("Mcp-Method")
		methods <- method

		var req struct {
			ID uint64 `json:"id"`
		}

		err := json.NewDecoder(request.Body).Decode(&req)
		if err != nil {
			t.Error(err)
		}

		result := any(map[string]any{})
		if method == subscribeMethod {
			result = map[string]any{"id": testSubscription, "refreshBefore": time.Now().Add(2 * time.Second)}

			subscriptionReady <- struct{}{}
		}

		_ = json.NewEncoder(writer).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	adapter := newAdapter(newClient(server.URL), "https://callback.example/events", testSecret, nil, nil)

	done := make(chan error, 1)
	go func() { done <- adapter.Run(ctx, map[string]any{testMentionArg: true}) }()

	<-subscriptionReady
	<-subscriptionReady
	cancel()

	err := <-done
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := []string{<-methods, <-methods, <-methods, <-methods, <-methods}

	want := []string{testDiscoverMethod, "events/subscribe", testDiscoverMethod, "events/subscribe", "events/unsubscribe"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("methods %v", got)
	}
}

//nolint:wsl,wsl_v5 // Keep the signed callback setup adjacent to each table case.
func TestServeHTTPRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	now := time.Now()
	validEvent := `{"eventId":"e1","name":"discord.message.created","data":{}}`

	tests := []struct {
		name       string
		method     string
		body       string
		badSig     bool
		oversized  bool
		wantStatus int
	}{
		{name: "method", method: http.MethodGet, body: validEvent,
			badSig: false, oversized: false, wantStatus: http.StatusMethodNotAllowed},
		{name: "signature", method: http.MethodPost, body: validEvent,
			badSig: true, oversized: false, wantStatus: http.StatusUnauthorized},
		{name: "malformed JSON", method: http.MethodPost, body: "{",
			badSig: false, oversized: false, wantStatus: http.StatusBadRequest},
		{name: "no event handler", method: http.MethodPost, body: validEvent,
			badSig: false, oversized: false, wantStatus: http.StatusBadRequest},
		{name: "oversized body", method: http.MethodPost, body: "",
			badSig: false, oversized: true, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			body := test.body
			if test.oversized {
				body = strings.Repeat("x", maxCallbackBytes+1)
			}
			adapter := newAdapter(nil, "", testSecret, nil, func() time.Time { return now })
			req := httptest.NewRequestWithContext(context.Background(), test.method, "/events", strings.NewReader(body))
			timestamp := strconv.FormatInt(now.Unix(), 10)
			req.Header.Set("Webhook-Id", "w1")
			req.Header.Set("Webhook-Timestamp", timestamp)
			signature := "invalid"
			if !test.badSig {
				mac := hmac.New(sha256.New, []byte(testSecret))
				_, _ = mac.Write([]byte("w1." + timestamp + "." + body))
				signature = "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
			}
			req.Header.Set("Webhook-Signature", signature)

			rec := httptest.NewRecorder()
			adapter.ServeHTTP(rec, req)
			if rec.Code != test.wantStatus {
				t.Fatalf("status %d, want %d", rec.Code, test.wantStatus)
			}
		})
	}
}

func TestRunRejectsIncompleteConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		adapter *Adapter
	}{
		{name: "missing client", adapter: newAdapter(nil, "callback", testSecret, nil, nil)},
		{name: "missing callback", adapter: newAdapter(newClient("endpoint"), "", testSecret, nil, nil)},
		{name: "missing secret", adapter: newAdapter(newClient("endpoint"), "callback", "", nil, nil)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.adapter.Run(context.Background(), nil)
			if !errors.Is(err, errConfiguration) {
				t.Fatalf("error %v", err)
			}
		})
	}
}

func TestClientRequestFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint string
		params   any
	}{
		{name: "invalid endpoint URL", endpoint: "://invalid", params: map[string]string{}},
		{name: "unmarshalable parameters", endpoint: "http://localhost", params: func() {}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := newRequest(context.Background(), newClient(test.endpoint), "server/discover", test.params, "")
			if err == nil {
				t.Fatal("expected request construction error")
			}
		})
	}
}

func TestSubscribeRejectsMalformedResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result string
	}{
		{name: "invalid JSON", result: "not-json"},
		{name: "invalid subscription fields", result: `{"id":true}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(`{"result":` + test.result + `}`))
			}))
			t.Cleanup(server.Close)

			_, err := newClient(server.URL).Subscribe(context.Background(), "callback", testSecret, nil)
			if err == nil {
				t.Fatal("expected malformed result error")
			}
		})
	}
}

func TestPrepareRejectsMalformedAndWronglyTypedArguments(t *testing.T) {
	t.Parallel()

	prepareTool := newAdapter(nil, "", "", nil, nil).Tools()[sendToolName]

	tests := []struct {
		name string
		args string
	}{
		{name: "malformed JSON", args: `{"channel_id":`},
		{name: "destination is not text", args: `{"channel_id":1,"content":"hello"}`},
		{name: "content is not text", args: `{"channel_id":"c1","content":1}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := prepareTool([]byte(test.args))
			if err == nil {
				t.Fatal("invalid arguments accepted")
			}
		})
	}
}

func TestVerifyWebhookRejectsMissingIdentityAndAcceptsAnyMatchingSignature(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_800_000_000, 0)
	body := []byte(`{"eventId":"e1"}`)
	timestamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(testSecret))
	_, _ = mac.Write([]byte("id." + timestamp + "." + string(body)))
	validSignature := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		secret    string
		id        string
		timestamp string
		signature string
		want      bool
	}{
		{name: "empty secret", secret: "", id: "id", timestamp: timestamp, signature: validSignature, want: false},
		{name: "empty webhook ID", secret: testSecret, id: "", timestamp: timestamp, signature: validSignature, want: false},
		{name: "future timestamp", secret: testSecret, id: "id",
			timestamp: strconv.FormatInt(now.Add(6*time.Minute).Unix(), 10),
			signature: validSignature, want: false},
		{name: "later matching signature", secret: testSecret, id: "id",
			timestamp: timestamp, signature: "v1,invalid " + validSignature, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := VerifyWebhook(test.secret, test.id, test.timestamp, test.signature, body, now)
			if got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

//nolint:wsl,wsl_v5,gocognit,cyclop // Keep each fake-server retry scenario in one focused table-driven test.
func TestRunRetriesTransientConnectionFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		failMethod string
	}{
		{name: "discovery", failMethod: "server/discover"},
		{name: "subscription", failMethod: subscribeMethod},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			methods := make(chan string, 8)
			subscribed := make(chan struct{}, 1)
			attempts := map[string]int{}
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				method := request.Header.Get("Mcp-Method")
				methods <- method
				attempts[method]++
				if method == test.failMethod && attempts[method] == 1 {
					writer.WriteHeader(http.StatusServiceUnavailable)

					return
				}
				if method == subscribeMethod {
					subscribed <- struct{}{}
					_, _ = writer.Write([]byte(`{"result":{"id":"sub1","refreshBefore":"2099-01-01T00:00:00Z"}}`))

					return
				}
				_, _ = writer.Write([]byte(`{"result":{}}`))
			}))
			t.Cleanup(server.Close)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			adapter := newAdapter(newClient(server.URL), "https://callback.example/events", testSecret, nil, nil)
			done := make(chan error, 1)

			go func() { done <- adapter.Run(ctx, map[string]any{testMentionArg: true}) }()

			select {
			case <-subscribed:
			case <-time.After(5 * time.Second):
				t.Fatal("adapter did not reconnect and subscribe")
			}

			cancel()

			err := <-done
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got := <-methods; got != "server/discover" {
				t.Fatalf("first MCP method %q", got)
			}

			seenUnsubscribe := false
			for range 7 {
				select {
				case method := <-methods:
					seenUnsubscribe = seenUnsubscribe || method == "events/unsubscribe"
				case <-time.After(100 * time.Millisecond):
					if seenUnsubscribe {
						return
					}
					t.Fatal("missing unsubscribe after cancellation")
				}
			}
		})
	}
}
