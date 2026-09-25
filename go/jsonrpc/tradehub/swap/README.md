# TradeHub Swap JSON-RPC

`TradeHub.AMMPool.Swap.Quote` returns a current AMM quote without preparing a
transaction. `TradeHub.AMMPool.Swap.Prepare` validates wallet funding, simulates,
and prepares one unsigned transaction for client signing. EVM funding uses ERC-20
allowance; Sui uses explicit owned Coin references.

Prepare returns one of two statuses:

- `ready`: the signing payload is the requested swap transaction.
- `approval-required`: the signing payload is an ERC-20 approval prerequisite.

For `approval-required`, clients submit the signed approval through
`TradeHub.Execution.Submit`, wait for completion, and call
`TradeHub.AMMPool.Swap.Prepare` again. The next successful preparation returns `ready`.
For EVM, the client sends an `approvalAmount` selected from its local execution policy;
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

## Sui Testnet / Cetus Prepare

Initial support is exact-input SUI ↔ Circle Testnet USDC in an explicitly
allowlisted pool. Add `sui` to `PrepareParams` and omit `approvalAmount` and
`stateReference`. EVM requests retain their existing approval requirement and
reject `sui`. Exact-output Sui preparation and checkpoint pinning are unsupported.

```json
{
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
funding parameters.

- For SUI input, omit `inputCoins`; the swap splits the input amount from GasCoin.
  Selected gas coins must cover **amount + gasBudget**.
- For USDC input, supply `sui.inputCoins` with the same reference shape. These coins
  must cover the input amount; separate SUI gas coins must cover `gasBudget`.
- All coins must belong to `signer`. Duplicate or overlapping object IDs are
  rejected. Each list is limited to 256 entries. No implicit coin replacement,
  sponsorship, or funding reservation is performed.
- The result uses `ready`, `chainFamily="sui"`, `encoding="base64"`, complete
  unsigned BCS `TransactionData`, and a `0x`-prefixed hex intent signing digest.
  `amountOut` is the full-transaction simulation output, excluding gas even when
  the recipient receives SUI. There is no approval transaction.
- The transaction enforces minimum output and full input consumption. Gas price
  uses the network reference price; expiration is the current epoch. `expiresAt`
  is an application admission deadline, not a millisecond chain expiration.
- The request and signing result are persisted as Execution snapshots. Prepare
  creates no OMS order. Identical idempotent retries return the original payload;
  changed coin versions require a new key. Callers must coordinate wallet funds
  and recheck references before signing.

This change implements Prepare only. The server's Sui `Execution.Submit`, receipt
observation, OMS integration, and Agent signing workflow are separate steps;
the returned submit reference does not imply Sui submission is already enabled.
