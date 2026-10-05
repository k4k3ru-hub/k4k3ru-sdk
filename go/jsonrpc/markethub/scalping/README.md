# MarketHub Scalping parameters and observations

This package owns the request, snapshot and subscription DTOs for MarketHub
Scalping. The SDK supports `MarketHub.Scalping.Subscribe` and
`MarketHub.Scalping.Unsubscribe` through `websocket.Module.MarketHubScalping()`.
Deploy the corresponding Gateway, CRM authentication, MarketHub and TradeHub
internal-client changes together. Public
`MarketHub.Scalping.Get` remains a separate step. TradeHub uses the internal
MarketHub Run feed.

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

Optional `buy.quantity` and `sell.quantity` specify independent exact inputs:

```json
{
  "buy": {"quantity": {"amount": "100000000", "decimals": 6}},
  "sell": {"quantity": {"amount": "1000000000", "decimals": 9}}
}
```

With the default `kind: "exact-input"`, for SUI/USDC this evaluates spending 100 USDC to buy SUI and selling 1 SUI
for USDC. Buy input is Quote; Sell input is Base. Each quantity requires both
`amount` and `decimals`: a positive unsigned integer string up to 384 digits,
and a decimal scale from 0 to 255. An omitted/null side or quantity disables
quantity calculations for that side. Empty sides normalize to omission. The
legacy top-level `baseQuantity` is rejected, with no compatibility conversion.
Historical metrics do not depend on these inputs. No trading thresholds,
execution rules or order IDs are added to MarketHub observations.

Spot sides also accept `kind: "exact-output"` with a required positive `quantity`.
It fixes **net received** Base for Buy or Quote for Sell, not the payment amount.
Omitted `kind` and explicit `exact-input` normalize to the same subscription key;
exact output has a distinct key. For example,
`"buy": {"kind":"exact-output","quantity":{"amount":"1000000","decimals":9}}`
estimates buying back exactly 0.001 SUI in SUI/USDC. No trade is submitted.

Perpetual **Buy** also accepts `kind: "exact-output"`: its quantity is fixed Base
contract size. For example, `{"kind":"exact-output","quantity":{"amount":"1","decimals":0}}`
prices a 1 SUI Long or a 1 SUI Short cover. It does not mean receipt of a token,
margin deposit or leveraged quantity. Buy fees are charged in Quote on the walked
notional; the contract size is not reduced by those fees. Perpetual Sell continues
to use `exact-input` Base contract quantity; Sell `exact-output` is unsupported.
Both directions return `netPrice` and `fees`, omitting Spot settlement quantities.

Spot VWAP results include `netPayQuantity` (fee-inclusive input) and
`netReceiveQuantity` (fee-inclusive output). Amounts come from raw calculations,
not inversion of the rounded public `netPrice`. Reference/fallback/unavailable
prices omit both quantities, as do Perpetual prices. Exact-output payments round
up at the input precision; received quantity must be exactly representable at
output precision. OrderBook depth includes received-token fees. Cetus, Turbos,
and Momentum use immutable cached Pool/tick state with normal swap fees and no
network fallback. Other AMM adapters without exact-output support retain
`fallback_reference` when a reference is available. An insufficient or incomplete
snapshot never supplies a fabricated payment estimate.

The flat Result contains `evaluatedAt` and optional `priceEvaluatedAt` (Unix milliseconds), optional consolidated
`ohlc` and `metrics`, and `buy` / `sell` lists of concrete markets. It has no
network groups or issues array. MarketPrice status is `reference`, `vwap`,
`fallback_reference` or `unavailable`. These types define the contract; they do
not perform calculations themselves. MarketHub's internal snapshot service
connects retained OrderBook/AMM pricing, ranking and spread calculation.

Current prices use `netPrice`, including trading fees and quantity-dependent
price impact, excluding gas. The former `price` and `receiveQuantity` fields are
removed. Spot VWAP entries return `netReceiveQuantity`: Base received for Buy,
Quote received for Sell. AMMs calculate the normal-fee output against the same
retained state. OrderBook Buy consumes Quote across asks; Sell consumes Base
across bids. Spot taker fees reduce the received asset; fee amounts are rounded
up and outputs down to that asset's atomic units. Inputs must be exactly
representable at the input asset's precision. Full input coverage and positive
output are required; otherwise a known fee-adjusted reference can be returned.

