package market

// DisplayName returns the venue brand name with version or deployment qualifiers.
// Unknown venues retain their identifier.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-22: Added.
func (v Venue) DisplayName() string {
	switch v {
	case Aerodrome:
		return "Aerodrome"
	case Arcus:
		return "Arcus"
	case Binance:
		return "Binance"
	case Bluefin:
		return "Bluefin"
	case BTSE:
		return "BTSE"
	case Bybit:
		return "Bybit"
	case Cetus:
		return "Cetus"
	case Coinbase:
		return "Coinbase"
	case DYDX:
		return "dYdX"
	case Hyperliquid:
		return "Hyperliquid"
	case Lighter:
		return "Lighter"
	case LighterRobinhood:
		return "Lighter (Robinhood Chain)"
	case Meteora:
		return "Meteora"
	case Momentum:
		return "Momentum"
	case OKX:
		return "OKX"
	case Raydium:
		return "Raydium"
	case Turbos:
		return "Turbos"
	case UniswapV3:
		return "Uniswap v3"
	case UniswapV4:
		return "Uniswap v4"
	default:
		return string(v)
	}
}
