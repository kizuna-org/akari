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
)

var (
	errEndpointRequired = errors.New("discord MCP endpoint is required")
	errToolDisabled     = errors.New("discord tool is not enabled")
	errHTTPStatus       = errors.New("discord MCP returned a non-success HTTP status")
	errRPC              = errors.New("discord MCP returned an RPC error")
)

type Client struct {
	Endpoint string
	Token    string
	HTTP     *http.Client
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

func call(ctx context.Context, client *Client, method string, params any, tool string) (json.RawMessage, error) {
	request, err := newRequest(ctx, client, method, params, tool)
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

	return decode(client, response)
}

func newRequest(ctx context.Context, client *Client, method string, params any, tool string) (*http.Request, error) {
	if client.Endpoint == "" {
		return nil, errEndpointRequired
	}

	id := client.next.Add(1)

	body, err := json.Marshal(map[string]any{jsonRPCKey: jsonRPCVersion, "id": id, "method": method, "params": params})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Mcp-Method", method)
	req.Header.Set("Mcp-Protocol-Version", protocolVersion)

	if tool != "" {
		req.Header.Set("Mcp-Name", tool)
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

func decode(client *Client, response *http.Response) (json.RawMessage, error) {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: %d", errHTTPStatus, response.StatusCode)
	}

	if value := response.Header.Get("Mcp-Session-Id"); value != "" {
		client.mu.Lock()
		client.session = value
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

	return envelope.Result, nil
}

func (c *Client) Discover(ctx context.Context) error {
	params := map[string]any{metaKey: map[string]any{protocolMetaKey: protocolVersion}}
	_, err := call(ctx, c, "server/discover", params, "")

	return err
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (json.RawMessage, error) {
	if name != replyToolName && name != sendToolName {
		return nil, errToolDisabled
	}

	params := map[string]any{
		nameKey:      name,
		argumentsKey: args,
		metaKey:      map[string]any{protocolMetaKey: protocolVersion},
	}

	return call(ctx, c, "tools/call", params, name)
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
	var sub subscription

	params := map[string]any{
		nameKey:      messageEventName,
		argumentsKey: args,
		"delivery":   map[string]string{"mode": "webhook", "url": callbackURL, secretKey: secret},
		"ttlMs":      defaultEventTTL.Milliseconds(),
		metaKey:      map[string]any{protocolMetaKey: protocolVersion},
	}

	raw, err := call(ctx, c, "events/subscribe", params, "")
	if err == nil {
		err = json.Unmarshal(raw, &sub)
	}

	return sub, err
}

// VerifyWebhook validates the Discord MCP v1 signature over id.timestamp.body.
func VerifyWebhook(secret, webhookID, timestamp, signature string, body []byte, now time.Time) bool {
	if secret == "" || webhookID == "" {
		return false
	}

	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || seconds < now.Unix()-300 || seconds > now.Unix()+300 {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(webhookID + "." + timestamp + "." + string(body)))

	expected := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	for candidate := range strings.FieldsSeq(signature) {
		if hmac.Equal([]byte(candidate), []byte(expected)) {
			return true
		}
	}

	return false
}
