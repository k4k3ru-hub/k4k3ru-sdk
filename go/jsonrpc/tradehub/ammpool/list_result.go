package ammpool

type ListResult struct {
	Pools []PoolMetadata `json:"pools"`
}

// PoolMetadata describes a swap-supported, allowlisted pool, not a guarantee
// that a swap can execute with the caller's balance or current liquidity.
type PoolMetadata struct {
	Chain   string        `json:"chain"`
	Network string        `json:"network"`
	Venue   string        `json:"venue"`
	PoolID  string        `json:"poolId"`
	Token0  TokenMetadata `json:"token0"`
	Token1  TokenMetadata `json:"token1"`
	// Fee is expressed in millionths: 500 means 0.05%, 3000 means 0.3%.
	Fee uint32 `json:"fee"`
}

// TokenMetadata identifies an on-chain token; symbols are display metadata.
// AssetID is the ERC20 contract address for EVM pools.
type TokenMetadata struct {
	AssetID  string `json:"assetId"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}
