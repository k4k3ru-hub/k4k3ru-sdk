# TradeHub Swap JSON-RPC

Clients call `TradeHub.Execution.Prepare` using the owning
`execution/prepare.Params` type with `kind: "swap"` and this package's
`PrepareParams` in `swap`. `TradeHub.AMMPool.Swap.Prepare` and its method constant
were removed in the 2026-09-29 source update. See
[common execution](../execution/README.md).

`TradeHub.AMMPool.Swap.Quote` returns a current AMM quote without preparing a
transaction. `TradeHub.Execution.Prepare` with `kind: "swap"` validates wallet funding and prepares
one unsigned transaction for client signing, with optional simulation. EVM funding uses ERC-20
allowance; Sui uses explicit owned Coin references.

`simulate` defaults to **false**. This deliberately changes the previous always-simulated
Prepare behavior. Set `simulate: true` explicitly to retain quoting and simulation.

| Parameter | `simulate: false` (default) | `simulate: true` |
| --- | --- | --- |
| `amountLimit` | Required positive atomic-unit integer string | Optional additional constraint |
| `maximumSlippageBps` | Omit; without a quote there is no slippage baseline | Required |
| `evm.gasLimit` | Required positive integer for EVM | Optional; estimated when omitted |
| `sui.gasBudget` | Required for Sui | Required for Sui |
| `stateReference` | Omit | Existing EVM quote-state comparison; unsupported on Sui |

For exact input, `amountLimit` is the minimum received **output** quantity. For
exact output, it is the maximum paid **input** quantity. With simulation enabled,
the stricter of this explicit limit and the quote-derived slippage limit is used.
With simulation disabled, the supplied limit and gas limit are used unchanged;
missing values are errors. Gross MarketHub values are never implicitly substituted.

No Quote, swap execution simulation, or gas estimation runs when `simulate` is false.
Metadata, allowance, Coin, epoch, nonce, and fee reads may still run. For EVM,
read-only contract calls used to fetch pool metadata or allowance still use `eth_call`.
The gas limit applies to the transaction returned, including an approval prerequisite.
Simulation-enabled preparation fails on a simulation error; it does not silently
fall back to an unsimulated transaction. No independent Simulate RPC is added.

The common Prepare response wraps this package's `PrepareResult` in `swap`:
`{kind: "swap", swap: {...}}`. That `swap` result includes `simulated` and, for a
ready swap, the enforced `amountLimit`.
For unsimulated exact input only `amountIn` is known; `amountOut` is omitted.
For unsimulated exact output only `amountOut` is known; `amountIn` is omitted.
When `simulated` is true both swap quantities are available. `ready` means the
signing payload is constructed; it does not by itself assert simulation success.

Prepare returns one of two statuses:

- `ready`: the signing payload is the requested swap transaction.
- `approval-required`: the signing payload is an ERC-20 approval prerequisite.

For `approval-required`, clients submit the signed approval through
`TradeHub.Execution.Submit`, wait for completion, and call
`TradeHub.Execution.Prepare` again with `kind: "swap"`. The next successful
preparation returns `swap.status: "ready"`.
For EVM, the client sends an `approvalAmount` selected from its local execution policy;
TradeHub resolves and validates the spender rather than accepting it from the
client.

Every signing payload is bound to `swap.submitParams.executionId` and
`swap.submitParams.payloadDigest` in the common response. Private keys and signed transactions are not part
of Prepare parameters.

After local signing, the client adds `signedPayload` to the returned submit
reference and sends it as the parameters of `TradeHub.Execution.Submit`.
EVM signed transaction bytes use `hex`; Sui and Solana transaction bytes use
`base64`. Detached signatures can be supplied through `signatures` when the
chain's submission protocol requires them.

```json
{
  "preparedToken": "<unchanged token returned by Prepare>",
  "executionId": "execution-1",
  "payloadDigest": "0xdigest",
  "signedPayload": {
    "chainFamily": "evm",
    "encoding": "hex",
    "transactionBytes": "0x02f8..."
  }
}
```

## Sui Testnet / Cetus Prepare

Initial support is exact-input SUI ↔ Circle Testnet USDC in an explicitly
allowlisted pool. Add `sui` to `PrepareParams` and omit `approvalAmount` and
`stateReference`. EVM requests retain their existing approval requirement and
reject `sui`. Exact-output Sui preparation and checkpoint pinning are unsupported.

```json
{
  "simulate": true,
  "chain": "sui",
  "network": "testnet",
  "venue": "cetus",
  "poolId": "0x64a908fd79e89e05b85aa5de00f73250c67318c2c8578c0cc31a1c212257fa6b",
  "tokenInAssetId": "0x2::sui::SUI",
  "tokenOutAssetId": "0xa1ec7fc00a6f40db9693ad1415d0c193ad3906494428cf252621037bd7117e29::usdc::USDC",
  "amount": "1000000",
  "kind": "exact-input",
  "maximumSlippageBps": 100,
  "signer": "<wallet address>",
  "recipient": "<recipient address>",
  "executionTtlMs": 30000,
  "idempotencyKey": "sui-swap-unique-key",
  "sui": {
    "gasPayment": [{"objectId": "<gas coin ID>", "version": "<u64 decimal version>", "digest": "<base58 digest>"}],
    "gasBudget": "50000000"
  }
}
```

Replace placeholders with current wallet-owned references. Amounts and gas budget
are atomic-unit decimal strings fitting u64. Object versions are decimal strings
to preserve values above JavaScript's safe integer range; digests are case-sensitive.
`execution.SuiObjectRef` owns the reference type; `swap.SuiPrepareParams` owns
funding parameters. The JSON example above is the inner `swap` value; wrap it in
`{"kind":"swap","swap":{...}}` when calling `TradeHub.Execution.Prepare`.

- For SUI input, omit `inputCoins`; the swap splits the input amount from GasCoin.
  Selected gas coins must cover **amount + gasBudget**.
- For USDC input, supply `sui.inputCoins` with the same reference shape. These coins
  must cover the input amount; separate SUI gas coins must cover `gasBudget`.
- All coins must belong to `signer`. Duplicate or overlapping object IDs are
  rejected. Each list is limited to 256 entries. No implicit coin replacement,
  sponsorship, or funding reservation is performed.
- The result uses `ready`, `chainFamily="sui"`, `encoding="base64"`, complete
  unsigned BCS `TransactionData`, and a `0x`-prefixed hex intent signing digest.
  When `simulated` is true, `amountOut` is the full-transaction simulation output,
  excluding gas even when the recipient receives SUI. Otherwise it is omitted.
  There is no approval transaction.
- The transaction enforces minimum output and full input consumption. Gas price
  uses the network reference price; expiration is the current epoch. `expiresAt`
  is an application admission deadline, not a millisecond chain expiration.
- Prepare does not persist its request, signing result, or an OMS order. Forward
  its `preparedToken` unchanged with the signed submission. First Submit records
  the OMS order before broadcast. Callers coordinate wallet funds, recheck
  references before signing, and retain the same signed submission for retries.

To prepare the same Sui request without simulation, omit `simulate` or set it to
false, remove `maximumSlippageBps`, and supply `amountLimit` in output-token atomic
units. Keep `sui.gasBudget` and owned Coin references. The resulting unsigned
transaction still enforces that explicit minimum output.