Spot `netPrice` is Quote input / published Net Base output for Buy and published
Net Quote output / Base input for Sell. Perpetual entries have `netPrice` and
`fees`, but never `netReceiveQuantity` or `netPayQuantity`: they do not deliver spot tokens. Their
Buy price is (Quote notional + Quote fee) / Base size; Sell price is
(Quote notional - Quote fee) / Base size. Request quantities remain notional or
Base size, independent of margin and leverage. Prices are published to 18
decimal places, ties to even. Different Buy/Sell sizes mean spread is not
roundtrip PnL.

Optional `feeAccounts` selects the actual Hyperliquid trading account for both
directions (not an API signing wallet):

```json
{
  "feeAccounts": [{
    "venue": "hyperliquid",
    "network": "mainnet",
    "address": "0x1111111111111111111111111111111111111111"
  }]
}
```

Each scope must match a requested market target, with at most one account per
venue/network. Omission uses the published standard Tier 0 taker schedule
without account discounts. Account identities are included in subscription
keys. A specified account's unavailable rate never falls back to standard fees.
Rates and supported market fee modifiers are refreshed in the background every
60 seconds by default and remain usable for five minutes after successful
retrieval. Server settings `MARKET_HUB_FEE_REFRESH_INTERVAL` and
`MARKET_HUB_FEE_MAXIMUM_AGE` accept Go duration strings (defaults `1m`, `5m`).
Unknown/expired fees make the price unavailable. Snapshot calculations perform
no fee-fetch RPC and do not wait for refresh; the initial snapshot may therefore
be unavailable while the cache warms.

Reference and VWAP entries share the ranking: Buy ascending, Sell descending,
unavailable entries last. With quantity specified, an uncomputable market
remains `fallback_reference` when a reference price is available. Without
quantity, prices are `reference` and no quantity-based calculations occur.
Spot Buy references are ask / (1 - fee); Sell references are bid * (1 - fee).
Perpetual Buy references are ask * (1 + fee), Sell bid * (1 - fee).
AMMs use the retained marginal price and direction-specific input fee.
Quantity fallback uses the same reference definition, never partial output.
Spread is `(bestBuy - bestSell) / ((bestBuy + bestSell) / 2) * 10000`, using
published `netPrice` values, and can be negative. Its status is `vwap` only if both
winning prices are VWAP; otherwise it is `fallback_reference` for a quantity
request, `reference` without quantity, or `unavailable` if either side is missing.

No age cutoff is applied to current price inputs; the fee-cache validity limit
is separate. `observedAt` records input receipt
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
Base/Quote mapping remain internal to the fee calculation. Input quantities
remain in the Request; market results do not repeat them.

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
it is not labeled as LP-only. Fees are already reflected in `netPrice` and
`netReceiveQuantity`; do not subtract them again. AMM fees are calculated with
normal fee settings on the same frozen state as Net output. Input-asset fees
cannot be subtracted directly from an output-asset quantity.

`taker` represents immediate OrderBook execution fees, using the retained
standard/account rate and supported market modifiers. Known zero fees retain
`amount: "0"` and explicit `decimals`. Quantity omission, reference fallback and
unavailable prices omit `fees`, `netPayQuantity` and `netReceiveQuantity`; gas is excluded. No
`issues` array or additional fee status is added.

Service coverage depends on configured adapters: directional Net AMM output is
implemented for Cetus, Bluefin Spot, Momentum, Turbos, Uniswap V3/V4 and
Aerodrome Slipstream. Hyperliquid fee metadata currently covers ordinary
USDC-quoted Spot and native USDC Perpetual markets. Other quote assets/HIP-3
markets need additional fee-modifier metadata and are unavailable. Perpetual
market observation also requires its execution-market metadata to be registered;
the DTO alone does not enable a new adapter.

