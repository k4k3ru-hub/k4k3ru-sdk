package newpair

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"

	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const MaxQueryBytes = 4096
const (
	CodeQueryTooLarge app.Code = "new_pair_query_too_large"
	CodeQueryBusy     app.Code = "new_pair_query_busy"
)

type Query struct {
	ActivityPeriod string          `json:"activityPeriod"`
	Conditions     QueryConditions `json:"conditions"`
	Sort           QuerySort       `json:"sort"`
}
type QuerySort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// Declaration order is the canonical ASCII order of the JSON property names.
type QueryConditions struct {
	MaxLiquidityRemovedCount      string `json:"maxLiquidityRemovedCount,omitempty"`
	MaxPoolFeeRate                string `json:"maxPoolFeeRate,omitempty"`
	MaxTokenBuyTaxRate            string `json:"maxTokenBuyTaxRate,omitempty"`
	MaxTokenSellTaxRate           string `json:"maxTokenSellTaxRate,omitempty"`
	MinLiquidityAddedCount        string `json:"minLiquidityAddedCount,omitempty"`
	MinLiquidityUSD               string `json:"minLiquidityUsd,omitempty"`
	MinSwapCount                  string `json:"minSwapCount,omitempty"`
	MinSwapCountChangePercentage  string `json:"minSwapCountChangePercentage,omitempty"`
	MinUniqueSenderCount          string `json:"minUniqueSenderCount,omitempty"`
	MinVolumeUSD                  string `json:"minVolumeUsd,omitempty"`
	MinVolumeUSDChangePercentage  string `json:"minVolumeUsdChangePercentage,omitempty"`
	RequireAllPositionsProtected  bool   `json:"requireAllPositionsProtected,omitempty"`
	RequireNoForcedBalanceChanges bool   `json:"requireNoForcedBalanceChanges,omitempty"`
	RequireNoMinting              bool   `json:"requireNoMinting,omitempty"`
	RequireNoProtectionWeakening  bool   `json:"requireNoProtectionWeakening,omitempty"`
	RequireNoTaxChanges           bool   `json:"requireNoTaxChanges,omitempty"`
	RequireNoTransferRestrictions bool   `json:"requireNoTransferRestrictions,omitempty"`
	RequireNoUpgrade              bool   `json:"requireNoUpgrade,omitempty"`
}

type QueryResult struct {
	MatchedCount string `json:"matchedCount"`
	ExpiresAt    *int64 `json:"expiresAt"`
}

type queryNumber struct {
	name  string
	value *string
	kind  string
}

func (c *QueryConditions) numbers() []queryNumber {
	return []queryNumber{
		{"maxLiquidityRemovedCount", &c.MaxLiquidityRemovedCount, "count"},
		{"maxPoolFeeRate", &c.MaxPoolFeeRate, "rate"},
		{"maxTokenBuyTaxRate", &c.MaxTokenBuyTaxRate, "rate"},
		{"maxTokenSellTaxRate", &c.MaxTokenSellTaxRate, "rate"},
		{"minLiquidityAddedCount", &c.MinLiquidityAddedCount, "count"},
		{"minLiquidityUsd", &c.MinLiquidityUSD, "amount"},
		{"minSwapCount", &c.MinSwapCount, "count"},
		{"minSwapCountChangePercentage", &c.MinSwapCountChangePercentage, "change"},
		{"minUniqueSenderCount", &c.MinUniqueSenderCount, "count"},
		{"minVolumeUsd", &c.MinVolumeUSD, "amount"},
		{"minVolumeUsdChangePercentage", &c.MinVolumeUSDChangePercentage, "change"},
	}
}

func queryDecimal(s, kind string) (string, bool) {
	if s == "" || len(s) > 98 {
		return "", false
	}
	negative := strings.HasPrefix(s, "-")
	if negative && kind != "change" {
		return "", false
	}
	digits := strings.TrimPrefix(s, "-")
	parts := strings.Split(digits, ".")
	if len(parts) > 2 || len(parts[0]) == 0 || len(parts[0]) > 78 || (kind == "count" && len(parts) > 1) {
		return "", false
	}
	if len(parts) == 2 && (len(parts[1]) == 0 || len(parts[1]) > 18) {
		return "", false
	}
	for _, p := range parts {
		for _, v := range p {
			if v < '0' || v > '9' {
				return "", false
			}
		}
	}
	n, ok := new(big.Rat).SetString(s)
	if !ok || (kind == "change" && n.Cmp(big.NewRat(-100, 1)) < 0) || (kind == "rate" && n.Cmp(big.NewRat(1, 1)) > 0) {
		return "", false
	}
	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	if len(parts) == 2 {
		if fraction := strings.TrimRight(parts[1], "0"); fraction != "" {
			whole += "." + fraction
		}
	}
	if negative && n.Sign() != 0 {
		whole = "-" + whole
	}
	return whole, true
}

// Normalize copies a query and supplies defaults without inventing thresholds.
//
// Version:
//   - 2026-09-29: Added.
func (q Query) Normalize() Query {
	if q.ActivityPeriod == "" {
		q.ActivityPeriod = "5m"
	}
	if q.Sort.Field == "" {
		q.Sort.Field = "poolCreatedAt"
	}
	if q.Sort.Direction == "" {
		q.Sort.Direction = "desc"
	}
	for _, f := range q.Conditions.numbers() {
		if v, ok := queryDecimal(*f.value, f.kind); ok {
			*f.value = v
		}
	}
	return q
}

