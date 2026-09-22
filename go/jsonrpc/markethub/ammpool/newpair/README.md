# MarketHub.AMMPool.NewPair

`Pair.LPPrincipal` optionally returns Token0/Token1 principal amounts as exact
decimal strings in whole-token units. `AmountPercentage` compares those quantities
without price weighting: 1 token versus 99,999 tokens gives 0.001% versus 99.999%.
Percentages use 0–100, unlike the fractional rates in `Fees`. Token0 is truncated
to 18 decimal places; Token1 is its complement to 100. Both percentages are nil
when the total amount is zero. A rounded 0% does not imply an exactly zero amount.
Principal covers all price ranges, excluding uncollected fees and direct transfers;
it is distinct from active liquidity, USD value and LP protection.
`EvaluatedAt` (Unix microseconds) and `Position` describe the adopted LP observation
and do not change on USD-reference-only updates. Missing/unverified LP state returns
nil; older JSON without the field remains readable. `CloneLPPrincipal` copies an
observation for independent retention. Quantity imbalance is not a listing filter.

Methods: `MarketHub.AMMPool.NewPair.List`, `.Get`, `.Subscribe`, `.Unsubscribe`.
Use this group for pool discovery. The former Launch RPC group and its SDK types have been removed.

```go
import newpair "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/newpair"

filter := newpair.Params{Chain: "base"}
// ws is a websocket.Module constructed with websocket.NewModule.
subscription, err := ws.AMMPoolNewPair().Subscribe(ctx, filter)
if err != nil {
    return err
}
for result := range subscription.Events() {
    // Read result.Pairs and result.ExcludedPairs.
    // With truncated=true, absence from Pairs does not prove exclusion.
    _ = result
}
```

`ListParams{Filter: filter, Limit: 100}` is the List request payload. Get uses
`GetParams{Chain, Network, Venue, PoolID}`. Timestamps are Unix microseconds.
Optional filters are chain, network and venue; empty values mean all.
The new server contract uses pool creation time for the 24-hour lifecycle, a
confirmed observed swap, liquidity of at least 1,000 USD, and a synchronized LP state with a successful
valuation. `SwapObservedAt` is not necessarily the first swap in pool history;
null means no swap has been observed, not that no swap ever occurred.
`SwapObservedPosition` identifies the observed event; `ConfirmedAt` is nullable
until its confirmation completes. `LiquidityEvaluatedAt` records the last
successful evaluation. All nullable timestamps are Unix microseconds.

`LiquidityUSD` keeps its Finding representation (decimal string value or unknown).
`LiquidityMethod` describes the valuation basis. The agreed valuation uses LP
principal across all price ranges, excluding uncollected fees/direct transfers,
with 1 USDC = 1 USD as a conversion assumption. FDV/MarketCap are not included.
A failed refresh or stale evaluation must not be rewritten as zero liquidity.
Capture retry budgets are server configuration, not request parameters. The removed age, swap and USD threshold request fields remain rejected.

Token `id` may represent an EVM contract, native currency, Solana mint, or Sui coin
type. Pool identifiers preserve case. Position number/index strings and optional
chain-specific details support blocks, slots and checkpoints without loss of
integer precision. These types do not imply an adapter is deployed for every chain.

WebSocket event type: `apnp`. Each event is a bounded replacement snapshot; inspect
`truncated` and use List pagination for larger result sets. Subscription identity
includes all normalized filters. Unsubscribe with the owning subscription client.

## Listing and exclusions

`Result.Pairs` contains listed pairs only (`isListed=true`). Subscribe snapshots
also carry `Result.ExcludedPairs`: the last full data for previously listed pools
that left the listing (`isListed=false`, `exclusionReason` set). Previously unlisted
candidates need not be exposed. Get/List use the same listing conditions; the
exclusion list supplements Subscribe. Exclusion reasons include `age_exceeded`,
`liquidity_below_minimum`, `liquidity_unavailable`, `liquidity_stale`,
`swap_unconfirmed`, and `creation_reverted`. Treat reasons as extensible strings.

For `liquidity_stale`, the last value and evaluation time remain available. A pair
that qualifies again returns to `pairs` without an exclusion reason. SDK routing
preserves `excludedPairs` when replacing an unread snapshot with a newer snapshot.
This is latest-state delivery, not a durable history of every transition.
Exclusion retention limits remain a server implementation detail to be finalized;
do not assume every exclusion can be replayed. When `truncated=true`, absence from
`pairs` does not prove exclusion. Resynchronize on epoch changes or reconnects.

## Compatibility and rollout

