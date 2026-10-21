# MarketHub.Asset.List

Discover registered production base assets from the same in-memory
Registry/Catalog as Asset.Get. Configured CEX and DEX markets are included;
Testnet, quote-only assets and unregistered discoveries are excluded. Reads
need no request-time Venue RPC, price history or Parquet access.

## Request

```json
{
  "id": "assets-1",
  "method": "MarketHub.Asset.List",
  "params": {
    "assetType": "crypto",
    "marketType": "spot",
    "search": "sui",
    "page": 1,
    "limit": 25
  }
}
```

- `assetType` is required and currently accepts only `crypto`.
- `marketType` optionally filters by `spot` or `perpetual`; omission selects
  both. Explicit empty or null values are invalid.
- `search` is an optional case-insensitive substring of any one of `asset`,
  `symbol` or `name`. Omission or an empty string selects all assets. Search
  uses literal text, not a regular expression; whitespace is not trimmed.
  At most 64 Unicode characters are accepted; control characters are invalid.
- `page` defaults to 1. `limit` defaults to 100, with range 1–500.
  Explicit zero, negative, fractional or null pagination is invalid.
- Unknown fields, duplicate keys and explicit null selectors are rejected.

## Result

```json
{
  "assetType": "crypto",
  "assets": [
    {
      "asset": "sui",
      "symbol": "SUI",
      "name": "Sui",
      "marketTypes": ["spot", "perpetual"]
    }
  ],
  "page": 1,
  "limit": 25,
  "total": 1,
  "sentAt": 1792566000123456
}
```

- One entry per canonical Catalog `asset`; matching symbols never merge
  distinct token identities. Existing Asset.Get identity and name rules apply.
- `marketTypes` retains full registered coverage, ordered `spot`, `perpetual`,
  even if the request filters by one type. Coverage does not promise price
  availability, Composite eligibility or TradeHub execution support.
- Assets remain discoverable without trades or retained prices. No freshness
  status, prices, counts of markets or historical metrics are bundled here.
- Order is lexicographic `(symbol, asset)` ascending; it is not price ranking.
- `total` counts matching assets before pagination. No match or a page beyond
  the end returns a non-null empty `assets` array. Catalog failures and
  conflicting metadata remain errors; they are not empty successful results.
- `sentAt` is the final Gateway send time in Unix microseconds. Individual
  entries do not repeat the asset type or send time.
- Each request reads the current Catalog. Separate pages are not pinned to a
  catalog version; changes between requests can require restarting pagination.

## Go

```go
result, err := client.MarketHub().Asset().List(ctx, asset.ListParams{
    AssetType: market.AssetClassCrypto,
    MarketType: market.MarketTypeSpot,
    Search: "sui",
    Limit: 25,
})
```

Types are owned by `jsonrpc/markethub/asset`: `ListParams`, `ListResult` and
`ListedAsset`. `AssetClient` is composed by the existing HTTP client constructor
using its injected transport. The operation normalizes a copy, makes one call,
validates metadata, pagination and filter scope, and neither retries nor
automatically fetches further pages. The existing Get API is unchanged.

## Rollout

SDK and MarketHub backend support are implemented. Gateway admission, pricing,
quotas, final send-time stamping and Console integration require the next
review. MarketHub's HTTP handler requires a signed Gateway-service credential;
WebSocket is unsupported. This stage does not publish a free public endpoint.