// Validate checks bounded decimal conditions and supported period/sort combinations.
//
// Version:
//   - 2026-09-29: Added.
func (q Query) Validate() error {
	for _, f := range q.Conditions.numbers() {
		if *f.value != "" {
			if _, ok := queryDecimal(*f.value, f.kind); !ok {
				return fmt.Errorf("failed to validate new pair query: %w: condition=%q value=invalid", app.InvalidParameter(), f.name)
			}
		}
	}
	q = q.Normalize()
	switch q.ActivityPeriod {
	case "5m", "15m", "1h", "24h":
	default:
		return fmt.Errorf("failed to validate new pair query: %w: activity_period=invalid", app.InvalidParameter())
	}
	switch q.Sort.Field {
	case "poolCreatedAt", "liquidityUsd", "swapCount", "volumeUsd", "uniqueSenderCount", "swapCountChangePercentage", "volumeUsdChangePercentage":
	default:
		return fmt.Errorf("failed to validate new pair query: %w: sort_field=invalid", app.InvalidParameter())
	}
	if q.Sort.Direction != "asc" && q.Sort.Direction != "desc" {
		return fmt.Errorf("failed to validate new pair query: %w: sort_direction=invalid", app.InvalidParameter())
	}
	if q.ActivityPeriod == "1h" || q.ActivityPeriod == "24h" {
		c := q.Conditions
		if c.MinUniqueSenderCount != "" || c.MinSwapCountChangePercentage != "" || c.MinVolumeUSDChangePercentage != "" || q.Sort.Field == "uniqueSenderCount" || q.Sort.Field == "swapCountChangePercentage" || q.Sort.Field == "volumeUsdChangePercentage" {
			return fmt.Errorf("failed to validate new pair query: unsupported period combination: %w", app.InvalidParameter())
		}
	}
	raw, err := json.Marshal(q)
	if err != nil {
		return fmt.Errorf("failed to validate new pair query: %w", err)
	}
	if len(raw) > MaxQueryBytes {
		return fmt.Errorf("failed to validate new pair query: %w: query=too_long", app.InvalidParameter())
	}
	return nil
}

// Digest returns the versioned digest of the normalized canonical query JSON.
//
// Version:
//   - 2026-09-29: Added.
func (q Query) Digest() (string, error) {
	if err := q.Validate(); err != nil {
		return "", fmt.Errorf("failed to identify new pair query: %w", err)
	}
	raw, err := json.Marshal(q.Normalize())
	if err != nil {
		return "", fmt.Errorf("failed to encode new pair query: %w", err)
	}
	sum := sha256.Sum256(append([]byte("newpair-query-v1\n"), raw...))
	return hex.EncodeToString(sum[:]), nil
}

// UnmarshalJSON rejects nulls, duplicate properties and invalid query values.
//
// Version:
//   - 2026-09-29: Added.
func (q *Query) UnmarshalJSON(data []byte) error {
	if q == nil {
		return fmt.Errorf("failed to decode new pair query: destination=null")
	}
	if len(data) > MaxQueryBytes {
		return fmt.Errorf("failed to decode new pair query: %w: query=too_long", app.InvalidParameter())
	}
	d := json.NewDecoder(bytes.NewReader(data))
	if err := queryObject(d); err != nil {
		return fmt.Errorf("failed to decode new pair query: %w: %w", app.InvalidParameter(), err)
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("failed to decode new pair query: %w: json=invalid", app.InvalidParameter())
	}
	type wire Query
	var v wire
	if err := decode(data, &v); err != nil {
		return fmt.Errorf("failed to decode new pair query: %w: %w", app.InvalidParameter(), err)
	}
	if err := Query(v).Validate(); err != nil {
		return err
	}
	*q = Query(v).Normalize()
	return nil
}

func queryObject(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return fmt.Errorf("failed to read query object: %w", err)
	}
	if token != json.Delim('{') {
		return fmt.Errorf("failed to read query object: object=invalid")
	}
	return queryMembers(d, "")
}
func queryMembers(d *json.Decoder, path string) error {
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return fmt.Errorf("failed to read query key: %w", err)
		}
		name, ok := token.(string)
		if !ok || seen[name] || !queryFieldAllowed(path, name) {
			return fmt.Errorf("failed to read query key: key=invalid")
		}
		seen[name] = true
		value, err := d.Token()
		if err != nil {
			return fmt.Errorf("failed to read query value: %w", err)
		}
		if value == nil || value == "" {
			return fmt.Errorf("failed to read query value: value=empty")
		}
		if delim, ok := value.(json.Delim); ok {
			if delim != '{' {
				return fmt.Errorf("failed to read query value: value=invalid")
			}
			if err := queryMembers(d, name); err != nil {
				return err
			}
		}
	}
	token, err := d.Token()
	if err != nil {
		return fmt.Errorf("failed to close query object: %w", err)
	}
	if token != json.Delim('}') {
		return fmt.Errorf("failed to close query object: object=invalid")
	}
	return nil
}

// Go's JSON decoder accepts case-insensitive aliases; the query wire contract does not.
func queryFieldAllowed(path, name string) bool {
	switch path {
	case "":
		return name == "activityPeriod" || name == "conditions" || name == "sort"
	case "sort":
		return name == "field" || name == "direction"
	case "conditions":
		var c QueryConditions
		for _, n := range c.numbers() {
			if n.name == name {
				return true
			}
		}
		switch name {
		case "requireAllPositionsProtected", "requireNoForcedBalanceChanges", "requireNoMinting", "requireNoProtectionWeakening", "requireNoTaxChanges", "requireNoTransferRestrictions", "requireNoUpgrade":
			return true
		}
	}
	return false
}
