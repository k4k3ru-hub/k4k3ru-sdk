# MarketHub Scalping request parameters

This package currently implements the shared **request DTO only** for planned
MarketHub Scalping Get and Subscribe operations. Result/ACK/event DTOs, client
composition, method registration and server routing are separate migration
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
    Markets: []market.MarketRef{{
        Venue: market.Hyperliquid, Network: onchain.NetworkMainnet, VenueSymbol: "SUI",
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

MarketRef requires venue/network and exactly one of poolId or venueSymbol.
Its Chain and Network fields directly use `onchain/go/core` types. Pools require
chain; use constants such as `onchain.ChainSui` and `onchain.NetworkTestnet`.
Network validation accepts custom names but requires valid UTF-8, 1–16 bytes,
and no whitespace or control characters after scope normalization. Network is
never defaulted. Chain support comes from the onchain registry, while actual
venue/deployment support remains a service check.
Normalized duplicate references are rejected. Asset IDs,
decimals, symbol matching, data quality and venue capabilities must be resolved
by the service; the SDK does not infer them from the symbol string. Request
validation alone does not establish observation availability.

There are no execution rules, trading thresholds, quantities or order IDs in
these observation parameters. Existing TradeHub Scalping parameters retain
their execution settings and asset references.
