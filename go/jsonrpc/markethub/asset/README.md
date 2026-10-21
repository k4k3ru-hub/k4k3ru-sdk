# MarketHub.Asset.Get

Registered Mainnet asset metadata from MarketHub's in-memory Registry/Catalog.
No request-time Parquet, Venue RPC, external metadata or price history is needed.

For paginated search and discovery, see [Asset.List](LIST.md). Both operations
use the same Catalog identities and full registered market-type coverage.

## Request

```json
{"jsonrpc":"2.0","id":1,"method":"MarketHub.Asset.Get","params":{"assetType":"crypto","asset":"sui"}}
```

- Both fields are required; `assetType` accepts only `crypto`.
- `asset` is the existing Catalog slug, matching `^[a-z0-9][a-z0-9-]{0,63}$`.
  It is not a trading pair, contract address or a display-symbol lookup.
- Unknown, duplicate, missing or null fields and unsupported values are errors.
  In particular, `marketType`, `network` and `testMode` are not selectors here.
- Do not normalize slugs or merge tokens by symbol. Existing Catalog identity
  mappings and same-symbol token isolation remain authoritative.

## Result

```json
{
  "assetType":"crypto",
  "asset":"sui",
  "symbol":"SUI",
  "name":"Sui",
  "marketTypes":["spot","perpetual"],
  "sentAt":1792548000123456
}
```

- `name` uses MarketHub's configured display-name mapping, falling back to the
  symbol when no name is configured. It is not externally verified metadata.
- `marketTypes` describes registered Mainnet CEX/DEX observation coverage.
  Values are unique and ordered Spot before Perpetual. At least one is present.
  It is not a statement about fresh prices, history availability, Composite
  eligibility or TradeHub execution support.
- A known asset remains retrievable when trades or current prices are absent.
  No `live`/`stale` status is added to this metadata result.
- Unknown assets return `not_found`. Catalog failures remain failures.
- `sentAt` is the Gateway's response send time in Unix microseconds, not
  the time the metadata changed or a price was observed.
- No price, market counts, icon URL or historical metrics are bundled here.

## Go

Use DTOs from this package and the composed HTTP client:

```go
result, err := client.MarketHub().Asset().Get(ctx, asset.Params{
    AssetType: market.AssetClassCrypto,
    Asset: "sui",
})
```

See `httpclient/asset_example_test.go` for explicit HTTP transport construction.
The client makes one call, preserves caller parameters, validates result scope,
and does not retry automatically.

## Rollout

SDK and MarketHub backend support are prepared. Public Gateway admission,
pricing, quotas, final send-time stamping and Console integration follow after
their separate review. This contract does not announce a deployed free endpoint.
