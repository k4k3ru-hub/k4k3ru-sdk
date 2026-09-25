# TradeHub Scalping contracts

Import request, result, and subscription DTOs from
`github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping`.
Import shared round-trip rules and market/asset references from
`github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule`.
`Params.MarketType` and `Result.MarketType` use the owning
`github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market` type. Go callers should use
`market.MarketTypeSpot` or `market.MarketTypePerpetual` for these fields.
There are no root-package aliases or new production dependencies.

The implemented subscription identifiers are `jsonrpc.MethodTradeHubScalpingSubscribe`
and `MethodTradeHubScalpingUnsubscribe`. Use `websocket.NewModule` and
`module.Scalping()` for authenticated candidate notifications. The existing
`MethodTradeHubScalpingGet` constant is reserved for future order/position state
retrieval; it is not a one-shot candidate evaluation operation.

## Request

`Params` describes immutable execution settings. `SubscribeParams` has two modes:

- Start: `IdempotencyKey` plus `Params: &settings` in Go. The JSON encoding flattens
  the settings, keeping `marketType` directly beside `idempotencyKey` in JSON-RPC
  params. Repeating the same key and normalized settings for the same account
  returns the same durable execution ID. Different settings conflict.
- Resume: `ExecutionID` only. Any additional JSON field, including null settings
  or an idempotency key, is rejected. Saved settings cannot be overwritten.

`marketType`, `baseAsset`, `quoteAsset`, `markets`, `conditions`, and
`executionRule` are required for a new execution. `executionRule.open` and
`executionRule.close` are both required, non-null objects. A future ordinary
Swap wrapper can omit the entire rule for a single swap; an incomplete Rule is
never valid. Existing Swap request/response types are unchanged.

The following is a **params-only structure example**, with placeholder asset
and pool IDs. Replace those with resolved metadata; the numbers illustrate
units and are not trading recommendations or SDK defaults. The omitted
`conditions.windowMs` defaults to 60000 (60 seconds).

