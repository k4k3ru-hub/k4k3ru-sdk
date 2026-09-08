package api

type ListCredentialParams struct {
	Page uint64 `json:"page"`
}

type ListCredentialCredential struct {
	ID                 uint64   `json:"id,string"`
	Name               string   `json:"name"`
	APIKey             string   `json:"apiKey"`
	Status             string   `json:"status"`
	SignatureAlgorithm *string  `json:"signatureAlgorithm"`
	Scopes             []string `json:"scopes"`
	ExpiresAt          *string  `json:"expiresAt"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
}

type ListCredentialResult struct {
	Credentials []ListCredentialCredential `json:"credentials"`
	Page        uint64                     `json:"page"`
	Limit       uint64                     `json:"limit"`
	Total       uint64                     `json:"total"`
	TotalPages  uint64                     `json:"totalPages"`
}
