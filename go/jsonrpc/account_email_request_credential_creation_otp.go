package jsonrpc

type AccountEmailRequestCredentialCreationOTPParams struct {
	Email string `json:"email"`
}

type AccountEmailRequestCredentialCreationOTPResult struct {
	Purpose   string `json:"purpose"`
	Email     string `json:"email"`
	ExpiresAt string `json:"expiresAt"`
}
