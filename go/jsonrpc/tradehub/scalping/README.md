# TradeHub Scalping contracts

Import request, result, and subscription DTOs from
`github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping`.
Import shared round-trip rules and market/asset references from
`github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule`.
There are no root-package aliases or new production dependencies.

The method identifiers are `jsonrpc.MethodTradeHubScalpingGet`,
`MethodTradeHubScalpingSubscribe`, and `MethodTradeHubScalpingUnsubscribe`.
This addition supplies the public data contract. It does not install a TradeHub
handler or add a `websocket.Module` Scalping client. HTTP JSON-RPC callers can
use the existing envelopes and their own transport after server support is
deployed. WebSocket composition and the execution providers are separate work.

## Request

Get and Subscribe share `Params`. `marketType` appears once, directly inside
JSON-RPC `params`, alongside `baseAsset`, `quoteAsset`, `markets`, `conditions`,
and `executionRule`. Every field is required. `executionRule.open` and
`executionRule.close` are both required, non-null objects. A future ordinary
Swap wrapper can omit the entire rule for a single swap; an incomplete Rule is
never valid. Existing Swap request/response types are unchanged.

The following is a **params-only structure example**, with placeholder asset
and pool IDs. Replace those with resolved metadata; the numbers illustrate
units and are not trading recommendations or SDK defaults.

```json
{
  "marketType": "spot",
  "baseAsset": {
    "chain": "sui", "network": "mainnet", "assetId": "0x2::sui::SUI"
  },
  "quoteAsset": {
    "chain": "sui", "network": "mainnet", "assetId": "USDC_COIN_TYPE"
  },
  "markets": [
    {"venue": "cetus", "chain": "sui", "network": "mainnet", "poolId": "OPEN_POOL_ID"}
  ],
  "conditions": {
    "windowMs": 30000,
    "maximumDataAgeMs": 2000,
    "priceChangeBps": {"minimum": "25"},
    "tradeCount": {"minimum": 10}
  },
  "executionRule": {
    "open": {
      "spot": {"amount": "1000000"},
      "limitPrice": "2",
      "maximumSlippageBps": 100,
      "executionTtlMs": 30000
    },
    "close": {
      "takeProfit": {"type": "price", "value": "2.1"},
      "stopLoss": {"type": "price", "value": "1.9"},
      "spot": {
        "markets": [
          {"venue": "cetus", "chain": "sui", "network": "mainnet", "poolId": "CLOSE_POOL_ID"}
        ]
      },
      "maximumSlippageBps": 100,
      "executionTtlMs": 30000
    }
  }
}
```

For a Perp request, set `marketType` to `perp`, provide native markets such as
`{"venue":"hyperliquid","network":"mainnet","venueSymbol":"SUI"}`,
replace `open.spot` with `open.perp`, and omit `close.spot`:

```json
{
  "side": "short",
  "quantity": "1000000000",
  "leverage": 1,
  "marginMode": "isolated"
}
```

This fragment is the `open.perp` object, not a complete request. With Sui SUI
as the reference BaseAsset (9 decimals), it describes one SUI of underlying
quantity. It is not a USDC margin amount or the venue's native wire quantity.
Perp mode supports the structural contract for linear underlying quantities;
server-side instrument capabilities, lot sizes, leverage limits, collateral,
and available margin must still be checked. The SDK supplies no leverage default.

## References and units

- `AssetRef`: exactly one of `chain` or `venue`, plus `network` and `assetId`.
  The server resolves decimals and verifies the reference. Do not use a ticker
  to assert cross-chain equivalence or treat a Perp instrument as a spot token.
- `MarketRef`: `venue` and `network`, plus exactly one of `poolId` or
  `venueSymbol`. Pools require `chain`. Native symbols and asset/pool IDs retain
  their case; scope names are trimmed and lowercased. A native order book may
  also carry its explicit chain. Network is never defaulted.
- Spot `amount` uses reference QuoteAsset atomic units; Perp `quantity` uses
  reference BaseAsset atomic units. Both are positive base-ten integer strings.
  Conversion to each destination requires verified asset mapping and exact
  decimal/lot conversion. Equal symbols or equal raw integers are insufficient.
- Prices use decimal strings in QuoteAsset per BaseAsset token units. A buy
  Open limit is an upper bound; a short Perp Open limit is a lower bound.
- Triggers are `price` (positive decimal) or `return_bps` (signed decimal).
  TP/SL are Close triggers, not a shared Close limit price. Price direction,
  realized cost, fees, gas, funding, and return denominators remain execution
  engine responsibilities; DTO validation does not evaluate trading decisions.
