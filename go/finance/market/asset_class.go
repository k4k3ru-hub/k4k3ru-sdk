package market

type AssetClass string

const (
	AssetClassUnknown AssetClass = ""
	AssetClassCrypto  AssetClass = "crypto"
	AssetClassFX      AssetClass = "fx"
	AssetClassStock   AssetClass = "stock"
	AssetClassIndex   AssetClass = "index"
	AssetClassFund    AssetClass = "fund"
)
