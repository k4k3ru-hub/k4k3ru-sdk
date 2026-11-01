package httpclient_test

import (
	"context"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset/overview"
)

// ExampleAssetOverviewClient_Get shows owning-package DTOs and the composed asset operation.
//
// Version:
//   - 2026-11-01: Added.
func ExampleAssetOverviewClient_Get() {
	client, err := httpclient.NewClient(httpclient.Config{EndpointURL: "https://api.example.test/", HTTPClient: &http.Client{Timeout: 10 * time.Second}})
	if err != nil {
		return
	}
	result, err := client.MarketHub().Asset().Overview().Get(context.Background(), overview.Params{AssetType: "crypto", Asset: "sui"})
	if err != nil {
		return
	}
	_ = result
}