- Close quantities come from the corresponding unresolved position/OMS
  allocation. Spot allows different venues/chains only with corresponding
  reserved inventory. Perp closes the original account/instrument position;
  a trade on another venue is not its Close. A caller cannot disable reduce-only
  behavior through these types.
- Slippage must be explicitly supplied, including zero, and is within
  0..10000 bps. TTL and maximum holding time, when set, must be positive.
  TTL concerns preparation validity, not venue order time-in-force.

## Conditions and normalization

All supplied conditions combine with AND, with at least one supplied metric.
Ranges are inclusive and require at least one bound. Missing pointers are
absent values; explicit zero bounds are preserved.

| Field | Unit |
| --- | --- |
| `priceChangeBps` | signed decimal basis points |
| `quoteVolume` | nonnegative integer reference QuoteAsset atomic units |
| `tradeCount` | nonnegative count |
| `buyVolumeRatioBps` | decimal basis points in 0..10000 |
| `windowMs`, `maximumDataAgeMs` | positive milliseconds |

Decimal/integer strings are compared with exact arithmetic, never float64.
Exponent notation, fractions such as `1/2`, leading `+`, and non-finite values
are rejected. Numeric strings are bounded to 384 bytes; scope names to 64 bytes;
asset/pool/native instrument IDs to 256 bytes. Normalize trims numeric strings
but does not change their scale or supply thresholds. Market lists must be
nonempty and contain no duplicate normalized references.

`Params.Normalize()` makes an independent copy of slices and optional values.
Call Normalize and Validate before sending Go-constructed requests. JSON
decoding validates parameters and rejects unknown fields at every level,
duplicate field names (including case variants), and missing/null required
objects. Decode failure leaves the receiver unchanged.

## Result and subscription

`Result` contains `evaluationId`, the common market type and reference asset
metadata, `evaluatedAt`, and `markets`. Every requested market has one evaluation:

| Status | Metrics | Candidate | Reasons |
| --- | --- | --- | --- |
| `matched` | required | required | absent |
| `not_matched` | required | absent | optional |
| `unavailable` | optional partial metrics | absent | required |

Candidate contains `candidateId`, a positive `revision`, and `expiresAt` after
the evaluation time. Its market is the parent evaluation's MarketRef. IDs stay
stable during the same signal activation; new snapshots have new EvaluationIDs.
The service must implement this lifecycle; the DTOs do not generate these IDs.

Metrics include window boundaries, last observation time, and nullable observed
indicators in the same units as Conditions. Unknown/unavailable observations
are not filled with zero. All timestamps are UTC Unix milliseconds. Reasons
are stable codes, for example `data_gap`, `insufficient_history`, `stale_data`,
`unknown_trade_side`, `unsupported_execution`, or `asset_mapping_unavailable`.
Observed statistics are not accounting quantities; the engine must establish
window coverage, asset conversion, numeric precision, and rounding before
evaluating thresholds.

Use `Result.Validate()` for snapshot invariants and `Result.ValidateFor(params)`
to additionally check market/asset identity, complete market coverage, requested
metric presence, and observation age at evaluation. This does not re-evaluate
the signal or compare expiry against the caller's current clock. Candidate
selection and Prepare must do fresh execution checks.

Get/Subscribe results contain **no prepared execution, signing payload, OMS
order ID, or inventory reservation guarantee**. Agent selects and reserves
funds before Prepare. Prepare does not write OMS; first Submit creates and
links the OMS order before transmission. This SDK change implements none of
those side effects.

Subscribe uses Params directly and returns SubscribeResult with an opaque
`subscriptionKey`. SubscriptionEvent carries a positive sequence and exactly
one `snapshot` (Result) or `error` (code and explicit retryability). Snapshots
replace the full requested-market state, so not_matched/unavailable withdraw
old candidates. The consumer checks the active subscription key and monotonic
sequence; a new connection gets a new subscription key. Unsubscribe accepts
the key and acknowledges it. Stopping candidate notifications does not cancel
existing Open orders or their independently persisted Close monitoring.

## Verification

Tests cover Spot/Perp requests, Close requirements, optional zero values,
cross-chain Close references, exact range boundaries, strict JSON, non-aliasing
normalization, snapshot states, candidate expiry, request/result identity,
metadata decimal presence, and subscription event variants. They use no live
venues, credentials, trading balances, or transactions.
