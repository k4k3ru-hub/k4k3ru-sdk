package scalping

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// TestScalpingPerpetualContract verifies canonical values and rejects former perp settings.
//
// Version:
//   - 2026-09-25: Added.
func TestScalpingPerpetualContract(t *testing.T) {
	p := perpParams()
	data, err := json.Marshal(p)
	if err != nil || !strings.Contains(string(data), `"marketType":"perpetual"`) {
		t.Fatalf("new request did not use perpetual: %s %v", data, err)
	}
	var decoded Params
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.MarketType != market.MarketTypePerpetual {
		t.Fatalf("canonical request rejected: %v", err)
	}
	legacy := strings.Replace(string(data), `"marketType":"perpetual"`, `"marketType":"perp"`, 1)
	if err := json.Unmarshal([]byte(legacy), &decoded); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatalf("legacy request accepted: %v", err)
	}
	subscribe := `{"idempotencyKey":"key",` + legacy[1:]
	var request SubscribeParams
	if err := json.Unmarshal([]byte(subscribe), &request); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatalf("legacy subscription accepted: %v", err)
	}
	for _, value := range []market.MarketType{"perp", market.MarketTypeFuture} {
		p.MarketType = value
		if err := p.Normalize().Validate(); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("unsupported request market type accepted: %q %v", value, err)
		}
	}
	result := matchedResult()
	result.MarketType = "perp"
	if err := result.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatalf("legacy result accepted: %v", err)
	}
	data, err = json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot Result
	if err := json.Unmarshal(data, &snapshot); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatalf("legacy result JSON accepted: %v", err)
	}
}
