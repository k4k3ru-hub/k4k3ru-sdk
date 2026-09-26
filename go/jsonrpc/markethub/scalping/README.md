# MarketHub Scalping parameters and observations

This package implements request and snapshot DTOs for planned MarketHub
Scalping Get and Subscribe operations. ACK/event DTOs, client
composition, method registration and server routing remain separate migration
steps. Importing this package does not make those RPC operations available.

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
not implement local OrderBook/AMM pricing, ranking or spread calculation.

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
Current book/AMM rankings, spread calculation,
public Get/Subscribe delivery and the TradeHub subscription migration remain
subsequent implementation steps.
