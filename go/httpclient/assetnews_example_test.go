package httpclient_test

import (
	"context"
	"net/http"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/httpclient"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/assetnews"
)

// ExampleAssetNewsClient_List composes official context reads with owning-package parameters.
//
// Version:
//   - 2026-11-02: Added.
func ExampleAssetNewsClient_List() {
	client, err := httpclient.NewClient(httpclient.Config{EndpointURL: "https://api.example.test/", HTTPClient: &http.Client{Timeout: 10 * time.Second}})
	if err != nil {
		return
	}
	result, err := client.MarketHub().AssetNews().List(context.Background(), assetnews.Params{AssetType: "crypto", Asset: "sui"})
	if err != nil {
		return
	}
	_ = result // Source claims and unknown network labels require separate review.
}
