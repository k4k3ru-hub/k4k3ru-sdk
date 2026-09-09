# Account API wire contracts

This package owns JSON parameters and results for API credential management.
`ListCredentialParams` and `RevokeCredentialParams` use the signing user's account;
neither accepts an account override. Credential IDs are decimal JSON strings.

Trusted Console services use `InternalListCredentialParams`,
`InternalRequestCredentialCreationOTPParams`, `InternalCreateCredentialParams` and
`InternalRevokeCredentialParams` for the corresponding `InternalApp.AccountAPI.*`
methods. Their `AccountID` must come from the verified server session, not browser
input. They require a dedicated internal credential with the exact scope
`console:api-keys:manage`. Results use the corresponding operation's existing result
structs in this package. These are wire models; transport and signing are injected
and composed by the consuming application, as for the existing contracts.

`CreateCredentialResult.SecretKey` is returned only when creating a key. Never log
or persist whole creation responses in browser storage.
