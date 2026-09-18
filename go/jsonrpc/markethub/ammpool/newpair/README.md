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
    // Replace the previous list with result.Pairs; absence removes a pair.
    _ = result
}
```

`ListParams{Filter: filter, Limit: 100}` is the List request payload. Get uses
`GetParams{Chain, Network, Venue, PoolID}`. Timestamps are Unix microseconds.
Optional filters are chain, network and venue; empty values mean all.
The server controls the lifecycle window (default 24 hours), using first liquidity
when observed and pool creation otherwise. Read `FirstLiquidityAt`, `FirstSwapAt`
and `LiquidityUSD` to evaluate activity. The removed age, swap and USD threshold
request fields are rejected; deploy matching server and client versions together.

Token `id` may represent an EVM contract, native currency, Solana mint, or Sui coin
type. Pool identifiers preserve case. Position number/index strings and optional
chain-specific details support blocks, slots and checkpoints without loss of
integer precision. These types do not imply an adapter is deployed for every chain.

WebSocket event type: `apnp`. Each event is a bounded replacement snapshot; inspect
`truncated` and use List pagination for larger result sets. Subscription identity
includes all normalized filters. Unsubscribe with the owning subscription client.
