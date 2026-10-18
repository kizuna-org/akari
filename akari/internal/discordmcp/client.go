// Package discordmcp connects Akari to the selected Discord MCP server.
// It deliberately exposes only message reply and send operations.
package discordmcp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	protocolVersion     = "2026-07-28"
	jsonRPCVersion      = "2.0"
	discoverMethod      = "server/discover"
	subscribeMethod     = "events/subscribe"
	unsubscribeMethod   = "events/unsubscribe"
	jsonRPCKey          = "jsonrpc"
	metaKey             = "_meta"
	argumentsKey        = "arguments"
	nameKey             = "name"
	protocolMetaKey     = "io.modelcontextprotocol/protocolVersion"
	secretKey           = "secret"
	messageEventName    = "discord.message.created"
	replyToolName       = "discord_reply"
	sendToolName        = "discord_send_message"
	mcpMaxResponseBytes = 4 << 20
	defaultEventTTL     = time.Hour
	requestTimeout      = 10 * time.Second
)

var (
	errEndpointRequired = errors.New("discordmcp: endpoint is required")
	errToolDisabled     = errors.New("discordmcp: tool is not enabled")
	errHTTPStatus       = errors.New("discordmcp: non-success HTTP status")
	errRPC              = errors.New("discordmcp: RPC error")
	errSubscription     = errors.New("discordmcp: subscription response is missing id or refreshBefore")
	errToolResult       = errors.New("discordmcp: tool result is missing")
)

const jsonNull = "null"

type Client struct {
	Endpoint string
	Token    string
	HTTP     *http.Client
	Timeout  time.Duration
	mu       sync.Mutex
	session  string
	next     atomic.Uint64
}

type resultEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type toolResult struct {
	IsError bool `json:"isError"`
}

type requestSpec struct {
	method string
	params any
	tool   string
}

func call(ctx context.Context, client *Client, spec requestSpec) (json.RawMessage, error) {
	request, err := newRequest(ctx, client, spec)
	if err != nil {
		return nil, err
	}

	httpClient := client.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusNotFound {
		client.clearSessionIfCurrent(request.Header.Get("Mcp-Session-Id"))
	}

	return decode(client, response, request.Header.Get("Mcp-Session-Id"), spec.tool != "")
}

func newRequest(ctx context.Context, client *Client, spec requestSpec) (*http.Request, error) {
	if client.Endpoint == "" {
		return nil, errEndpointRequired
	}

	id := client.next.Add(1)

	body, err := json.Marshal(map[string]any{jsonRPCKey: jsonRPCVersion, "id": id,
		"method": spec.method, "params": spec.params})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Mcp-Method", spec.method)
	req.Header.Set("Mcp-Protocol-Version", protocolVersion)

	if spec.tool != "" {
		req.Header.Set("Mcp-Name", spec.tool)
	}

	if client.Token != "" {
		req.Header.Set("Authorization", "Bearer "+client.Token)
	}

	client.mu.Lock()
	session := client.session
	client.mu.Unlock()

	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
	}

	return req, nil
}

//nolint:cyclop,wsl,wsl_v5 // Response checks intentionally follow protocol validation order.
func decode(client *Client, response *http.Response, requestSession string, isToolCall bool) (json.RawMessage, error) {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: %d", errHTTPStatus, response.StatusCode)
	}

	if value := response.Header.Get("Mcp-Session-Id"); value != "" {
		client.mu.Lock()
		if client.session == requestSession {
			client.session = value
		}
		client.mu.Unlock()
	}

	var envelope resultEnvelope

	err := json.NewDecoder(io.LimitReader(response.Body, mcpMaxResponseBytes)).Decode(&envelope)
	if err != nil {
		return nil, err
	}

	if envelope.Error != nil {
		return nil, fmt.Errorf("%w %d: %s", errRPC, envelope.Error.Code, envelope.Error.Message)
	}

	if isToolCall {
		if len(envelope.Result) == 0 || string(envelope.Result) == jsonNull {
			return envelope.Result, errToolResult
		}
		var result toolResult

		err = json.Unmarshal(envelope.Result, &result)
		if err != nil {
			return envelope.Result, err
		}

		if result.IsError {
			return envelope.Result, fmt.Errorf("%w: MCP tool returned isError", errRPC)
		}
	}

	return envelope.Result, nil
}

func (c *Client) Discover(ctx context.Context) error {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	params := map[string]any{metaKey: map[string]any{protocolMetaKey: protocolVersion}}
	_, err := call(requestCtx, c, requestSpec{method: discoverMethod, params: params, tool: ""})

	return err
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (json.RawMessage, error) {
	if name != replyToolName && name != sendToolName {
		return nil, errToolDisabled
	}

	requestCtx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	params := map[string]any{
		nameKey:      name,
		argumentsKey: args,
		metaKey:      map[string]any{protocolMetaKey: protocolVersion},
	}

	return call(requestCtx, c, requestSpec{method: "tools/call", params: params, tool: name})
}

type Event struct {
	ID      string          `json:"eventId"`
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"data"`
}

type subscription struct {
	ID            string    `json:"id"`
	RefreshBefore time.Time `json:"refreshBefore"`
}

func (c *Client) Subscribe(ctx context.Context, callbackURL, secret string, args map[string]any) (subscription, error) {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	var sub subscription

	params := map[string]any{
		nameKey:      messageEventName,
		argumentsKey: args,
		"delivery":   map[string]string{"mode": "webhook", "url": callbackURL, secretKey: secret},
		"ttlMs":      defaultEventTTL.Milliseconds(),
		metaKey:      map[string]any{protocolMetaKey: protocolVersion},
	}

	raw, err := call(requestCtx, c, requestSpec{method: subscribeMethod, params: params, tool: ""})
	if err == nil {
		err = json.Unmarshal(raw, &sub)
	}

	if err == nil && (sub.ID == "" || sub.RefreshBefore.IsZero()) {
		err = errSubscription
	}

	return sub, err
}

func (c *Client) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}

	return requestTimeout
}

func (c *Client) clearSessionIfCurrent(requestSession string) {
	if requestSession == "" {
		return
	}

	c.mu.Lock()
	if c.session == requestSession {
		c.session = ""
	}
	c.mu.Unlock()
}

type WebhookSignature struct {
	Secret    string
	WebhookID string
	Timestamp string
	Signature string
	Body      []byte
	Now       time.Time
}

// VerifyWebhook validates the Discord MCP v1 signature over id.timestamp.body.
func VerifyWebhook(input WebhookSignature) bool {
	if input.Secret == "" || input.WebhookID == "" {
		return false
	}

	seconds, err := strconv.ParseInt(input.Timestamp, 10, 64)
	if err != nil || seconds < input.Now.Unix()-300 || seconds > input.Now.Unix()+300 {
		return false
	}

	mac := hmac.New(sha256.New, []byte(input.Secret))
	_, _ = mac.Write([]byte(input.WebhookID + "." + input.Timestamp + "." + string(input.Body)))

	expected := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	for candidate := range strings.FieldsSeq(input.Signature) {
		if hmac.Equal([]byte(candidate), []byte(expected)) {
			return true
		}
	}

	return false
}
