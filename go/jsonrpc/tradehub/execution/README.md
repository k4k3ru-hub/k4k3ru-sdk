# Execution observation

`TradeHub.Execution.Subscribe` and `TradeHub.Execution.Unsubscribe` observe an
execution previously submitted through TradeHub. They require authenticated
WebSocket requests. Supported executions are one Base Mainnet/Sepolia EVM
transaction (ERC-20 approval or Uniswap V3 swap), or one configured Cetus swap on
Sui Testnet.

Subscribe with `{"executionId":"exec_..."}`. The acknowledgement returns
`executionId` and an opaque `subscriptionKey`. Notifications use event type `exs`
and the `SubscriptionEvent` type in this package. Unsubscribe with both identifiers.

- `pending`: the chain-specific completion condition has not been recorded.
- `success`: EVM receipt status is 1, or the Sui transaction succeeded and was
  included in a checkpoint, with its actual fill reconciled into OMS.
- `failed`: EVM receipt status is 0 (`transaction_reverted`), or the checkpointed
  Sui transaction failed (`transaction_failed`).
- `kind: "error"`: observation was interrupted, with sanitized code
  `observation_unavailable`. This is not a failed transaction.

EVM completion means block inclusion, not finality. Sui completion requires
checkpoint inclusion. The server releases a terminal watch but keeps the
connection open. A fresh subscription rechecks EVM receipts or reads the committed
Sui OMS snapshot. Sui reconciliation continues independently of subscriptions and
resumes from persisted pending submissions after a server restart.
Unknown or other-account executions return `not_found`. Unsupported chain/leg
shapes return `unsupported`; executions without persisted submission identity
return `invalid_parameter`. No additional observation credit charge is configured.

The composed WebSocket SDK exposes `module.Execution()`. Register through
`Subscribe(ctx, execution.SubscribeParams{ExecutionID: id})`, consume `Events()`
and `Errors()`, and explicitly call `Unsubscribe` if stopping before completion.
Duplicate active SDK calls reuse the same local handle. Terminal events close
the event channel; transport/delivery interruptions are reported on `Errors()`.
Treat interruption separately from the terminal status. The SDK does not
automatically resubscribe or send transactions.

Gateway closes the session after five minutes without application traffic.
Protocol heartbeat traffic does not keep it alive. Other application traffic on
the same connection does. A pending transaction can therefore outlive its watch;
resubscribe explicitly to obtain a fresh snapshot. Call `module.Close()` when
the caller is finished with the connection.

## Sui Submit

Use `SubmitParams` with the original Prepare `executionId` and `payloadDigest`.
Set `SignedPayload{ChainFamily: ChainFamilySui, Encoding: PayloadEncodingBase64,
TransactionBytes: preparedBytes, Signatures: []string{signatureBase64}}`.
`TransactionBytes` is the unchanged full TransactionData BCS in base64;
`signatureBase64` encodes the 97-byte Ed25519 `flag || signature || publicKey`.
The server checks the signature, ownership, exact prepared bytes and gas owner.
The signing intent digest and the returned base58 onchain `TransactionID` differ.

The initial relay supports configured Cetus swaps on Sui Testnet with sender-paid
gas. Prepare writes Execution only; first valid Submit writes an OMS order and a
submitted/pending record before broadcast. Quantities in the request are atomic
integer strings; OMS order quantity uses input-token decimals. Submit acceptance
does not mean a successful swap or a fill. Retry an uncertain Submit with identical params, including
after Prepare TTL if a claim was committed; do not rebuild or spend reserved coins
until the onchain result has been reconciled.

## Sui OMS snapshots and explicit Close

`ExecutionSnapshot.Onchain.Checkpoint` identifies checkpoint inclusion; EVM block
fields are absent for Sui. `ExecutionSnapshot.OMS` contains the string `orderId`,
optional `openExecutionId`, actual `fill`, separate `fee`, and `pnl`. Amounts in
these public fields are exact integer strings in the specified token's smallest
units. The corresponding OMS records use exact token-denominated decimal strings.

For a full Close, set `SubmitParams.OpenExecutionID` to the successful Open's
execution ID on the first Submit. Keep it unchanged on every retry. The server
requires the same account, chain/network, reversed token pair and wallet path,
and the Open's entire acquired amount as Close input. A locked OMS parent order
prevents concurrent Close orders from claiming the same Open. A checkpointed
failed Close releases that claim for a new Close execution. Partial Close and
cross-chain inventory matching are not implemented in this Sui endpoint.

Successful checkpoint reconciliation records actual input/output quantities from
the verified Cetus event and balance changes. Failed transactions have no fill.
Both record `fee.amount = computationCost + storageCost - storageRebate` in MIST
(`fee.assetId` is SUI, `fee.decimals` is 9), including a negative amount when rebates
exceed costs. No quote-currency conversion is performed.

Only the successful explicitly linked Close returns `pnl.status: "realized"`:
`Close.amountOut - Open.amountIn`, denominated in the original input token and
excluding gas. DEX fees are already included in actual amounts. All other snapshots
return `pnl.status: "unavailable"` with no numeric PnL, rather than assuming zero.
