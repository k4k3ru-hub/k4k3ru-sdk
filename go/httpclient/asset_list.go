package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset"
)

// List retrieves one registered asset page without retries or automatic pagination.
// It preserves caller parameters and validates returned metadata and query scope.
//
// Version:
//   - 2026-10-21: Added.
func (c *AssetClient) List(ctx context.Context, params asset.ListParams) (*asset.ListResult, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to list assets: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to list assets: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}
	params = params.Normalize()
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode asset list parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubAssetList, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}
	var result *asset.ListResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode asset list response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to list assets: result=null")
	}
	if err := result.ValidateFor(params); err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}
	return result, nil
}
