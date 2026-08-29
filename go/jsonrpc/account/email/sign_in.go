package email

type SignInParams struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type SignInResult struct {
	AccountID uint64 `json:"accountId,string"`
}
