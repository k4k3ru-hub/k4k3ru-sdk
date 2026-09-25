# Execution observation

`TradeHub.Execution.Subscribe` and `TradeHub.Execution.Unsubscribe` observe an
execution previously submitted through TradeHub. They require authenticated
WebSocket requests. The initial implementation supports one Base Mainnet/Sepolia
EVM transaction per execution, including ERC-20 approval and Uniswap V3 swap.

Subscribe with `{"executionId":"exec_..."}`. The acknowledgement returns
`executionId` and an opaque `subscriptionKey`. Notifications use event type `exs`
and the `SubscriptionEvent` type in this package. Unsubscribe with both identifiers.

- `pending`: no receipt yet; this is not transaction failure.
- `success`: receipt status is 1.
- `failed`: receipt status is 0, with failure code `transaction_reverted`.
- `kind: "error"`: observation was interrupted, with sanitized code
  `observation_unavailable`. This is not a failed transaction.

Completion means block inclusion, not finality. The server releases a terminal
watch but keeps the connection open. A fresh subscription rechecks the chain.
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
does not mean a successful swap or a fill. Sui observation and fill/PnL accounting
are not yet connected. Retry an uncertain Submit with identical params, including
after Prepare TTL if a claim was committed; do not rebuild or spend reserved coins
until the onchain result has been reconciled.
