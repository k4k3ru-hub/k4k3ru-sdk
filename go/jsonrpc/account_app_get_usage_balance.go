package jsonrpc

type AccountAppGetUsageBalanceParams struct {
	AccountID uint64 `json:"accountId,string"`
}

type AccountAppGetUsageBalanceResult struct {
	AccountID    uint64 `json:"accountId,string"`
	BalanceTicks uint64 `json:"balanceTicks,string"`
}
