package websocket_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/websocket"
)

// ExampleScalpingClient_Run shows authenticated composition and a full snapshot read without submitting orders.
//
// Version:
//   - 2026-10-01: Added.
func ExampleScalpingClient_Run() {
	observe := func(ctx context.Context, credentials authentication.CredentialProvider, params scalping.RunParams) (snapshot scalping.RunSnapshot, err error) {
		module, err := websocket.NewModule(ctx, websocket.ModuleConfig{EndpointURL: "wss://api.k4k3ru.com/", CredentialProvider: credentials})
		if err != nil {
			return snapshot, err
		}
		defer func() { err = errors.Join(err, module.Close()) }()
		stream, err := module.Scalping().Run(ctx, params)
		if err != nil {
			return snapshot, err
		}
		// Persist stream.Reference().ExecutionID before acting on candidates.
		// Resume with RunParams{ExecutionID: savedID}; restore state from the next snapshot.
		select {
		case e, open := <-stream.Events():
			if !open {
				return snapshot, fmt.Errorf("failed to observe scalping run: stream=closed")
			}
			if e.Error != nil {
				return snapshot, fmt.Errorf("failed to observe scalping run: code=%q", e.Error.Code)
			}
			if e.Snapshot == nil {
				return snapshot, fmt.Errorf("failed to observe scalping run: snapshot=null")
			}
			snapshot = *e.Snapshot // Replace the entire view, including orders: [].
			return snapshot, module.Scalping().UnsubscribeRun(ctx, stream)
		case interruption, open := <-stream.Errors():
			if !open {
				return snapshot, fmt.Errorf("failed to observe scalping run: stream=closed")
			}
			return snapshot, interruption
		case <-ctx.Done():
			return snapshot, ctx.Err()
		}
	}
	_ = observe // Applications supply credentials and a new or execution-ID-only request.
}
