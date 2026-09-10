# TradeHub Swap JSON-RPC

`TradeHub.Swap.Quote` returns a current AMM quote without preparing a
transaction. `TradeHub.Swap.Prepare` validates allowance and prepares exactly
one unsigned transaction for client signing.

Prepare returns one of two statuses:

- `ready`: the signing payload is the requested swap transaction.
- `approval-required`: the signing payload is an ERC-20 approval prerequisite.

For `approval-required`, clients submit the signed approval through
`TradeHub.Execution.Submit`, wait for completion, and call
`TradeHub.Swap.Prepare` again. The next successful preparation returns `ready`.
The client sends an `approvalAmount` selected from its local execution policy;
TradeHub resolves and validates the spender rather than accepting it from the
client.

Every signing payload is bound to `submitParams.executionId` and
`submitParams.payloadDigest`. Private keys and signed transactions are not part
of Prepare parameters.

After local signing, the client adds `signedPayload` to the returned submit
reference and sends it as the parameters of `TradeHub.Execution.Submit`.
EVM signed transaction bytes use `hex`; Sui and Solana transaction bytes use
`base64`. Detached signatures can be supplied through `signatures` when the
chain's submission protocol requires them.

```json
{
  "executionId": "execution-1",
  "payloadDigest": "0xdigest",
  "signedPayload": {
    "chainFamily": "evm",
    "encoding": "hex",
    "transactionBytes": "0x02f8..."
  }
}
```
