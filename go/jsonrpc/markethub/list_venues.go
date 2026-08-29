package markethub

type ListVenuesParams struct{}

type ListVenuesVenue struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
}

type ListVenuesResult struct {
	Venues []ListVenuesVenue `json:"venues"`
}