This is a breaking type/wire update: `firstLiquidityAt`, `firstSwapAt` and
`backfillAbandonedAt` are removed rather than aliased to fields with different
meanings. New nullable evidence fields serialize as JSON null when unknown.
`isListed` always serializes; no listing inference is made from an older response
that omits it. Deploy matching server and client versions together.

This SDK change defines and decodes the new contract. It does not implement the
server's listing policy, valuation worker, exclusion retention or notification
production; those service changes follow separately.

### Live LP synchronization

`lpStateStatus` is `syncing`, `synced` or `unavailable`. A missing/unknown status
must not be interpreted as synced. `lpStatePosition` identifies the last established
LP state, including an optional transaction/event position; it is not proof that
all subsequent blocks were checked. Synchronization is process-local and must be
re-established after a restart.

`liquidityEvaluatedAt` is the time of the last adopted USD calculation. It is no
longer the LP-state block timestamp. There is no fifteen-minute expiry: quiet
synchronized pools remain eligible until their creation-based 24-hour limit.
Disconnects or detected gaps withdraw listing with `lp_state_unavailable`;
initial/recovery capture uses `lp_state_syncing`. Missing reference prices use
`usd_reference_unavailable`. Last amounts/times remain available in excludedPairs.


## Cumulative activity

`Pair.activity` is optional for compatibility with older servers. It contains
observed cumulative activity throughout the monitoring lifetime, including time
before listing and temporary exclusion. It has no completeness or gap status.
Missing history is not inferred as zero and no full-history guarantee is made.

- `startedAt` / `updatedAt`: aggregation start / last incorporated activity update,
  in Unix microseconds. `startedAt` is not a claim that recovered events began then.
- `swapCount`: total Swap event count, as a decimal string.
- `token0ToToken1` / `token1ToToken0`: `count`, `token0Amount`, `token1Amount`,
  `volumeUsd`. A zero-amount Swap is included only in `swapCount`.
- `liquidityAdded` / `liquidityRemoved`: `count`, `token0Amount`, `token1Amount`.
- `netToken0Liquidity` / `netToken1Liquidity`: additions minus removals.

Quantities use token units and normalized decimal strings, without float64.
Unavailable quantities are JSON null. V4 LP quantities are null after a liquidity
change because ModifyLiquidity does not emit token amounts. LP reductions refer
to principal removed from positions, not subsequent Collect transfers; fees and
Donate events are excluded. Swap amounts are pool-level, not final user receipts.

USD volume uses one available shared reference leg per swap and excludes gas.
The existing 1 USDC = 1 USD convention applies. Values are fixed at observation
and truncated to 18 decimals. A directional USD volume is null if any of its
counted swaps could not be priced. Recovery does not use current spot prices for
old trades; only directly configured USDC amounts can be priced during recovery.

HTTP List/Get and the `apnp` stream use the same Activity type. Request parameters,
listing conditions and other existing response fields remain unchanged.

## Pool swap fee observations

`Pair.Fees` is optional for compatibility with older servers. Current MarketHub
listing requires both directional rates. `rate` is a decimal fraction (`"0.003"`
means 0.3%); nil does not mean zero. Fees exclude token taxes, gas and price impact.
Variable fees are reference values for `Position` and `ReferenceSender` at
`ObservedAt` (Unix microseconds), not next-order guarantees. No fee status or
freshness TTL is supplied. Use `CloneFees` when retaining independently mutable
snapshots. `fees_unavailable` is a possible exclusion reason.

## Token tax observations

`Pair.TokenTaxes` is nullable and contains independently nullable `token0` and
`token1` observations. A missing field in an older response and explicit JSON null
both decode as nil. Each observation contains nullable `buyRate`, `sellRate`,
`canChange`, `hasExemptions`, plus `source`, `observedAt` and `position`.
Rates are decimal fractions: `"0.01"` means 1%; `"0"` is confirmed zero and differs
from null. False is a confirmed negative finding and differs from null as well.
`observedAt` uses Unix microseconds; `position` identifies the evaluated block.
`CloneTokenTaxes` / `CloneTokenTax` detach pointer fields for retained snapshots.

Initial server analysis covers reviewed code-only tax-free models on Base mainnet
and native currency. The onchain-defined Base USDC address is trusted and skipped:
its observation is null, not an inferred zero-tax verdict. An unknown, unsupported,
or failed analysis also remains null, without excluding an otherwise listed pool.
Analysis runs asynchronously, so a later `apnp` replacement snapshot may contain
observations missing in an earlier List/Get/Subscribe result.

`source` is `contract_analysis` or `native_currency`. These findings describe token
tax rules at the observed block; they do not prove tradability, LP protection,
owner renouncement, or future fees. Pool swap fees, token taxes, gas and price
impact remain separate fields/responsibilities. There is no new request parameter,
status enum, or subscription operation for this additive response field.
