package websocket_test

import (
	"context"
	"errors"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/websocket"
)

// ExampleModule_MarketHubScalping shows composition and full-result replacement with application credentials.
//
// Version:
//   - 2026-09-26: Added.
func ExampleModule_MarketHubScalping() {
	observe := func(ctx context.Context, credentials authentication.CredentialProvider) (scalping.Result, error) {
		module, err := websocket.NewModule(ctx, websocket.ModuleConfig{EndpointURL: "wss://api.k4k3ru.com/", CredentialProvider: credentials})
		if err != nil {
			return scalping.Result{}, err
		}
		read := func() (scalping.Result, error) {
			sub, err := module.MarketHubScalping().Subscribe(ctx, scalping.Params{MarketType: market.MarketTypeSpot, Symbol: market.SUIUSDC, WindowMS: scalping.DefaultWindowMS, Markets: []market.MarketTarget{{Venue: market.Cetus, Chain: "sui", Network: "testnet"}}})
			if err != nil {
				return scalping.Result{}, err
			}
			select {
			case snapshot, ok := <-sub.Events():
				if !ok {
					return scalping.Result{}, <-sub.Errors()
				}
				if err := module.MarketHubScalping().Unsubscribe(ctx, sub); err != nil {
					return scalping.Result{}, err
				}
				return snapshot, nil
			case err := <-sub.Errors():
				return scalping.Result{}, err
			case <-ctx.Done():
				return scalping.Result{}, ctx.Err()
			}
		}
		snapshot, err := read()
		return snapshot, errors.Join(err, module.Close())
	}
	_ = observe // Applications supply their own context and CredentialProvider.
}
