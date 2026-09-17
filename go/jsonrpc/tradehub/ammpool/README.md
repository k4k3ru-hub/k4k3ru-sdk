# TradeHub AMM Pool list

`TradeHub.AMMPool.List` returns pools that TradeHub supports for Swap execution
and includes in its configured allowlist. It is distinct from the broader
MarketHub pool catalog. Being listed does not guarantee wallet balance,
allowance, liquidity, a successful quote, or successful execution.

Import the method from `jsonrpc` and the request/result types from
`jsonrpc/tradehub/ammpool`. As with other HTTP JSON-RPC methods in this SDK,
these types are used with the caller's HTTP transport and the shared envelopes.
This SDK addition does not itself install the server or Gateway handler.

## Request

`ListParams` accepts optional `chain`, `network`, and `venue` filters. Filters
are independently usable, matched exactly after trimming whitespace and
lowercasing. Supplied filters combine with AND. `{}` requests all eligible
pools; no pagination parameters are defined. Unknown filter values that have
valid syntax produce no matches. Unknown fields and non-object params are
rejected. Call `Normalize` and `Validate` before constructing an outbound request.

```json
{
  "id": "pools-1",
  "method": "TradeHub.AMMPool.List",
  "params": {"chain": "base", "network": "sepolia", "venue": "uniswap-v3"}
}
```

## Result

```json
{
  "pools": [{
    "chain": "base",
    "network": "sepolia",
    "venue": "uniswap-v3",
    "poolId": "0x94bfc0574ff48e92ce43d495376c477b1d0eeec0",
    "token0": {
      "assetId": "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
      "symbol": "USDC",
      "decimals": 6
    },
    "token1": {
      "assetId": "0x4200000000000000000000000000000000000006",
      "symbol": "WETH",
      "decimals": 18
    },
    "fee": 500
  }]
}
```

An empty result is `{"pools":[]}`. Result producers must initialize `Pools` to
an empty slice when there are no matches. Consumers must not rely on list order.

- Identify a pool by chain, network, venue, and `poolId`.
- `token0` / `token1` preserve the pool's on-chain ordering. Neither position
  implies USDC, base/quote currency, or the direction of a swap.
- `assetId` contains the EVM token address and maps to Swap's `tokenInAssetId`
  or `tokenOutAssetId` according to the selected direction.
- `fee` is an integer in millionths: `100` = 0.01%, `500` = 0.05%,
  `3000` = 0.3%, `10000` = 1%. `decimals` and `fee` are JSON numbers.
- Symbols and decimals come from metadata resolved at TradeHub startup.
  Fee metadata is not a live quote; dynamic venue fees can change.
- Signer configuration, secret names, and RPC endpoints are not exposed.

Authentication, routing, and deployment availability are server concerns;
this package does not imply that the endpoint allows anonymous access.
