package execution

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestSubmitParamsValidate(t *testing.T) {
	t.Parallel()

	valid := SubmitParams{
		ExecutionID:   "execution-1",
		PayloadDigest: "0xdigest",
		SignedPayload: &SignedPayload{
			ChainFamily:      ChainFamilyEVM,
			Encoding:         PayloadEncodingHex,
			TransactionBytes: "0x02aabb",
		},
	}

	tests := []struct {
		name       string
		params     SubmitParams
		wantReason string
	}{
		{name: "valid evm", params: valid},
		{name: "missing execution id", params: func() SubmitParams { value := valid; value.ExecutionID = ""; return value }(), wantReason: "execution_id=empty"},
		{name: "missing digest", params: func() SubmitParams { value := valid; value.PayloadDigest = ""; return value }(), wantReason: "payload_digest=empty"},
		{name: "missing payload", params: SubmitParams{ExecutionID: "execution-1", PayloadDigest: "0xdigest"}, wantReason: "signed_payload=null"},
		{name: "wrong evm encoding", params: func() SubmitParams {
			value := valid
			payload := *value.SignedPayload
			payload.Encoding = PayloadEncodingBase64
			value.SignedPayload = &payload
			return value
		}(), wantReason: "encoding=invalid"},
		{name: "invalid evm bytes", params: func() SubmitParams {
			value := valid
			payload := *value.SignedPayload
			payload.TransactionBytes = "0xzz"
			value.SignedPayload = &payload
			return value
		}(), wantReason: "transaction_bytes=invalid"},
		{name: "unsupported family", params: func() SubmitParams {
			value := valid
			payload := *value.SignedPayload
			payload.ChainFamily = "bitcoin"
			value.SignedPayload = &payload
			return value
		}(), wantReason: "chain_family=invalid"},
		{name: "empty detached signature", params: func() SubmitParams {
			value := valid
			payload := *value.SignedPayload
			payload.Signatures = []string{" "}
			value.SignedPayload = &payload
			return value
		}(), wantReason: "signatures=invalid"},
		{
			name: "valid sui",
			params: SubmitParams{
				ExecutionID:   "execution-2",
				PayloadDigest: "digest",
				SignedPayload: &SignedPayload{
					ChainFamily:      ChainFamilySui,
					Encoding:         PayloadEncodingBase64,
					TransactionBytes: base64.StdEncoding.EncodeToString([]byte{1, 2, 3}),
					Signatures:       []string{"signature"},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.params.Validate()
			if test.wantReason == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantReason) {
				t.Fatalf("Validate() error = %v, want reason %q", err, test.wantReason)
			}
		})
	}
}

func TestSubmitParamsValidateReference(t *testing.T) {
	t.Parallel()

	params := SubmitParams{ExecutionID: " execution-1 ", PayloadDigest: " digest "}
	if err := params.ValidateReference(); err != nil {
		t.Fatalf("ValidateReference() error = %v", err)
	}
	if err := params.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing signed payload error")
	}
}

func TestSubmitParamsUnmarshalJSON(t *testing.T) {
	t.Parallel()

	var params SubmitParams
	if err := json.Unmarshal([]byte(`{"executionId":"execution-1","payloadDigest":"digest","signedPayload":{"chainFamily":"evm","encoding":"hex","transactionBytes":"0x02","unknown":true}}`), &params); err == nil {
		t.Fatal("Unmarshal() error = nil, want unknown field error")
	}
	if err := json.Unmarshal([]byte(`{"executionId":"execution-1","payloadDigest":"digest"} {}`), &params); err == nil {
		t.Fatal("Unmarshal() error = nil, want trailing value error")
	}
}
