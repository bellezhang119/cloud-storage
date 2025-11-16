package tests

import (
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

func TestGenerateVerificationToken(t *testing.T) {
	token1, err := util.GenerateVerificationToken()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if token1 == "" {
		t.Fatal("Expected non-empty token")
	}

	// Each byte is encoded as 2 hex chars, so 32 bytes -> 64 chars
	if len(token1) != 64 {
		t.Fatalf("Expected token length 64, got %d", len(token1))
	}

	token2, err := util.GenerateVerificationToken()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if token1 == token2 {
		t.Fatal("Expected two tokens to be different")
	}
}
