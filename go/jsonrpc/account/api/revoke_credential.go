package api

type RevokeCredentialParams struct {
	CredentialID uint64 `json:"credentialId,string"`
}

type RevokeCredentialResult struct {
	CredentialID uint64 `json:"credentialId,string"`
	Status       string `json:"status"`
}
