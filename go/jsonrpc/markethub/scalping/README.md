# MarketHub Scalping parameters and observations

This package owns the request, snapshot and subscription DTOs for MarketHub
Scalping. The SDK supports `MarketHub.Scalping.Subscribe` and
`MarketHub.Scalping.Unsubscribe` through `websocket.Module.MarketHubScalping()`.
Deploy the corresponding Gateway and MarketHub server changes together. Public
`MarketHub.Scalping.Get` and the TradeHub internal-feed migration remain separate steps.

```go
import (
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
    onchain "github.com/k4k3ru-hub/onchain/go/core"
)

params := scalping.Params{
    MarketType: market.MarketTypePerpetual,
    Symbol:     market.Symbol("SUI/USDC"),
    WindowMS:   scalping.DefaultWindowMS,
    Markets: []market.MarketTarget{{
        Venue: market.Hyperliquid, Network: onchain.NetworkMainnet,
    }},
}
params = params.Normalize()
err := params.Validate()
```

This example illustrates structure, not adapter availability or a trading
recommendation. Each request describes one symbol across 1–64 distinct markets.
`marketType` accepts `spot` or `perpetual`; the legacy spelling `perp` is rejected.
`symbol` is normalized to uppercase, limited to 16 bytes, and requires a single
`BASE/QUOTE` pair. It uses `Symbol.Validate()` for the shared empty/length checks
and adds pair-specific checks here. Neither `Symbol.Validate()` nor its boolean
wrapper `Symbol.IsValid()` restricts symbols to a compiled known-symbol list.

`windowMs` defaults to 60,000 only when omitted from JSON. Its explicit range is
1–60,000; null and zero are invalid. Go callers must set a positive value.
Unknown/duplicate fields and null required fields are rejected. JSON decoding
normalizes parameters and leaves the receiver unchanged on failure. Go callers
use Normalize explicitly before encoding; Validate does not mutate its receiver.

MarketTarget requires venue/network. Chain, poolId and venueSymbol are optional
catalog filters; poolId and venueSymbol cannot both be set. When both instrument
filters are omitted, the service resolves matching configured markets using
marketType, symbol and the requested scope. It deduplicates overlapping targets
and rejects an expansion beyond 64 markets. This does not discover arbitrary
unconfigured pools. MarketRef in results is a concrete identity: it requires
exactly one instrument ID, and pools require chain. Chain and Network directly
use `onchain/go/core` types, such as `onchain.ChainSui` and `onchain.NetworkTestnet`.
Network validation accepts custom names but requires valid UTF-8, 1–16 bytes,
and no whitespace or control characters after scope normalization. Network is
never defaulted. Chain support comes from the onchain registry, while actual
venue/deployment support remains a service check.
Normalized duplicate targets are rejected. Asset IDs,
decimals, symbol matching, data quality and venue capabilities must be resolved
by the service; the SDK does not infer them from the symbol string. Request
validation alone does not establish observation availability.

`baseQuantity` optionally requests local quantity-aware current pricing:
`{"amount":"1000000000","decimals":9}` means one Base token. Both fields
must be present when an object is supplied. Amount is a positive unsigned
integer string, at most 384 characters; decimals is an integer from 0 to 255.
Omitted/null quantity disables quantity-based calculations. Historical metrics
do not depend on this input. Quantity values in results may be zero.
There are no execution rules, trading thresholds or order IDs in these
observation parameters. Existing TradeHub settings remain separate.

The flat Result contains `evaluatedAt` (Unix milliseconds), optional consolidated
`ohlc` and `metrics`, and `buy` / `sell` lists of concrete markets. It has no
network groups or issues array. MarketPrice status is `reference`, `vwap`,
`fallback_reference` or `unavailable`. These types define the contract; they do
not perform calculations themselves. MarketHub's internal snapshot service
connects retained OrderBook/AMM pricing, ranking and spread calculation.

Current prices, `quoteQuantity`, rankings and spread are gross: swap fees and
gas are excluded, while quantity-dependent price impact is included. AMM gross
amounts come from a separate zero-swap-fee simulation on the same retained
inputs and Base quantity. Quantity-aware native OrderBook prices integrate
the entire requested amount; insufficient depth falls back to a reference
price without a partial `quoteQuantity`. Buy Quote quantities round up and
Sell quantities round down to the market's minimum units, after summation.
Prices round to 18 decimal places, ties to even.

Reference and VWAP entries share the ranking: Buy ascending, Sell descending,
unavailable entries last. With quantity specified, an uncomputable market
remains `fallback_reference` when a reference price is available. Without
quantity, prices are `reference` and no quantity-based calculations occur.
Spread is `(bestBuy - bestSell) / ((bestBuy + bestSell) / 2) * 10000`, using
published prices, and can be negative. Its status is `vwap` only if both
winning prices are VWAP; otherwise it is `fallback_reference` for a quantity
request, `reference` without quantity, or `unavailable` if either side is missing.

No age cutoff is applied to current prices. `observedAt` records input receipt
or verification time, not calculation time. Missing inputs, lost synchronization,
identity mismatches and concurrent AMM invalidation can still prevent a price.
`lastTradeAt` separately records the last observed, admitted, non-canceled
Trade/Swap **event time**, in Unix milliseconds. Unknown times are omitted;
receipt time never substitutes for event time. It can be older than the history
window and can remain present for an unavailable price. Its bounded in-memory
record survives raw-event expiry, but not a process restart or metadata
replacement. A cancellation clears it if no retained predecessor can be found.

Each Buy/Sell entry can contain a `fees` object keyed by fee kind:

