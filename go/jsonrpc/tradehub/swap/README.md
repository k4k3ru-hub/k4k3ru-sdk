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
