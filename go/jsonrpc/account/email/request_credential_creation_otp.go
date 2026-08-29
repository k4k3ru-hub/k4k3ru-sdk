package email

type RequestCredentialCreationOTPParams struct {
	Email string `json:"email"`
}

type RequestCredentialCreationOTPResult struct {
	Purpose   string `json:"purpose"`
	Email     string `json:"email"`
	ExpiresAt string `json:"expiresAt"`
}
