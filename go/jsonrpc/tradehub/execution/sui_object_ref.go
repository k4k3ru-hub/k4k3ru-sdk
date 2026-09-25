package execution

import (
	"strconv"
	"strings"

	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	sui "github.com/k4k3ru-hub/onchain/go/sui"
)

type SuiObjectRef struct {
	ObjectID string `json:"objectId"`
	Version  string `json:"version"`
	Digest   string `json:"digest"`
}

// Normalize returns a canonical object ID and decimal version without changing digest case.
//
// Version:
//   - 2026-09-24: Added.
func (r SuiObjectRef) Normalize() SuiObjectRef {
	r.ObjectID, r.Version, r.Digest = strings.TrimSpace(r.ObjectID), strings.TrimSpace(r.Version), strings.TrimSpace(r.Digest)
	if address, err := sui.ParseAddress(r.ObjectID); err == nil {
		r.ObjectID = address.String()
	}
	if version, err := strconv.ParseUint(r.Version, 10, 64); err == nil {
		r.Version = strconv.FormatUint(version, 10)
	}
	return r
}

// Validate checks a complete, nonzero owned object reference.
//
// Version:
//   - 2026-09-24: Added.
func (r SuiObjectRef) Validate() error {
	r = r.Normalize()
	address, err := sui.ParseAddress(r.ObjectID)
	if err != nil || address.IsZero() {
		return app.Tracef("failed to validate sui object reference: %w: object_id=invalid", app.InvalidParameter())
	}
	version, err := strconv.ParseUint(r.Version, 10, 64)
	if err != nil || version == 0 {
		return app.Tracef("failed to validate sui object reference: %w: version=invalid", app.InvalidParameter())
	}
	digest, err := sui.ParseObjectDigest(r.Digest)
	if err != nil || digest.IsZero() {
		return app.Tracef("failed to validate sui object reference: %w: digest=invalid", app.InvalidParameter())
	}
	return nil
}
