package jsonrpc

type AccountEmailRequestSignInOTPParams struct {
	Email string `json:"email"`
}

type AccountEmailRequestSignInOTPResult struct {
	ExpiresAt string `json:"expiresAt"`
}
