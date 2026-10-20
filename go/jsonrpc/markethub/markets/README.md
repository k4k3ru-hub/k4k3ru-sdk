# MarketHub.Markets.List

Discover registered production markets for one canonical asset and market type.
This is a metadata API: prices, current Composite contributors, execution
estimates, funding and open interest remain separate operations.

This SDK contract is implemented before the server endpoint. MarketHub, Gateway,
Console integration, access/credit policy and deployment are subsequent work.
No production dependency or existing method contract is changed.

## Request

```json
{
  "method": "MarketHub.Markets.List",
  "params": {
    "assetType": "crypto",
    "asset": "sui",
    "marketType": "perpetual",
    "page": 1,
    "limit": 100
  }
}
```

| Field | Required | Meaning |
| --- | --- | --- |
| assetType | Yes | `crypto`; no default |
| asset | Yes | The same canonical asset ID as Price.Get, e.g. `sui` |
| marketType | Yes | `spot` or `perpetual`; `perp` is not accepted |
| page | No | One-based page; omitted selects 1 |
| limit | No | 1–500 markets per page; omitted selects 100 |

Asset IDs follow Price.Get: 1–64 lowercase ASCII letters, digits or hyphens,
starting with a letter or digit. Asset identifiers are not derived from token
symbols by the client. For example, distinct assets sharing a display symbol
may have scoped catalog IDs. Required selectors are neither defaulted nor
silently changed.

On the wire, explicit page/limit must be positive integers; null, zero,
negative, fractional and unrepresentable values are rejected. Go zero values
omit these optional fields and `Normalize` fills their defaults. Unknown fields,
duplicate fields (including case variants) and trailing JSON values fail.
Failed decoding leaves the receiver unchanged.

The initial catalog covers production/mainnet CEX and DEX markets, across
supported chains and quote assets. There is no request network, account, side,
quantity or timeframe. `mainnet` identifies the production environment for CEX
entries; it does not assert an on-chain CEX order book. The listing covers the
asset as Base, not every market in which the asset appears as Quote.

## Result

The following is an illustrative single-entry page, not a live market count.

```json
{
  "asset": "sui",
  "marketType": "perpetual",
  "markets": [
    {
      "market": {
        "venue": "hyperliquid",
        "network": "mainnet",
        "venueSymbol": "SUI"
      },
      "symbol": "SUI/USDC",
      "liquidityModel": "order-book",
      "baseAssetId": "SUI",
      "quoteAssetId": "USDC",
      "compositeEligible": true,
      "candlesSupported": true
    }
  ],
  "page": 1,
  "limit": 100,
  "total": 1,
  "sentAt": 1792508400000000
}
```

- `market` reuses `finance/market.MarketRef`. AMMs require chain/poolId;
  order books use venueSymbol. Pool ID and venue symbol are mutually exclusive.
  All entries have `network: "mainnet"`. Instrument IDs retain their exact case.
- `symbol` is the canonical Base/Quote pair. `liquidityModel` is `amm` or
  `order-book` for that market; it is not inferred from a venue-wide default.
- Asset IDs are identifiers within the market's scope, including token addresses
  or Coin Types where available. They are nonempty, at most 512 bytes, and have
  no surrounding whitespace or control characters. Equal display symbols do
  not prove equal tokens across chains or pools. Perpetual IDs describe the
  instrument and do not imply delivery of Spot tokens.
- `compositeEligible` means the registered market is eligible for Composite
  calculation. It is not evidence that a current observation passed freshness,
  volume, conversion or quality checks or contributed to the published price.
- `candlesSupported` means this market is supported as a Candles.Get selector.
  It does not promise retained history, full coverage, freshness or executable
  quotes. The current disclosure policy keeps individual CEX candles unavailable
  while permitting CEX contribution to Composite.
- `sentAt` is Gateway send time in Unix microseconds, not observation time.

Each Pool or order-book instrument is a distinct row. Order is ascending
lexicographic order of `(venue, network, chain, poolId, venueSymbol)`, with empty
optional fields ordered as empty strings. The returned identities are normalized,
unique and strictly ordered within each page. Prices do not affect ordering.

`total` counts all matching markets before pagination. A complete page has
`min(limit, remaining)` rows. No matches or a page beyond the end returns
`markets: []` while preserving the total; null is not an empty page. An error
reading the catalog must not be disguised as a successful empty catalog.
Pagination does not pin a catalog revision: registrations can change between
requests. The client performs no implicit next-page fetching or retry.

## Data and reuse

MarketHub will use the shared in-memory Registry/Catalog. List requests do not
read Parquet or call venues. The metadata flags are derived from the same
selection/disclosure policy used by the relevant APIs, not from UI guesses.

Use each row's `symbol` and `market` with the API's owning request contract.
For example, Quote/Funding/OpenInterest take market selectors and a symbol;
Candles takes the asset ID and one market selector. These APIs still enforce
their own supported markets, permissions, availability and pricing.
Existing Candles.Get uses `perp`; adapt that spelling at its boundary rather
than changing this contract's `perpetual`.

## Go client

```go
import (
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/markets"
)

result, err := client.MarketHub().Markets().List(ctx, markets.Params{
    AssetType: market.AssetClassCrypto,
    Asset: "sui",
    MarketType: market.MarketTypePerpetual,
})
if err != nil {
    return err
}
// Use result.Markets; an empty non-null slice is valid.
```

`httpclient.NewClient` composes this operation using the same injected transport
and credential provider as other methods. The client validates the result and
matches asset, market type, page and limit to its request. Error causes remain
inspectable. Contract and fake-transport tests do not access live services.
