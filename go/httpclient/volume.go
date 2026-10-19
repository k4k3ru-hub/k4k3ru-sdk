package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/volume"
)

type VolumeClient struct{ caller jsonRPCCaller }

func newVolumeClient(caller jsonRPCCaller) (*VolumeClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create volume client: caller=null")
	}
	return &VolumeClient{caller: caller}, nil
}

// Get retrieves selected Composite volumes in one HTTP request without retries.
// Omitted periods select all six supported periods; the caller's parameters are not mutated.
//
// Version:
//   - 2026-10-19: Added.
func (c *VolumeClient) Get(ctx context.Context, params volume.Params) (*volume.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to get volumes: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get volumes: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get volumes: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get volumes: %w", err)
	}
	params = params.Normalize()
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode volume parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubVolumeGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get volumes: %w", err)
	}
	var result *volume.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode volume response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to get volumes: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get volumes: %w", err)
	}
	if result.Asset != params.Asset || result.MarketType != params.MarketType {
		return nil, fmt.Errorf("failed to get volumes: selectors=invalid")
	}
	if len(result.Volumes) != len(params.Periods) {
		return nil, fmt.Errorf("failed to get volumes: periods=invalid")
	}
	for i, period := range params.Periods {
		if result.Volumes[i].Period != period {
			return nil, fmt.Errorf("failed to get volumes: period=invalid index=%d", i)
		}
	}
	return result, nil
}
