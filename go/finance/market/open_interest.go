package market

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type OpenInterestUnit string
type OpenInterestPriceType string
type ContractSizeUnit string

const (
	OpenInterestUnitBaseAsset          OpenInterestUnit      = "base_asset"
	OpenInterestUnitContracts          OpenInterestUnit      = "contracts"
	OpenInterestUnitQuoteNotional      OpenInterestUnit      = "quote_notional"
	OpenInterestPriceTypeMark          OpenInterestPriceType = "mark"
	OpenInterestPriceTypeIndex         OpenInterestPriceType = "index"
	OpenInterestPriceTypeOracle        OpenInterestPriceType = "oracle"
	OpenInterestPriceTypeVenueReported OpenInterestPriceType = "venue_reported"
	ContractSizeUnitBaseAsset          ContractSizeUnit      = "base_asset"
	ContractSizeUnitQuoteNotional      ContractSizeUnit      = "quote_notional"
)

// OpenInterest represents one normalized perpetual Open Interest observation.
type OpenInterest struct {
	AssetClass               AssetClass
	Venue                    Venue
	MarketType               MarketType
	Symbol                   Symbol
	VenueSymbol              string
	EventTimestamp           time.Time
	ReceivedTimestamp        time.Time
	RawQuantity              float64
	RawUnit                  OpenInterestUnit
	Quantity                 float64
	NotionalValue            float64
	NotionalCurrency         string
	ConversionPrice          *float64
	ConversionPriceType      OpenInterestPriceType
	ConversionPriceTimestamp *time.Time
	ContractSize             *float64
	ContractSizeUnit         *ContractSizeUnit
	ContractSizeCurrency     *string
}

// Validate validates a normalized Open Interest observation.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-27: Added.
func (o OpenInterest) Validate() error {
	if o.AssetClass == AssetClassUnknown || !o.Venue.IsValid() || o.MarketType != MarketTypePerpetual {
		return fmt.Errorf("failed to validate open interest: market_identity=invalid")
	}
	if err := o.Symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate open interest: %w", err)
	}
	if strings.TrimSpace(o.VenueSymbol) == "" {
		return fmt.Errorf("failed to validate open interest: venue_symbol=empty")
	}
	if o.EventTimestamp.IsZero() || o.ReceivedTimestamp.IsZero() {
		return fmt.Errorf("failed to validate open interest: timestamp=empty")
	}
	if (o.RawUnit != OpenInterestUnitBaseAsset && o.RawUnit != OpenInterestUnitContracts && o.RawUnit != OpenInterestUnitQuoteNotional) || invalidOpenInterestNumber(o.RawQuantity) || invalidOpenInterestNumber(o.Quantity) || invalidOpenInterestNumber(o.NotionalValue) {
		return fmt.Errorf("failed to validate open interest: quantity=invalid")
	}
	if strings.TrimSpace(o.NotionalCurrency) == "" {
		return fmt.Errorf("failed to validate open interest: notional_currency=empty")
	}
	if o.ConversionPrice == nil || *o.ConversionPrice <= 0 || math.IsNaN(*o.ConversionPrice) || math.IsInf(*o.ConversionPrice, 0) || (o.ConversionPriceType != OpenInterestPriceTypeMark && o.ConversionPriceType != OpenInterestPriceTypeIndex && o.ConversionPriceType != OpenInterestPriceTypeOracle && o.ConversionPriceType != OpenInterestPriceTypeVenueReported) || o.ConversionPriceTimestamp == nil || o.ConversionPriceTimestamp.IsZero() {
		return fmt.Errorf("failed to validate open interest: conversion_price=invalid")
	}
	if o.RawUnit == OpenInterestUnitContracts && (o.ContractSize == nil || o.ContractSizeUnit == nil || o.ContractSizeCurrency == nil) {
		return fmt.Errorf("failed to validate open interest: contract_size=null")
	}
	return nil
}

func invalidOpenInterestNumber(value float64) bool {
	return value < 0 || math.IsNaN(value) || math.IsInf(value, 0)
}
