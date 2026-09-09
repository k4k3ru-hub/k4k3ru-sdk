package api

// Internal parameters are for a trusted Console service, never a browser-selected account.
// Results use the corresponding public operation result types in this package.
type InternalListCredentialParams struct {
	AccountID uint64 `json:"accountId,string"`
	ListCredentialParams
}
type InternalRequestCredentialCreationOTPParams struct {
	AccountID uint64 `json:"accountId,string"`
	Email     string `json:"email"`
}
type InternalCreateCredentialParams struct {
	AccountID uint64 `json:"accountId,string"`
	CreateCredentialParams
}
type InternalRevokeCredentialParams struct {
	AccountID uint64 `json:"accountId,string"`
	RevokeCredentialParams
}
