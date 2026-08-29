package jsonrpc

type AccountAPICreateCredentialParams struct {
	Email              *string `json:"email,omitempty"`
	Phone              *string `json:"phone,omitempty"`
	Code               string  `json:"code"`
	APIName            string  `json:"apiName"`
	SignatureAlgorithm string  `json:"signatureAlgorithm"`
	ExpiresIn          string  `json:"expiresIn"`
}

type AccountAPICreateCredentialResult struct {
	AccountID          uint64 `json:"accountId,string"`
	APIName            string `json:"apiName"`
	APIKey             string `json:"apiKey"`
	SignatureAlgorithm string `json:"signatureAlgorithm"`
	SecretKey          string `json:"secretKey"`
	ExpiresAt          string `json:"expiresAt"`
}
