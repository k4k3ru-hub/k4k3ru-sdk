package signature

import (
	"encoding/hex"
	"testing"
)

func TestGenerateNonce(t *testing.T) {
	t.Parallel()

	first, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce() error = %v", err)
	}
	second, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce() second error = %v", err)
	}
	if first == second {
		t.Fatal("GenerateNonce() returned duplicate values")
	}

	decoded, err := hex.DecodeString(first)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	if len(decoded) != nonceSizeBytes {
		t.Fatalf("nonce size = %d, want %d", len(decoded), nonceSizeBytes)
	}
}
