package websocket_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/websocket"
)

type exampleExecutionCredentials struct{}

// Credential reads API credentials from the caller's environment.
//
// Version:
//   - 2026-09-16: Added.
func (exampleExecutionCredentials) Credential(context.Context) (authentication.Credential, error) {
	return authentication.Credential{APIKey: os.Getenv("K4K3RU_API_KEY"), SecretKey: os.Getenv("K4K3RU_SECRET_KEY"), SignatureAlgorithm: authentication.SignatureAlgorithmHMACSHA256}, nil
}

// ExampleModule_Execution demonstrates observing an already-submitted transaction.
// This example requires a caller-supplied execution ID and does not submit trades.
//
// Version:
//   - 2026-09-16: Added.
func ExampleModule_Execution() {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	status, err := observeExampleExecution(ctx, os.Getenv("K4K3RU_EXECUTION_ID"))
	if err != nil {
		fmt.Println("Observation interrupted:", err)
		return
	}
	fmt.Println("Receipt inclusion status:", status)
}

func observeExampleExecution(ctx context.Context, id string) (status execution.ObservationStatus, err error) {
	module, err := websocket.NewModule(ctx, websocket.ModuleConfig{EndpointURL: "wss://api.k4k3ru.com/", CredentialProvider: exampleExecutionCredentials{}})
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, module.Close()) }()
	subscription, err := module.Execution().Subscribe(ctx, execution.SubscribeParams{ExecutionID: id})
	if err != nil {
		return "", err
	}
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case e, ok := <-subscription.Events():
			if !ok {
				if interruption, ok := <-subscription.Errors(); ok {
					return "", interruption
				}
				return "", fmt.Errorf("failed to observe execution: watch ended without receipt")
			}
			if e.Error != nil {
				return "", fmt.Errorf("failed to observe execution: %s", e.Error.Code)
			}
			if e.Snapshot != nil && e.Snapshot.Status.Terminal() {
				return e.Snapshot.Status, nil
			}
		}
	}
}
