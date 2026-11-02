package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/assetnews"
)

type AssetNewsClient struct{ caller jsonRPCCaller }

func newAssetNewsClient(caller jsonRPCCaller) (*AssetNewsClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create asset news client: caller=null")
	}
	return &AssetNewsClient{caller: caller}, nil
}

// List reads one page of retained official Asset context without automatic retries or pagination.
//
// Version:
//   - 2026-11-02: Added.
func (c *AssetNewsClient) List(ctx context.Context, params assetnews.Params) (*assetnews.Result, error) {
	if c == nil || c.caller == nil || ctx == nil {
		return nil, fmt.Errorf("failed to list asset news: dependency=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to list asset news: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to list asset news: %w", err)
	}
	encoded, err := json.Marshal(params.Normalize())
	if err != nil {
		return nil, fmt.Errorf("failed to encode asset news request: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubAssetNewsList, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to list asset news: %w", err)
	}
	var result assetnews.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode asset news result: %w", err)
	}
	if err := result.ValidateFor(params); err != nil {
		return nil, fmt.Errorf("failed to list asset news: %w", err)
	}
	return &result, nil
}

// AssetNews returns the composed official Asset context operation client.
//
// Version:
//   - 2026-11-02: Added.
func (c *MarketHubClient) AssetNews() *AssetNewsClient {
	if c == nil {
		return nil
	}
	return c.assetNews
}
