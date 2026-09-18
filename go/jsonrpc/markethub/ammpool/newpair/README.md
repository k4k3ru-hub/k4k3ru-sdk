# MarketHub.AMMPool.NewPair

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
