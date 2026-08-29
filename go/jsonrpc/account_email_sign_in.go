package jsonrpc

type AccountEmailSignInParams struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type AccountEmailSignInResult struct {
	AccountID uint64 `json:"accountId,string"`
}
