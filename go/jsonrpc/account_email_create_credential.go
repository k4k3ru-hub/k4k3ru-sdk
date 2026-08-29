package jsonrpc

type AccountEmailCreateCredentialParams struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type AccountEmailCreateCredentialResult struct {
	AccountID  uint64 `json:"accountId,string"`
	Status     string `json:"status"`
	Email      string `json:"email"`
	OTPPurpose string `json:"otpPurpose"`
	BonusTicks uint64 `json:"bonusTicks"`
}
