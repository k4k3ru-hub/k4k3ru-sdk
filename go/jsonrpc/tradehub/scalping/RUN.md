# TradeHub.Scalping.Run

`RunParams` and `RunConfiguration` define the new entry/exit request contract.
The method identifier is `jsonrpc.MethodTradeHubScalpingRun`.
Import the types from `jsonrpc/tradehub/scalping`, quantities and market types
from `finance/market`, and TP/SL triggers from `jsonrpc/tradehub/executionrule`.

The SDK implements request validation, `module.Scalping().Run(ctx, params)`,
`RunResult` ACKs and typed `RunEvent` full snapshots. Construct the module with
`websocket.NewModule`; the composition root injects its transport, event registry
and connection lifecycle into the Run client. Use `UnsubscribeRun(ctx, handle)`
to stop monitoring without cancelling accepted orders.

The server connects authenticated Gateway forwarding, saved Run settings,
MarketHub observations and Spot OMS order restoration. Perpetual position-limit
enforcement, venue-position synchronization and Agent execution require separate
runtime integration; accepting a request in this SDK does not enforce its limit.
The existing Subscribe client and its `Params`/Result types remain separate
during migration; old settings are not converted into Run settings.

## Start and resume

In Go, construct `RunParams{IdempotencyKey: key, Params: &configuration}`.
The JSON is flat: `idempotencyKey`, `marketType`, `symbol`, optional `testMode`,
`observation`, `executionRule`. There is no nested `params` or `configuration`.

- `marketType`: `spot` or `perpetual`. `perp` is rejected.
- `symbol`: one canonical `BASE/QUOTE` pair; structural validation does not use a
  fixed symbol list or imply that a market is supported.
- `testMode`: omitted/false selects the production environment; true selects
  the venue's supported test environment. The SDK does not resolve networks.
- `idempotencyKey`: required for a new execution; 1–128 bytes after trimming.
- Resume is **only** `{"executionId":"saved-execution-id"}`. No configuration
  overrides, including `testMode:false` or null fields, are accepted. The server
  must restore persisted environment, accounts, defaults and order state.

`Normalize()` returns a detached copy; `Validate()` never mutates the caller's
data. Marshal and unmarshal normalize and validate automatically. Persisting a
`RunConfiguration` with `json.Marshal` materializes defaults and removes disabled
zero-return triggers. A failed decode leaves the destination unchanged.

## Observation and execution markets

`observation.markets` is a nonempty array of `MarketSelector`:

```json
{"venue":"cetus","chain":"sui","poolId":"OPTIONAL_POOL_ID"}
```

Only `venue` is required. `chain`, `poolId` and `venueSymbol` are optional;
`poolId` and `venueSymbol` are mutually exclusive. Omitted identifiers are
resolved using market type, canonical symbol, venue and environment. There is
no `network` field. Venue/chain names are normalized, while native identifier
case is preserved. Both market arrays allow up to 64 selectors.

`executionRule.markets` is required and contains `ExecutionMarket`, which adds
`accountAddress` and, for Perpetual, mandatory `perpetual` settings:

```json
{
  "venue":"hyperliquid",
  "venueSymbol":"SUI",
  "accountAddress":"YOUR_TRADING_ACCOUNT",
  "perpetual":{"leverage":3,"marginMode":"isolated"}
}
```

- `accountAddress` names the actual inventory/trading account, not necessarily
  the signing wallet. Agent profile resolution supplies it before sending.
- Each Perpetual execution market requires integer `leverage >= 1` and
  `marginMode: cross | isolated`. Spot rejects the `perpetual` object.
- Execution markets must be a subset of the **resolved** observation markets.
  The SDK rejects explicit venue/chain/identifier conflicts and identical
  duplicate selectors; it cannot prove inclusion when identifiers are omitted.
  TradeHub must enforce the final resolved subset, supported environment,
  account uniqueness, asset identity, permissions and venue-specific constraints.
- Catalog refresh policy during continuous operation is outside this request
  implementation and still requires its own review.
