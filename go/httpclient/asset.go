package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset"
)

type AssetClient struct{ caller jsonRPCCaller }

func newAssetClient(caller jsonRPCCaller) (*AssetClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create asset client: caller=null")
	}
	return &AssetClient{caller: caller}, nil
}

// Get retrieves registered asset metadata in one request, preserving caller parameters.
// It validates response scope and does not retry automatically.
//
// Version:
//   - 2026-10-21: Added.
func (c *AssetClient) Get(ctx context.Context, params asset.Params) (*asset.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to get asset: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get asset: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode asset parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubAssetGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}
	var result *asset.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode asset response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to get asset: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}
	if result.AssetType != params.AssetType || result.Asset != params.Asset {
		return nil, fmt.Errorf("failed to get asset: response_scope=invalid")
	}
	return result, nil
}
