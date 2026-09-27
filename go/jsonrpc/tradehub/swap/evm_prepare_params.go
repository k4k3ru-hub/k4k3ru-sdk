package swap

type EVMPrepareParams struct {
	// GasLimit is used unchanged for the returned swap or prerequisite approval.
	// It is required when simulation is disabled; no gas estimation runs then.
	GasLimit uint64 `json:"gasLimit"`
}
