package aggregation

import (
	"errors"
	"reflect"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestAggregationSourceFilterNormalize(t *testing.T) {
	t.Parallel()

	filter := AggregationSourceFilter{
		VenueCategories: []VenueCategory{" DEX ", " CEX "},
		LiquidityModels: []LiquidityModel{" ORDER-BOOK ", " AMM "},
		AMMPoolChains:   []Chain{" ETHEREUM ", " BASE "},
	}
	normalized := filter.Normalize()
	if !reflect.DeepEqual(normalized.VenueCategories, []VenueCategory{VenueCategoryCEX, VenueCategoryDEX}) {
		t.Fatalf("VenueCategories = %#v", normalized.VenueCategories)
	}
	if !reflect.DeepEqual(normalized.LiquidityModels, []LiquidityModel{LiquidityModelAMM, LiquidityModelOrderBook}) {
		t.Fatalf("LiquidityModels = %#v", normalized.LiquidityModels)
	}
	if !reflect.DeepEqual(normalized.AMMPoolChains, []Chain{ChainBase, ChainEthereum}) {
		t.Fatalf("AMMPoolChains = %#v", normalized.AMMPoolChains)
	}
}

func TestAggregationSourceFilterValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []AggregationSourceFilter{
		{VenueCategories: []VenueCategory{}},
		{LiquidityModels: []LiquidityModel{}},
		{AMMPoolChains: []Chain{}},
		{VenueCategories: []VenueCategory{"broker"}},
		{VenueCategories: []VenueCategory{VenueCategoryCEX, VenueCategoryCEX}},
		{LiquidityModels: []LiquidityModel{"rfq"}},
		{LiquidityModels: []LiquidityModel{LiquidityModelAMM, LiquidityModelAMM}},
		{VenueCategories: []VenueCategory{VenueCategoryDEX}, LiquidityModels: []LiquidityModel{LiquidityModelOrderBook}, AMMPoolChains: []Chain{ChainBase}},
		{VenueCategories: []VenueCategory{VenueCategoryDEX}, LiquidityModels: []LiquidityModel{LiquidityModelAMM}, AMMPoolChains: []Chain{ChainNone}},
		{VenueCategories: []VenueCategory{VenueCategoryDEX}, LiquidityModels: []LiquidityModel{LiquidityModelAMM}, AMMPoolChains: []Chain{ChainBase, ChainBase}},
	}
	for i, filter := range tests {
		if err := filter.Validate(); err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
			t.Fatalf("Validate() error = %v, want invalid parameter for test_index=%d", err, i)
		}
	}
}

func TestAggregationSourceFilterValidateAcceptsDEXAMMChains(t *testing.T) {
	t.Parallel()

	filter := AggregationSourceFilter{
		VenueCategories: []VenueCategory{VenueCategoryDEX},
		LiquidityModels: []LiquidityModel{LiquidityModelAMM},
		AMMPoolChains:   []Chain{ChainEthereum, ChainBase, ChainBNB},
	}
	if err := filter.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
