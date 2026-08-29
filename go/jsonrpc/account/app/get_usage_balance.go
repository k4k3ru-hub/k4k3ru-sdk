package app

type GetUsageBalanceParams struct {
	AccountID uint64 `json:"accountId,string"`
}

type GetUsageBalanceResult struct {
	AccountID    uint64 `json:"accountId,string"`
	BalanceTicks uint64 `json:"balanceTicks,string"`
}
