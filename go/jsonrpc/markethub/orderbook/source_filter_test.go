package orderbook

import (
	"errors"
	"reflect"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestSourceFilterNormalizeDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	categories := []VenueCategory{" DEX ", " CEX "}
	filter := SourceFilter{VenueCategories: categories, LiquidityModels: []LiquidityModel{" ORDER-BOOK ", " AMM "}}
	got := filter.Normalize()
	if !reflect.DeepEqual(got.VenueCategories, []VenueCategory{VenueCategoryCEX, VenueCategoryDEX}) || !reflect.DeepEqual(got.LiquidityModels, []LiquidityModel{LiquidityModelAMM, LiquidityModelOrderBook}) {
		t.Fatalf("Normalize() = %#v", got)
	}
	if !reflect.DeepEqual(categories, []VenueCategory{" DEX ", " CEX "}) {
		t.Fatalf("Normalize() mutated input = %#v", categories)
	}
}

func TestSourceFilterValidateRejectsInvalidAMMChainSelection(t *testing.T) {
	t.Parallel()

	filter := SourceFilter{VenueCategories: []VenueCategory{VenueCategoryCEX}, LiquidityModels: []LiquidityModel{LiquidityModelOrderBook}, AMMPoolChains: []Chain{ChainBase}}
	if err := filter.Validate(); err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("Validate() error = %v, want invalid parameter", err)
	}
}
