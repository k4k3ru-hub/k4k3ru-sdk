package websocket

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

var errWebSocketConnectionClosed = errors.New("websocket connection closed")

type requestOutcome struct {
	response k4k3ruSDKJSONRPC.Response
	err      error
}

type pendingRequest struct {
	key     string
	outcome chan requestOutcome
}

type requestTracker struct {
	mu      sync.Mutex
	pending map[string]*pendingRequest
}

func newRequestTracker() *requestTracker {
	return &requestTracker{pending: make(map[string]*pendingRequest)}
}

func (t *requestTracker) register(id json.RawMessage) (*pendingRequest, error) {
	if t == nil {
		return nil, fmt.Errorf("failed to register websocket request: request_tracker=null")
	}
	key, err := requestIDKey(id)
	if err != nil {
		return nil, fmt.Errorf("failed to register websocket request: %w", err)
	}
	request := &pendingRequest{key: key, outcome: make(chan requestOutcome, 1)}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.pending[key]; exists {
		return nil, fmt.Errorf("failed to register websocket request: request_id=duplicate")
	}
	t.pending[key] = request
	return request, nil
}

func (t *requestTracker) cancel(request *pendingRequest) {
	if t == nil || request == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if current, exists := t.pending[request.key]; exists && current == request {
		delete(t.pending, request.key)
	}
}

func (t *requestTracker) resolve(response k4k3ruSDKJSONRPC.Response) (bool, error) {
	if t == nil {
		return false, fmt.Errorf("failed to resolve websocket request: request_tracker=null")
	}
	key, err := requestIDKey(response.ID)
	if err != nil {
		return false, fmt.Errorf("failed to resolve websocket request: %w", err)
	}
	t.mu.Lock()
	request, exists := t.pending[key]
	if exists {
		delete(t.pending, key)
	}
	t.mu.Unlock()
	if !exists {
		return false, nil
	}
	request.outcome <- requestOutcome{response: response}
	close(request.outcome)
	return true, nil
}

func (t *requestTracker) failAll(err error) {
	if t == nil {
		return
	}
	if err == nil {
		err = errWebSocketConnectionClosed
	}
	t.mu.Lock()
	pending := t.pending
	t.pending = make(map[string]*pendingRequest)
	t.mu.Unlock()
	for _, request := range pending {
		request.outcome <- requestOutcome{err: err}
		close(request.outcome)
	}
}

func requestIDKey(id json.RawMessage) (string, error) {
	if len(bytes.TrimSpace(id)) == 0 {
		return "", fmt.Errorf("failed to validate websocket request id: request_id=empty")
	}
	decoder := json.NewDecoder(bytes.NewReader(id))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", fmt.Errorf("failed to validate websocket request id: failed to decode json: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return "", fmt.Errorf("failed to validate websocket request id: failed to decode json: %w", err)
	}
	switch value.(type) {
	case string, json.Number:
	default:
		return "", fmt.Errorf("failed to validate websocket request id: request_id=invalid")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, id); err != nil {
		return "", fmt.Errorf("failed to validate websocket request id: failed to compact json: %w", err)
	}
	return compact.String(), nil
}
