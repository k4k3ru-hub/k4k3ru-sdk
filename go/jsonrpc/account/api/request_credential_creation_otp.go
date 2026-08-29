package api

type RequestCredentialCreationOTPParams struct {
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

type RequestCredentialCreationOTPResult struct {
	Purpose   string  `json:"purpose"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	ExpiresAt string  `json:"expiresAt"`
}
