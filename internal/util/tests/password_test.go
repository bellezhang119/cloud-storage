package tests

import (
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "mySuperSecret123!"

	// Test hashing password
	hashed, err := util.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hashed == "" {
		t.Fatalf("HashPassword returned empty string")
	}

	// Test checking correct password
	err = util.CheckPassword(hashed, password)
	if err != nil {
		t.Errorf("CheckPassword failed on correct password: %v", err)
	}

	// Test checking incorrect password
	err = util.CheckPassword(hashed, "wrongPassword")
	if err == nil {
		t.Errorf("CheckPassword did not return error on incorrect password")
	}
}
