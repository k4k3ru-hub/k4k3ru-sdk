package execution

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type SignedPayload struct {
	ChainFamily      ChainFamily     `json:"chainFamily"`
	Encoding         PayloadEncoding `json:"encoding"`
	TransactionBytes string          `json:"transactionBytes"`
	Signatures       []string        `json:"signatures,omitempty"`
}

type SubmitParams struct {
	OpenExecutionID string         `json:"openExecutionId,omitempty"`
	ExecutionID     string         `json:"executionId"`
	PayloadDigest   string         `json:"payloadDigest"`
	SignedPayload   *SignedPayload `json:"signedPayload,omitempty"`
}

// Normalize applies canonical formatting to execution submission parameters.
//
// Returns:
//   - Normalized parameters.
//
// Version:
//   - 2026-09-25: Normalize the optional full-close reference.
//   - 2026-09-10: Added.
func (p SubmitParams) Normalize() SubmitParams {
	p.OpenExecutionID = strings.TrimSpace(p.OpenExecutionID)
	p.ExecutionID = strings.TrimSpace(p.ExecutionID)
	p.PayloadDigest = strings.TrimSpace(p.PayloadDigest)
	if p.SignedPayload != nil {
		payload := *p.SignedPayload
		payload.ChainFamily = ChainFamily(strings.ToLower(strings.TrimSpace(string(payload.ChainFamily))))
		payload.Encoding = PayloadEncoding(strings.ToLower(strings.TrimSpace(string(payload.Encoding))))
		payload.TransactionBytes = strings.TrimSpace(payload.TransactionBytes)
		if payload.Signatures != nil {
			payload.Signatures = append([]string(nil), payload.Signatures...)
			for index := range payload.Signatures {
				payload.Signatures[index] = strings.TrimSpace(payload.Signatures[index])
			}
		}
		p.SignedPayload = &payload
	}
	return p
}

// ValidateReference validates the execution and payload reference returned by a prepare operation.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Validate the optional full-close reference.
//   - 2026-09-10: Added.
func (p SubmitParams) ValidateReference() error {
	p = p.Normalize()
	if p.OpenExecutionID != "" {
		if err := validateObservationID(p.OpenExecutionID, "open_execution_id"); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to validate trade hub execution submission parameters: %w", err)
		}
		if p.OpenExecutionID == p.ExecutionID {
			return invalidSubmitParameterError("open_execution_id=self")
		}
	}
	if p.ExecutionID == "" {
		return invalidSubmitParameterError("execution_id=empty")
	}
	if p.PayloadDigest == "" {
		return invalidSubmitParameterError("payload_digest=empty")
	}
	return nil
}

// Validate validates execution submission parameters, including the signed payload.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Limit explicit full-close references to Sui submissions.
//   - 2026-09-10: Added.
func (p SubmitParams) Validate() error {
	p = p.Normalize()
	if err := p.ValidateReference(); err != nil {
		return err
	}
	if p.SignedPayload == nil {
		return invalidSubmitParameterError("signed_payload=null")
	}
	if p.OpenExecutionID != "" && p.SignedPayload.ChainFamily != ChainFamilySui {
		return invalidSubmitParameterError("open_execution_id=unsupported")
	}
	if err := p.SignedPayload.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub execution submission parameters: %w", err)
	}
	return nil
}

// Validate validates a signed transaction payload.
// Sui initially accepts one serialized Ed25519 signature and at most 1 MiB of BCS.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Require bounded Sui payloads and an Ed25519 signature.
//   - 2026-09-10: Added.
func (p SignedPayload) Validate() error {
	p.ChainFamily = ChainFamily(strings.ToLower(strings.TrimSpace(string(p.ChainFamily))))
	p.Encoding = PayloadEncoding(strings.ToLower(strings.TrimSpace(string(p.Encoding))))
	p.TransactionBytes = strings.TrimSpace(p.TransactionBytes)

	switch p.ChainFamily {
	case ChainFamilyEVM:
		if p.Encoding != PayloadEncodingHex {
			return invalidSignedPayloadParameterError("encoding=invalid")
		}
		if !strings.HasPrefix(p.TransactionBytes, "0x") {
			return invalidSignedPayloadParameterError("transaction_bytes=invalid")
		}
		encoded := strings.TrimPrefix(p.TransactionBytes, "0x")
		if encoded == "" {
			return invalidSignedPayloadParameterError("transaction_bytes=empty")
		}
		if _, err := hex.DecodeString(encoded); err != nil {
			return invalidSignedPayloadParameterError("transaction_bytes=invalid")
		}
	case ChainFamilySui, ChainFamilySolana:
		if p.ChainFamily == ChainFamilySui {
			if len(p.TransactionBytes) > base64.StdEncoding.EncodedLen(1<<20) {
				return invalidSignedPayloadParameterError("transaction_bytes=too_long")
			}
			if len(p.Signatures) != 1 {
				return invalidSignedPayloadParameterError("signatures=invalid")
			}
			value := strings.TrimSpace(p.Signatures[0])
			if len(value) != base64.StdEncoding.EncodedLen(97) {
				return invalidSignedPayloadParameterError("signature=invalid")
			}
			raw, err := base64.StdEncoding.Strict().DecodeString(value)
			if err != nil || len(raw) != 97 || raw[0] != 0 {
				return invalidSignedPayloadParameterError("signature=invalid")
			}
		}
		if p.Encoding != PayloadEncodingBase64 {
			return invalidSignedPayloadParameterError("encoding=invalid")
		}
		if p.TransactionBytes == "" {
			return invalidSignedPayloadParameterError("transaction_bytes=empty")
		}
		if decoded, err := base64.StdEncoding.DecodeString(p.TransactionBytes); err != nil || len(decoded) == 0 {
			return invalidSignedPayloadParameterError("transaction_bytes=invalid")
		}
	default:
		return invalidSignedPayloadParameterError("chain_family=invalid")
	}

	for _, signature := range p.Signatures {
		if strings.TrimSpace(signature) == "" {
			return invalidSignedPayloadParameterError("signatures=invalid")
		}
	}
	return nil
}

// UnmarshalJSON decodes execution submission parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded parameters.
//
// Version:
//   - 2026-09-10: Added.
func (p *SubmitParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalidSubmitParameterError("destination=null")
	}
	type wireParams SubmitParams
	var decoded wireParams
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub execution submission parameters: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub execution submission parameters: %w", err)
	}
	*p = SubmitParams(decoded)
	return nil
}

func invalidSubmitParameterError(reason string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub execution submission parameters: %w: %s", k4k3ruSDKAppError.InvalidParameter(), reason)
}

func invalidSignedPayloadParameterError(reason string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate signed transaction payload: %w: %s", k4k3ruSDKAppError.InvalidParameter(), reason)
}