```go
type Fees struct {
    Swap  *Fee `json:"swap,omitempty"`
    Taker *Fee `json:"taker,omitempty"`
}

type Fee struct {
    Token    FeeToken        `json:"token"`
    Quantity market.Quantity `json:"quantity"`
}

type FeeToken struct {
    AssetID string `json:"assetId"`
    Symbol  string `json:"symbol"`
}
```

`token.assetId` identifies the charged asset in the enclosing market's
chain/network (or venue/network) namespace. `token.symbol` is descriptive;
it must not be used alone to establish asset identity. Token0/Token1 and
Base/Quote mapping remain internal to the fee calculation. The existing
request `baseQuantity` and gross result `quoteQuantity` retain their meanings.

For example, this is a **fragment inside a Buy entry** for a Sui Testnet pool;
the Token ID and amounts illustrate the shape, not a deployed pool or live quote:

```json
{
  "fees": {
    "swap": {
      "token": {"assetId": "0x3::usdc::USDC", "symbol": "USDC"},
      "quantity": {"amount": "3000", "decimals": 6}
    }
  }
}
```

This estimates a 0.003 USDC swap fee. A Sell entry can instead identify SUI
and its own decimal scale. `swap` includes both LP and protocol fee shares;
it is not labeled as LP-only. It is estimated with normal fee settings on the
same retained inputs used for gross pricing. Because the normal-fee and zero-fee
simulations can follow different price paths, adding/subtracting a converted
fee does not necessarily reconstruct a net quote.

Only AMM `swap` fees are currently produced by MarketHub. `taker` represents
immediate OrderBook execution fees when known; account-dependent fees are not
inferred. Unknown fees are omitted rather than zero-filled. Known zero fees
retain `amount: "0"` and explicit `decimals`. Quantity omission, reference
fallback and unavailable prices omit `fees`; gas is excluded. No `issues`
array or additional fee status is added.

Historical analytics use event time in `[T-windowMs,T)`. For each UTC second
(clipped at both window edges), the service computes market Quote/Base VWAP,
the median within each venue, then the median across venues. OHLC describes
that representative series, not raw trade extrema or an executable quote.
An even-sized median is the mean of its two central values. No forward fill or
interpolation is performed. Missing coverage or a missing constituent price
prevents publishing a complete price series; independent totals can still exist.

Metrics include price-change bps, actual Quote volume, event count, buy Base
ratio bps, and aggregate trade VWAP. Quote volume uses Amount/Decimals, selecting
the greatest verified Quote precision across the selected markets and flooring
once after summation. Prices and bps use 18 decimal places, ties to even.
Quantities remain exact before output conversion. An unavailable field is
omitted rather than reported as zero.

`trend` compares `[T-5s,T)` with `[T-10s,T-5s)`: the difference in price-change
bps, Quote-volume percentage change in bps, and the difference in buy-ratio bps.
Windows below 10 seconds omit trend. Missing inputs omit only affected values;
a zero previous Quote volume prevents calculating its change rate.

`realizedVolatilityBps` uses consecutive complete one-second consolidated prices:
`sqrt(sum(log(P[i]/P[i-1])^2)) * 10000`, without annualization. At least three
prices are required, with no missing full-second samples. Partial edge seconds
are excluded from volatility, while remaining part of OHLC. This sampling
interval is independent of delivery frequency and the five-second trend ranges.
The implementation uses guarded arbitrary-precision arithmetic and requires
the same rounded output at successive precisions; unstable output is omitted.
Public Get/Subscribe delivery and the TradeHub subscription migration remain
subsequent implementation steps.


## Observation subscriptions

Subscribe accepts the existing `Params` unchanged. ACK contains
`{"subscriptionKey":"MarketHub.Scalping:<opaque-key>","intervalMs":1000}`.
The first full Snapshot follows ACK; later full Snapshots target a one-second
interval, even without trades. Arrival timing is not guaranteed. Observation
windows retain millisecond precision; delivery does not alter the one-second
volatility samples or five-second trend windows.

Notifications use the existing outer event envelope:

```json
{
  "e": "msc",
  "data": {
    "subscriptionKey": "MarketHub.Scalping:<opaque-key>",
    "snapshot": {
      "evaluatedAt": 1790380800000,
      "metrics": { "spread": { "status": "unavailable" } },
      "buy": [],
      "sell": []
    }
  }
}
```

This example is an unavailable observation, not a fabricated zero price.
The SDK exposes `Events() <-chan scalping.Result`, `Errors() <-chan error` and
`Reference() scalping.SubscribeResult` on `MarketHubScalpingSubscription`.
Replace the previous Result entirely, including omitted fields. A slow reader
receives only the latest unread Snapshot; intermediate evaluations are not an
event history. Cached initial Snapshots keep their original `evaluatedAt`.
A terminal error or connection close clears buffered observations and closes
the handle. Discard the previous value and explicitly subscribe again to resume.

Use `module.MarketHubScalping().Unsubscribe(ctx, handle)` to stop delivery.
The wire request and ACK contain only `subscriptionKey`. The same active
conditions on the same SDK client return the existing handle; unsubscribe that
handle once. The key includes every request condition, ignores target order,
and is not a credential. The existing `module.Scalping()` / `e="sc"` remain
TradeHub execution-candidate APIs.

Public Subscribe is signed and costs 100 ticks at creation, then 100 ticks per
minute from the first successful ACK. Billing is per WebSocket connection and
normalized conditions, independent of market count, optional quantity and event
count. Same-connection duplicates do not incur another charge or reset the
billing period; a different connection or different conditions do. Missing
metrics do not pause billing. Initial debit occurs before upstream acceptance;
upstream failure does not automatically refund it. Unsubscribe costs zero and
stops new billing periods. Reconnection creates a new billable subscription.
Internal service-to-service subscriptions are signed and are not separately
charged these public ticks. Credit exhaustion terminates only the affected
subscription and is reported through `Errors()`.
