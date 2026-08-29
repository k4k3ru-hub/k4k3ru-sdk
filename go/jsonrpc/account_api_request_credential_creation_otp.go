package jsonrpc

type AccountAPIRequestCredentialCreationOTPParams struct {
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

type AccountAPIRequestCredentialCreationOTPResult struct {
	Purpose   string  `json:"purpose"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	ExpiresAt string  `json:"expiresAt"`
}
