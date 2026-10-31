package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/news"
)

type NewsClient struct{ caller jsonRPCCaller }

func newNewsClient(caller jsonRPCCaller) (*NewsClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create news client: caller=null")
	}
	return &NewsClient{caller: caller}, nil
}

// List reads one page of retained Mainnet Asset context without automatic retries or pagination.
//
// Version:
//   - 2026-10-31: Added.
func (c *NewsClient) List(ctx context.Context, params news.Params) (*news.Result, error) {
	if c == nil || c.caller == nil || ctx == nil {
		return nil, fmt.Errorf("failed to list news: dependency=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to list news: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to list news: %w", err)
	}
	encoded, err := json.Marshal(params.Normalize())
	if err != nil {
		return nil, fmt.Errorf("failed to encode news request: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubNewsList, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to list news: %w", err)
	}
	var result news.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode news result: %w", err)
	}
	if err := result.ValidateFor(params); err != nil {
		return nil, fmt.Errorf("failed to list news: %w", err)
	}
	return &result, nil
}

// News returns the composed Asset news operation client.
//
// Version:
//   - 2026-10-31: Added.
func (c *MarketHubClient) News() *NewsClient {
	if c == nil {
		return nil
	}
	return c.news
}
