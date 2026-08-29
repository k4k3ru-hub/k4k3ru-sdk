package email

type RequestSignInOTPParams struct {
	Email string `json:"email"`
}

type RequestSignInOTPResult struct {
	ExpiresAt string `json:"expiresAt"`
}