Price analytics use validated OrderBook midpoints or marginal Pool state prices,
without order quantities, account fees or gas. Prices are canonical Quote/Base:
the median within each venue, then the median across venues. An even-sized
median is the mean of its two central values. OHLC covers every admitted state
change, rather than trade extrema or executable quotes.

`priceEvaluatedAt` (Unix milliseconds) is the latest common confirmed time of
all resolved markets, floored to milliseconds. Price analytics cover
`[priceEvaluatedAt-windowMs, priceEvaluatedAt)`. It can trail `evaluatedAt`, the
actual calculation time. Internal admission, receipt and source timestamps
retain sub-microsecond precision. Late inputs are not backdated. Every market
remains in the basket; missing coverage, disconnection or warmup omit the price
series and its timestamp. No trade-price fallback is used.

A state can remain constant during a verified no-trade interval. A socket or
local timer alone never establishes coverage, and a gap is never interpolated.
Sui checkpoint watermarks confirm progress even without Pool transactions.
Hyperliquid full book updates confirm the observed book stream. EVM retained
state currently advances only on validated input updates; an idle log stream
alone does not advance the price watermark. Reconnection starts a new history.
State history is bounded by market/point caps and two minutes of storage,
including a one-minute allowance for watermark lag; the public window remains
at most 60 seconds. If the required historical interval is no longer retained,
price analytics are omitted. Current `netPrice` ranking is independent.

Trade activity continues to use actual event time in
`[evaluatedAt-windowMs, evaluatedAt)`. No-trade intervals produce known zero
volume/count only when activity observation is continuous. Undefined ratios
or VWAP remain omitted. MarketHub does not make per-evaluation RPC calls.

Metrics include price-change bps, actual Quote volume, event count, buy Base
ratio bps, and aggregate trade VWAP. Quote volume uses Amount/Decimals, selecting
the greatest verified Quote precision across the selected markets and flooring
once after summation. Prices and bps use 18 decimal places, ties to even.
Quantities remain exact before output conversion. An unavailable field is
omitted rather than reported as zero.

`trend` compares `[T-5s,T)` with `[T-10s,T-5s)`, with `T=priceEvaluatedAt`
for price changes and `T=evaluatedAt` for trade activity: the difference in price-change
bps, Quote-volume percentage change in bps, and the difference in buy-ratio bps.
Windows below 10 seconds omit trend. Missing inputs omit only affected values;
a zero previous Quote volume prevents calculating its change rate.

`realizedVolatilityBps` uses the closing state price of each complete UTC
one-second interval within the confirmed price window:
`sqrt(sum(log(P[i]/P[i-1])^2)) * 10000`, without annualization. At least three
prices are required, with no missing full-second samples. Partial edge seconds
are excluded from volatility, while remaining part of OHLC. This sampling
interval is independent of delivery frequency and the five-second trend ranges.
The implementation uses guarded arbitrary-precision arithmetic and requires
the same rounded output at successive precisions; unstable output is omitted.



## Observation runs

Run accepts the existing `Params` unchanged. ACK contains
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

Public Run is signed and costs 100 ticks at creation, then 100 ticks per
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

The start RPC is `MarketHub.Scalping.Subscribe`; the previous
`MarketHub.Scalping.Run` method is no longer routed. Use
`module.MarketHubScalping().Subscribe(ctx, params)` to start the existing ACK and
continuous Snapshot stream. The request fields, subscription key, event type,
interval, Credit policy and `Unsubscribe` operation retain their existing meaning.

### Fee-exclusive limit evidence

Optional `grossPrice` is Quote/Base for the same market, direction and requested
quantity with trading fees excluded. Reference and fallback results use the best
OrderBook quote or pool marginal price without a quantity calculation. A full
VWAP uses the independently calculated fee-free depth/pool estimate. Ranking,
spread and TP/SL continue to use `netPrice`.

The field is omitted when that independent estimate is unavailable. In
particular, AMM exact-output adapters currently retain only fee-inclusive input;
they do not advertise a gross estimate for a different output quantity. OrderBook
exact-output estimates walk depth again for the same fee-free output target.
