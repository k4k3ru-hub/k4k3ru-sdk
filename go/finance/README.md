# Finance primitives and market models

`market` and `orderbook` are copied from the service's `shared/finance` as the
first step of the finance migration. The service packages remain in place for
callers that have not migrated. The SDK does not import service packages or add
production dependencies.

Import the owning packages directly:

```go
import (
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
    "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/orderbook"
)
```

`market` owns Venue, Symbol, Chain, MarketType, AssetClass, EventType, market
data models and MarketRef. Models use these SDK-owned types instead of the
service's subscription types. `orderbook` provides the copied snapshot and
store implementations. Market catalogs, adapters, subscriptions and billing
remain service responsibilities.

The canonical perpetual market value is **`perpetual`**. Use
`market.MarketTypePerpetual`; `market.MarketType.Validate` rejects `perp`.
Other market types in the shared enum do not imply support in every API.
The new MarketHub Scalping Params accept only `spot` and `perpetual`.
TradeHub Scalping also rejects `perp`, including saved settings during
resubscription. No automatic conversion or stored-data migration is performed.
Existing APIs outside this
migration retain their own contracts.

Symbol validation uses `Validate` as its single source of rules. `IsValid`
delegates to `Validate() == nil` for callers that need a boolean:

```go
symbol := market.Symbol("NEW/USDC")
valid := symbol.IsValid() // true: delegates to Validate, with no known-symbol list
err := symbol.Validate() // nil: nonempty and at most 16 bytes
```

`Validate` checks for an empty symbol or a length exceeding 16 bytes. It does not
enforce pair syntax, normalize the value, or check market existence. Symbol
constants are convenient values, not an allowlist. MarketHub Scalping separately
requires one `BASE/QUOTE` symbol without wildcards or multiple symbols. The
server must resolve actual market metadata and verify its symbol. The retained
service `shared/finance` package has not yet adopted this SDK change.

MarketRef requires a known Venue, a Network, and exactly one of PoolID or
VenueSymbol. Pools also require Chain. Identifier case is preserved; scope
names are normalized. Recognizing a venue or chain does not establish adapter
support or asset equivalence. New shared AssetRef and AssetMetadata types are
not part of this step; existing TradeHub references remain in executionrule.

See [MarketHub Scalping Params](../jsonrpc/markethub/scalping/README.md) and
[TradeHub Scalping](../jsonrpc/tradehub/scalping/README.md).