- One actual scope uses one trading account within a Run. Initial Perpetual
  management also excludes other Runs/external positions for the same actual
  account, venue, network and product. These checks require server/Agent state.
- Fee accounts for execution are derived from these trading accounts; this
  request has no independent `feeAccounts` override. Observation-only venues
  use the standard fee scope when no trading account applies.

## Quantities and conditions

All quantities use `market.Quantity`:

```json
{"amount":"100","decimals":1}
```

This represents **10 units** (`100 / 10^1`), independent of chain token decimals.
Both fields are required, including `decimals:0`. `amount` is an integer string;
`decimals` is 0–255. Maximum order quantities and specified observation quantities
must be positive; volume range bounds may be zero.

| Field | Unit for `SUI/USDC` |
| --- | --- |
| `observation.buy.quantity` | USDC input for observing a Buy |
| `observation.sell.quantity` | SUI input for observing a Sell |
| Spot Buy `entry.maximumQuantity` | Maximum USDC payment per initial order |
| Spot Sell `entry.maximumQuantity` | Maximum SUI sold per initial order |
| Perpetual Buy/Sell `entry.maximumQuantity` | Maximum SUI position quantity per initial order |
| Perpetual `executionRule.perpetual.maximumPositionQuantity` | Maximum aggregate absolute SUI position quantity across this Run's execution markets |

`observation.buy`/`sell` use MarketHub's `SideParams`. Omitted quantities cause
no quantity-based observation calculation. They are not inferred from order
caps. Leverage never multiplies `maximumQuantity`; actual order sizing still
requires inventory, reservations and venue metadata.

### Product-specific position and order limits

Spot retains `executionRule.maximumUnsettledOrders` (default 1). It rejects
`executionRule.perpetual`, even an empty object.

Perpetual uses a Run-wide `PerpetualRunSettings` object instead:

```json
{
  "perpetual": {
    "maximumPositionQuantity": {"amount":"30","decimals":0}
  }
}
```

This fragment belongs directly inside `executionRule`. The limit uses Base units
and applies to the sum of absolute venue position quantities across all execution
markets in this Run. Long and Short quantities do not offset, and leverage does
not multiply the limit. It is neither a per-venue allowance nor a margin amount.
`executionRule.markets[].perpetual` still contains only venue-specific leverage
and margin mode; placing the position limit there is invalid.

Omitting the Run-wide object or its quantity defaults the limit to
`entry.maximumQuantity`, preserving the exact amount and decimal scale. An
explicit quantity must be positive. A smaller limit than the per-order maximum
is valid because the latter is a cap, not a mandatory order size. Normalization
and saved JSON materialize the effective limit without changing the caller's data.

Perpetual rejects an explicit `maximumUnsettledOrders`, including zero, and does
not supply its Spot default. Previously saved Perpetual configurations containing
that field are invalid; no compatibility conversion or silent removal is made.

Runtime sizing must also reserve pending position increases, without counting
fills already reflected in venue positions twice. Unresolved submissions hold
new entries until reconciliation. Starting whole-position settlement stops
additional entries even when partial closes create quantity headroom. These are
runtime requirements, not computations performed by the SDK's validator.

For Spot exits, these fixed observation inputs are independent of settlement
estimates. TradeHub derives the full remaining Base quantity from each OMS order
and internally subscribes to MarketHub for that quantity. Omitting or changing
the observation Sell quantity does not disable those order-specific estimates.

Run does not expose an exact-input/exact-output selector. Execution follows the
order purpose: Spot entries and Buy-start settlement fix the input quantity;
Sell-start buyback fixes the remaining sold Base quantity derived from OMS.
The latter uses exact output internally, with a maximum Quote payment and a
refund of unused funds. Users do not supply a buyback quantity or `kind` in Run.
The common Swap preparation contract still carries `kind` internally; omitting
it there is not supported. Run observation quantities retain exact-input semantics;
non-default `kind` in a Run observation is rejected. TradeHub creates the
exact-output MarketHub request internally for Sell-start settlement.

