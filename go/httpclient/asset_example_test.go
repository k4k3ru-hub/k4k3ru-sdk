package httpclient_test

import (
	"context"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset"
)

// ExampleAssetClient_Get shows owning-package DTOs and explicit HTTP dependency composition.
//
// Version:
//   - 2026-10-21: Added.
func ExampleAssetClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.example.test/",
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		return
	}
	result, err := client.MarketHub().Asset().Get(context.Background(), asset.Params{AssetType: "crypto", Asset: "sui"})
	if err != nil {
		return
	}
	_ = result
}

// ExampleAssetClient_List shows asset discovery with optional type and search filters.
//
// Version:
//   - 2026-10-21: Added.
func ExampleAssetClient_List() {
	client, err := httpclient.NewClient(httpclient.Config{
		EndpointURL: "https://api.example.test/",
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		return
	}
	result, err := client.MarketHub().Asset().List(context.Background(), asset.ListParams{
		AssetType: "crypto", MarketType: "spot", Search: "sui", Limit: 25,
	})
	if err != nil {
		return
	}
	_ = result
}
