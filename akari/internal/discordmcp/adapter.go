package discordmcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/kizuna-org/akari/internal/action"
)

var (
	errArguments     = errors.New("discordmcp: expected a destination and content")
	errDestination   = errors.New("discordmcp: destination is required")
	errContent       = errors.New("discordmcp: content is required")
	errConfiguration = errors.New("discordmcp: adapter configuration is incomplete")
)

const operationArgumentCount = 2

const (
	challengeKey        = "challenge"
	challengeType       = "verification"
	maxCallbackBytes    = 1 << 20
	maxReconnectDelay   = 30 * time.Second
	reconnectMultiplier = 2
	unsubscribeTimeout  = 3 * time.Second
)

type Adapter struct {
	Client      *Client
	CallbackURL string
	Secret      string
	Handler     func(context.Context, Event) error
	Now         func() time.Time
}

// Tools returns the two explicitly selected speech capabilities. Other
// discovered Discord tools are intentionally never registered.
func (a *Adapter) Tools() map[string]action.Prepare {
	return map[string]action.Prepare{
		replyToolName: prepare(a, replyToolName, "message_id", "content"),
		sendToolName:  prepare(a, sendToolName, "channel_id", "content"),
	}
}

func prepare(adapter *Adapter, tool, destinationArg, contentArg string) action.Prepare {
	return func(raw []byte) (action.Operation, error) {
		var args map[string]any

		err := json.Unmarshal(raw, &args)
		if err != nil {
			return action.Operation{}, err
		}

		if len(args) != operationArgumentCount {
			return action.Operation{}, errArguments
		}

		destination, exists := args[destinationArg].(string)
		if !exists || destination == "" {
			return action.Operation{}, errDestination
		}

		content, contentExists := args[contentArg].(string)
		if !contentExists || content == "" {
			return action.Operation{}, errContent
		}

		return action.Operation{
			Impact:      action.Speech,
			Destination: "discord:" + destination,
			ConflictKey: "discord:" + destination,
			Call: func(ctx context.Context) ([]byte, error) {
				result, err := adapter.Client.CallTool(ctx, tool, map[string]any{destinationArg: destination, contentArg: content})

				return result, err
			}}, nil
	}
}

// ServeHTTP accepts only authenticated webhook verification and event callbacks.
func (a *Adapter) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	limited := http.MaxBytesReader(writer, request.Body, maxCallbackBytes)

	data, err := io.ReadAll(limited)
	if err != nil {
		http.Error(writer, "invalid body", http.StatusBadRequest)

		return
	}

	handleWebhook(a, writer, request, data)
}

func handleWebhook(adapter *Adapter, writer http.ResponseWriter, request *http.Request, data []byte) {
	now := adapter.Now
	if now == nil {
		now = time.Now
	}

	id := request.Header.Get("Webhook-Id")
	timestamp := request.Header.Get("Webhook-Timestamp")
	signature := request.Header.Get("Webhook-Signature")

	if !VerifyWebhook(WebhookSignature{Secret: adapter.Secret, WebhookID: id, Timestamp: timestamp,
		Signature: signature, Body: data, Now: now()}) {
		http.Error(writer, "invalid signature", http.StatusUnauthorized)

		return
	}

	var envelope struct {
		Event

		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}

	err := json.Unmarshal(data, &envelope)
	if err != nil {
		http.Error(writer, "invalid event", http.StatusBadRequest)

		return
	}

	writer.Header().Set("Content-Type", "application/json")

	if envelope.Type == challengeType {
		encodeErr := json.NewEncoder(writer).Encode(map[string]string{challengeKey: envelope.Challenge})
		if encodeErr != nil {
			return
		}

		return
	}

	if envelope.Name != messageEventName || adapter.Handler == nil {
		http.Error(writer, "unsupported event", http.StatusBadRequest)

		return
	}

	err = adapter.Handler(request.Context(), envelope.Event)
	if err != nil {
		http.Error(writer, "event handler failed", http.StatusServiceUnavailable)

		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

// Run subscribes, refreshes before expiry, and retries transient disconnects
// with bounded backoff. Cancellation stops retries and attempts unsubscribe.
func (a *Adapter) Run(ctx context.Context, args map[string]any) error {
	if a.Client == nil || a.CallbackURL == "" || a.Secret == "" {
		return errConfiguration
	}
	defer a.unsubscribe(ctx, args)

	delay := time.Second

	for ctx.Err() == nil {
		sub, err := a.connect(ctx, args)
		if err != nil {
			if !sleep(ctx, delay) {
				break
			}

			delay = min(delay*reconnectMultiplier, maxReconnectDelay)

			continue
		}

		delay = time.Second

		if !waitUntilRefresh(ctx, sub.RefreshBefore) {
			return nil
		}
	}

	return nil
}

func (a *Adapter) connect(ctx context.Context, args map[string]any) (subscription, error) {
	err := a.Client.Discover(ctx)
	if err != nil {
		return subscription{}, err
	}

	return a.Client.Subscribe(ctx, a.CallbackURL, a.Secret, args)
}

func waitUntilRefresh(ctx context.Context, refreshBefore time.Time) bool {
	delay := max(time.Until(refreshBefore.Add(-time.Minute)), time.Second)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (a *Adapter) unsubscribe(parent context.Context, args map[string]any) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), unsubscribeTimeout)
	defer cancel()

	params := map[string]any{
		nameKey:      messageEventName,
		argumentsKey: args,
		"delivery":   map[string]string{"mode": "webhook", "url": a.CallbackURL},
		metaKey:      map[string]any{protocolMetaKey: protocolVersion},
	}
	_, _ = call(ctx, a.Client, requestSpec{method: unsubscribeMethod, params: params, tool: ""})
}

func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
