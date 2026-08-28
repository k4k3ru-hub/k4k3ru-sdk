package jsonrpc

import (
	"fmt"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type Auth struct {
	APIKey    string `json:"apiKey,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// ValidateTimestamp validates the authentication timestamp against the current time.
//
// Parameters:
//   - maxAgeSec: maximum accepted timestamp age in seconds.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-28: Added.
func (a Auth) ValidateTimestamp(maxAgeSec int64) error {
	if a.Timestamp == 0 {
		return fmt.Errorf("failed to validate authentication timestamp: %w: timestamp=empty", apperror.InvalidParameter())
	}
	if maxAgeSec < 0 {
		return fmt.Errorf(
			"failed to validate authentication timestamp: %w: max_age_sec=out_of_range min_value=0",
			apperror.InvalidParameter(),
		)
	}

	nowUnix := time.Now().Unix()
	if a.Timestamp > nowUnix {
		return fmt.Errorf(
			"failed to validate authentication timestamp: %w: timestamp=invalid current_timestamp=%d",
			apperror.InvalidParameter(),
			nowUnix,
		)
	}
	if a.Timestamp < nowUnix-maxAgeSec {
		return fmt.Errorf(
			"failed to validate authentication timestamp: %w: timestamp=expired current_timestamp=%d max_age_sec=%d",
			apperror.Expired(),
			nowUnix,
			maxAgeSec,
		)
	}

	return nil
}
