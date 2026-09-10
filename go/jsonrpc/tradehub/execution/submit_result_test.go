package execution

import (
	"strings"
	"testing"
)

func TestSubmitResultValidate(t *testing.T) {
	t.Parallel()

	valid := SubmitResult{ExecutionID: "exec_1", ChainFamily: ChainFamilyEVM, TransactionID: "0x1", SubmittedAt: 1}
	tests := []struct {
		name       string
		result     SubmitResult
		wantReason string
	}{
		{name: "valid", result: valid},
		{name: "empty execution id", result: func() SubmitResult { value := valid; value.ExecutionID = ""; return value }(), wantReason: "execution_id=empty"},
		{name: "invalid chain family", result: func() SubmitResult { value := valid; value.ChainFamily = ChainFamilyUnknown; return value }(), wantReason: "chain_family=invalid"},
		{name: "empty transaction id", result: func() SubmitResult { value := valid; value.TransactionID = ""; return value }(), wantReason: "transaction_id=empty"},
		{name: "invalid submitted at", result: func() SubmitResult { value := valid; value.SubmittedAt = 0; return value }(), wantReason: "submitted_at=out_of_range"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.result.Validate()
			if test.wantReason == "" && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if test.wantReason != "" && (err == nil || !strings.Contains(err.Error(), test.wantReason)) {
				t.Fatalf("Validate() error = %v, want reason %q", err, test.wantReason)
			}
		})
	}
}
