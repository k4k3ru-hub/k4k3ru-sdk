package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset/overview"
)

type AssetOverviewClient struct{ caller jsonRPCCaller }

func newAssetOverviewClient(caller jsonRPCCaller) (*AssetOverviewClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create asset overview client: caller=null")
	}
	return &AssetOverviewClient{caller: caller}, nil
}

// Overview returns the composed retained asset overview operation.
//
// Version:
//   - 2026-11-01: Added.
func (c *AssetClient) Overview() *AssetOverviewClient {
	if c == nil {
		return nil
	}
	return c.overview
}

// Get reads one retained asset snapshot without retrying or modifying request parameters.
//
// Version:
//   - 2026-11-01: Added.
func (c *AssetOverviewClient) Get(ctx context.Context, p overview.Params) (*overview.Result, error) {
	if c == nil || c.caller == nil || ctx == nil {
		return nil, fmt.Errorf("failed to get asset overview: dependency=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get asset overview: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get asset overview: %w", err)
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to encode asset overview parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubAssetOverviewGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset overview: %w", err)
	}
	r, err := overview.DecodeResult(body)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset overview: %w", err)
	}
	if err := r.ValidateFor(p); err != nil {
		return nil, fmt.Errorf("failed to get asset overview: %w", err)
	}
	return r, nil
}