`entry.condition` is required and contains at least one of these nine fields.
An optional `exit.condition` uses the same type. All specified metrics combine
with **AND**, inclusive of their bounds. A missing specified metric holds that
condition; a missing omitted metric has no effect.

| Field | Bounds |
| --- | --- |
| `priceChangeBps` | Signed decimal strings |
| `quoteVolume` | Nonnegative scaled `Quantity` values |
| `tradeCount` | Nonnegative integers |
| `buyVolumeRatioBps` | Decimal strings in 0..10000 |
| `priceChangeDeltaBps` | Signed decimal strings |
| `quoteVolumeChangeBps` | Signed decimal strings |
| `buyVolumeRatioDeltaBps` | Decimal strings in -10000..10000 |
| `realizedVolatilityBps` | Nonnegative decimal strings |
| `spreadBps` | Signed decimal strings |

Each range supplies `minimum`, `maximum`, or both. Both require `minimum <=
maximum`; comparisons are exact, including volume bounds with different scales.
Decimal strings allow fractional values, but no exponent, NaN or Infinity.
No common 10000 cap is imposed on signed changes, volatility or spread.

State-price metrics and trade-flow metrics retain distinct timestamps.
`maximumTradeAgeMs` limits trade age only when supplied;
`maximumSnapshotAgeMs` independently limits snapshot age. Omission does not
permit using disconnected observations or missing required metrics.

## Entry and exit

`executionRule.entry` requires `side: buy | sell`, `maximumQuantity`, and
`condition`. Optional `limitPrice` is a positive decimal price in Quote per Base:
Buy ceiling/Sell floor, **excluding trading fees**. Runtime evaluation must use
the corresponding fee-excluded market price, and execution must also respect
the limit. It is not a venue resting-order instruction.

`executionRule.exit` must contain at least one effective field after normalization:
`condition`, `takeProfit`, `stopLoss`, or `maximumHoldingMs`. These alternatives
combine with **OR**; metrics inside `condition` still combine with AND. Entry and
exit matching simultaneously prioritizes settlement over new orders.

TP/SL reuse `executionrule.Trigger`:

```json
{"type":"return_bps","value":"100"}
```

- `return_bps`: TP is positive, SL is negative, for either initial side.
  Valid decimal zeros (`"0"`, `"0.0"`, `"-0"`) disable that trigger and are omitted
  from normalized/persisted JSON. Invalid types or numeric syntax remain errors.
- `price`: strictly positive; zero is an error. Evaluation uses settlement-side
  **netPrice**, including trading fees, unlike the entry limit.
- Same-type thresholds require SL < TP for return bps and Buy price triggers;
  Sell price triggers require TP < SL. Equality is invalid.
- Mixed types are allowed without comparing their numeric values across units.
- Spot TP/SL refer to each initial order's weighted fill cost and remaining
  quantity, not the cost of all orders in the wallet. Sell-first Spot settles by
  buying back the sold Base quantity.
- Perpetual TP/SL refer to the venue position's current quantity and average
  entry price. Further entries update that shared basis rather than creating
  separate order-level exit allocations. Return bps uses entry notional, not
  margin ROE. Trading fees are included in exit estimates; gas and Funding are
  excluded. Reported Perpetual PnL comes from the venue, without constructing
  independent per-entry realized PnL. Position monitoring and Result integration
  are subsequent runtime work.

Spot `maximumHoldingMs` starts at that initial order's first valid fill timestamp.
Further fills do not extend it; downtime counts. Resume must restore the origin
from OMS. This is a settlement trigger, not a guaranteed fill deadline.
The SDK validates these settings; it does not implement monitoring or settlement.

## Defaults and validation