```json
{
  "idempotencyKey": "my-scalping-start-001",
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

For a perpetual request, set `marketType` to `perpetual`, provide native markets such as
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

Scalping rejects the former `perp` value in requests and results. Saved settings
containing `perp` also fail on resume or same-key retry; they are not automatically
converted or rewritten. Create a new execution using `perpetual` and a new
idempotency key. The `open.perp` field name is unchanged.

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
| `windowMs` | integer milliseconds in 1..60000; omitted JSON defaults to 60000 |
| `maximumDataAgeMs` | required positive integer milliseconds |

An omitted JSON `windowMs` becomes `DefaultWindowMS` (60000) before validation
and persistence. Explicit zero, null, and values above `MaximumWindowMS` (60000)
are rejected, never replaced or clamped. Set `"windowMs": 30000` for a shorter
30-second window. This is the observation period, not the notification interval.
No metric thresholds or data-age limit receive defaults.

Go callers retain the existing `uint64` field and must explicitly set
`Conditions.WindowMS`, for example `WindowMS: scalping.DefaultWindowMS`.
Go's zero value is invalid; `Normalize()` does not supply the JSON omission
default. After JSON decoding, omission and an explicit 60000 produce identical
settings, so a retry with the same idempotency key does not conflict.
Resume uses the saved window without applying a new configuration. Saved windows
above 60000 fail validation when loaded; existing settings are not rewritten.

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

The agreed observation contract uses venue/chain transaction time within
`[evaluatedAt - windowMs, evaluatedAt)`. The start is inclusive and the end is
exclusive. `lastObservedAt` is the latest valid transaction time in that window;
receive time and retransmissions do not refresh it. Unknown transaction time
makes the market unavailable.

| Metric | Definition |
| --- | --- |
| `priceChangeBps` | `10000 * (last window price / first window price - 1)`; unavailable with fewer than two events |
| `quoteVolume` | Sum of exact Quote quantities, converted to reference atomic units and floored once after summation |
| `tradeCount` | Deduplicated normalized Trade/Swap event count after cancellations; not necessarily the venue's individual fill count |
| `buyVolumeRatioBps` | `10000 * buy Base quantity / (buy + sell Base quantity)`; unavailable if any included event has unknown side |

Bps are rounded to 18 decimal places using nearest, ties-to-even rounding.
Thresholds compare the same rounded values published in Metrics. Perp shorts
retain the Base price direction and buy-side meaning. Unknown metrics are omitted,
not replaced with zero; unavailable unrequested indicators alone do not prevent
evaluation of the requested indicators. Missing window coverage, a data gap,
or inability to compute any requested indicator makes the market unavailable.

MarketHub's initial exact-input adapters are Hyperliquid Trade and Cetus Swap.
MarketHub tracks subscription readiness, connection generations and observed losses.
Cetus starts coverage on the first successful upstream notification; Hyperliquid
Spot starts on the Trade subscription ACK. Neither backdates coverage to a trade time.
A new or interrupted connection must collect the entire requested window before
matching. TradeHub reads the signed internal window feed every second and emits
replacement snapshots, including when no new trades arrive.

The initial connected sources are configured Cetus Spot pools and enabled
Hyperliquid Spot instruments. Cetus asset IDs/decimals come from its Token catalog;
Hyperliquid Spot uses metadata `tokenId` and `weiDecimals`, not order size precision.
Asset identity must match chain/venue, network and identifier. Equivalent Sui Move
type addresses are normalized; ticker aliases do not establish cross-venue asset
equivalence. Reversed pairs and unconfigured networks are unavailable. Hyperliquid
Perp exact trades are retained, but its underlying/reference asset mapping remains
unavailable until authoritative metadata is connected.

Candidates use inclusive AND comparisons. Their exclusive `expiresAt` is capped
by two polling intervals (currently 2 seconds), the first millisecond outside
`maximumDataAgeMs`, and the first known event leaving the window. A matched
candidate keeps its ID with increasing revisions; a source restart, observation
interruption, nonmatch, unavailability or expired lease ends that identity.
`open.executionTtlMs` does not determine candidate expiry.

Use `Result.Validate()` for snapshot invariants and `Result.ValidateFor(params)`
to additionally check market/asset identity, complete market coverage, requested
metric presence, and observation age at evaluation. This does not re-evaluate
the signal or compare expiry against the caller's current clock. Candidate
selection and Prepare must do fresh execution checks.

The subscription ACK and every event include the Scalping `executionId` and
`subscriptionKey`. The execution ID survives socket/process restarts; the key
belongs to a connection. This execution groups potential Open/Close trades and
is distinct from a single prepared transaction's `TradeHub.Execution` ID.
Do not pass a Scalping ID to Execution.Submit. There is no OMS order, prepared
transaction, signing payload or inventory reservation created by Subscribe.
Prepare still does not write OMS; first Submit remains the order-creation boundary.

```go
subscription, err := module.Scalping().Subscribe(ctx, scalping.SubscribeParams{
    IdempotencyKey: "my-scalping-start-001",
    Params:         &settings, // scalping.Params, validated before sending
})
if err != nil {
    return err
}
executionID := subscription.Reference().ExecutionID // retain for reconnect
// Consume subscription.Events() and subscription.Errors().
if err := module.Scalping().Unsubscribe(ctx, subscription); err != nil {
    return err
}
resumed, err := module.Scalping().Subscribe(ctx, scalping.SubscribeParams{
    ExecutionID: executionID,
})
if err != nil {
    return err
}
_ = resumed
```

After an interrupted start before receiving the ACK, retry the same start key
and settings. After an ACK, reconnect explicitly using the execution ID.
Authentication is required; the owner is taken from credentials, never request
parameters. Unsubscribe requires both identifiers and acknowledges both. It
stops notifications, leaving saved settings and existing order/Close management
intact. It is not an execution deletion or order cancellation operation.

Notification envelope type `sc` carries `SubscriptionEvent`: a positive sequence
and exactly one full `snapshot` (Result) or `error` (code and retryability).
The server acknowledges before sending the initial update. Available sources
send an initial snapshot then replacement snapshots, including not_matched and
unavailable transitions. The SDK buffers the ACK/event race and rejects events
from old keys or executions and non-increasing sequences. Sequences restart at
one for a new key. Consumers replace candidate state; a stream error or connection
loss invalidates all previous actionable candidates. A `retryable: true` error
keeps the stream open; a false value ends the stream. Transport loss closes the
handle and reports through Errors. Buffer overflow also reports an interruption;
release that handle before resubscribing. No automatic trading or reconnect is
performed.

The service composes the MarketHub-backed evaluator. Unknown asset metadata
initially produces `asset_metadata_unavailable` with `retryable: true`; it never
invents symbols or decimal counts. Once verified metadata is available, warm-up,
data gaps, stale observations and unavailable markets appear in replacement
snapshots. An internal transport failure withdraws previous candidates and is
retried. The stream stays open through these temporary failures.

The initial server admits up to 64 markets per subscription and 256 evaluator
subscriptions per TradeHub process, with 16 concurrent internal reads. Polling
is not a guarantee of detecting every intra-second threshold crossing. Agent
selection, inventory reservation, Prepare, local signing, Submit and OMS recording
remain separate execution work. This feed neither prepares nor submits orders.
Deploy the MarketHub and TradeHub service changes together; no new public SDK
request/result fields or database migrations are required by this connection.

## Verification

Tests cover Spot/Perp requests, Close requirements, optional zero values,
cross-chain Close references, idempotent start/resume forms, exact range boundaries, strict JSON, non-aliasing
normalization, snapshot states, candidate expiry, request/result identity,
metadata decimal presence, and subscription event variants. They use no live
venues, credentials, trading balances, or transactions.