| Field | Omitted | Explicit range |
| --- | --- | --- |
| `observation.windowMs` | 60000 | 1..60000 |
| `observation.maximumTradeAgeMs` | No age cap | 1..D |
| `observation.maximumSnapshotAgeMs` | No age cap | 1..D |
| `executionRule.minimumOrderIntervalMs` | 1000 | 0..D |
| Spot `executionRule.maximumUnsettledOrders` | 1 | Positive uint64; forbidden for Perpetual |
| Perpetual `executionRule.perpetual.maximumPositionQuantity` | `entry.maximumQuantity` | Positive scaled Base quantity; Run-wide object forbidden for Spot |
| `executionRule.maximumSlippageBps` | 50 | 0..9999 |
| `executionRule.reserveBufferBps` | 100 | Nonnegative uint64 |
| `executionRule.executionTtlMs` | 30000 | 1..D |
| `executionRule.exit.maximumHoldingMs` | No holding timer | 1..D |

`D = 9223372036854` milliseconds, the largest whole millisecond duration that
fits Go `time.Duration`. Runtime timestamp arithmetic and venue-specific limits
still require validation. TTL controls the prepared execution's validity, not
the maximum holding period or a network response timeout.

Zero interval means no extra interval delay; zero buffer means no extra buffer,
not free margin/gas/fees. Purchase funds and buffers remain subject to the
agreed reservation rules. Null, unknown and duplicate JSON fields are rejected;
explicit out-of-range values are not clamped. JSON integer fields must be sent
exactly; callers should not round large counts through a floating-point number.

## Executable examples

These are request examples with **placeholder accounts**, not orders to send.
The quantities and thresholds illustrate the contract, not venue minimums or
profitable trading settings. Supported venues and test environments require
server integration.

- [Spot Buy](testdata/run_spot_buy.json): up to 10 USDC per purchase, with separate
  per-order exits and up to three unsettled initial orders.
- [Spot Sell](testdata/run_spot_sell.json): sell up to 10 SUI, then buy back the
  filled Base quantity when an exit alternative matches.
- [Perpetual Sell](testdata/run_perpetual_sell.json): short up to 10 SUI per order,
  with an explicit Run-wide 30 SUI position cap and 3x isolated settings. The
  leverage does not turn the order into 30 SUI or the position cap into 90 SUI.
- [Go construction example](run_example_test.go): owning-package imports,
  normalization, validation and reference-only resume.

Tests decode and round-trip all three JSON examples. No network or credentials
are needed to validate them.

A compiled [Go client example](../../../websocket/scalping_run_example_test.go)
shows composition, snapshot replacement and explicit unsubscribe.

## ACK and full replacement notifications

`RunResult` always contains `executionId`, `subscriptionKey` and `params` (the
normalized, saved `RunConfiguration`). A resume ACK also contains the original
environment and trading accounts. New requests persist the resolved market
scopes together with settings; resume restores those scopes without selecting
another network or account. Dynamic catalog refresh remains a separate stage.

`RunEvent` uses envelope type `scr`. It has `executionId`, `subscriptionKey`,
monotonically increasing `sequence`, and `kind: snapshot | error`.

A snapshot contains:

| Field | Meaning |
|---|---|
| `evaluationId`, `evaluatedAt` | Evaluation identity and calculation time, Unix ms |
| `marketType`, `symbol` | The saved Run instrument |
| `priceEvaluatedAt` | Optional confirmed end of state-price metrics, Unix ms |
| `metrics` | Consolidated MarketHub metrics, once per snapshot |
| `entry` | Entry evaluation and eligible market candidates |
| `orders` | Complete list of unsettled or uncertain initial OMS orders |

`RunEvaluation` has `status`, `markets` and optional `reasons`. Each market
contains the MarketHub `price`, condition `status`, optional `candidate`, and
(for exits) optional `trigger` and `returnBps`. Candidate expiry is optional:
it derives from explicit observation age limits, not the transaction TTL.
Candidates describe a signal, not a balance reservation or permission to submit.

`RunOrder` contains `orderId` (decimal string), OMS-derived `revision`,
`status: pending | holding | settling | unavailable`, the original `market`,
`accountAddress`, canonical entry `side`, and `exit`. Optional fields are
`remainingQuantity` (Base), `entryValue` (allocated Quote basis), and `acquiredAt`
(first valid fill, Unix ms). For Spot Buy, basis is fee-inclusive acquisition
cost; for Spot Sell it is net sale proceeds. Gas is excluded. The contract's
Perpetual entry value means remaining entry notional, not margin or ROE.
Perpetual restoration is not connected in this implementation stage.

For Spot Buy, `orders[].exit.markets[].price` is evaluated for that same order's
`remainingQuantity`. Bind it to the accompanying `orderId` and `revision`; never
combine an estimate from one snapshot with an order from another. A `vwap` net
receipt can supply the settlement minimum output after slippage; `reference` and
`fallback_reference` cannot. A changed amount withdraws the old estimate until the
replacement is available. A missing estimate does not imply zero holdings or PnL.
The wire fields are unchanged; deploy the corresponding server behavior before
an Agent that relies on this quantity contract.

For Spot Sell, the exit estimate fixes `netReceiveQuantity` to the order's
remaining sold Base quantity and supplies fee-inclusive `netPayQuantity` in Quote.
The return is `(entryValue - netPayQuantity) / entryValue * 10000`; `entryValue`
is that same order's remaining allocated sale proceeds. Agent rounds the maximum
payment up after slippage and refuses to exceed the existing per-order Quote
reservation. Neither side derives an amount by inverting `netPrice`.

Replace the entire prior snapshot. `orders: []` clears the active-order view;
closed orders are available through OMS/history rather than retained here.
A stream error invalidates prior actionable candidates. Disconnect, malformed
notifications and local buffer overflow surface on `Errors()`; resume explicitly
with the saved execution ID. Never infer a filled or cancelled order from a
missing connection or stream error. Unsubscribe and disconnect stop monitoring,
while saved settings and OMS records remain.

## Current server boundaries

- Nine entry/exit metric conditions use AND. Only requested metrics must exist.
  Exit condition, TP, SL and maximum holding time are alternatives; an eligible
  exit suppresses new entry candidates. Pending/uncertain OMS state also blocks entry.
- Spot order quantities come from effective OMS executions. Explicit
  `position_order_id` allocations reduce only their own initial order. Separate
  entry orders never share acquisition cost. Closed and zero-fill completed
  orders leave the snapshot; unresolved settlement remains `settling`.
- TP/SL price conditions use the corresponding net settlement-side price.
  Spot Buy return thresholds use an OMS-sized net receipt. The existing observation
  is reusable if its Sell input matches exactly; otherwise an internal MarketHub
  Run follows the required quantity. Equivalent quantity/market/fee inputs share
  an estimate within a Run, but order costs remain separate. Unneeded streams are
  released on quantity changes, settlement and Run cancellation; resume rebuilds
  them from OMS. Missing cost or an unavailable receipt holds return evaluation.
  Analysis metrics still come from the original observation scope. Both source
  and estimate timestamps respect the configured freshness limits; publishing a
  new Run snapshot does not refresh a cached estimate's age.
  Non-terminating allocated basis is omitted instead of rounded into a trigger.
- Sui Spot Buy and Sell execution use common Execution.Prepare/Submit with the
  Run ID and, for settlement, the initial order ID and OMS revision. Sell buyback
  uses exact output for Cetus, Turbos and Momentum. OMS acceptance compares the
  output Base quantity to the remaining sold quantity; confirmed input is actual
  payment after refunds. Per-order realized PnL is allocated sale proceeds minus
  actual buyback cost, including trading fees and excluding gas. Perpetual
  execution and position reconciliation remain subsequent work.
- Gross entry Limit comparison requires a gross venue price, which the current
  MarketHub price result does not expose. Such entries report
  `gross_price_unavailable`; net price is not substituted.
- Cross-venue exit candidates require the same verified Base/Quote asset IDs,
  chain/network and inventory account in the configured catalog. Cross-chain
  inventory allocation and currency conversion are not inferred from symbols.
- The server resolves only configured supported execution scopes. Request
  validation alone does not establish execution support. This change performs
  no Venue leverage changes, balance reservations, signing or submission.
