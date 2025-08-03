package tests

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func TestGenerateJWTTokensAndVerify(t *testing.T) {
	userID := int32(123)
	email := "user@example.com"
	refreshExpiry := time.Now().Add(24 * time.Hour)

	accessToken, refreshToken, err := util.GenerateJWTTokens(userID, email, refreshExpiry)
	if err != nil {
		t.Fatalf("GenerateJWTTokens failed: %v", err)
	}

	// Verify Access Token
	accessClaims, err := util.VerifyAccessToken(accessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken failed: %v", err)
	}
	if accessClaims["user_id"].(float64) != float64(userID) {
		t.Errorf("Access token user_id mismatch: got %v want %v", accessClaims["user_id"], userID)
	}
	if accessClaims["email"].(string) != email {
		t.Errorf("Access token email mismatch: got %v want %v", accessClaims["email"], email)
	}

	// Verify Refresh Token
	refreshClaims, err := util.VerifyRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("VerifyRefreshToken failed: %v", err)
	}
	if refreshClaims["user_id"].(float64) != float64(userID) {
		t.Errorf("Refresh token user_id mismatch: got %v want %v", refreshClaims["user_id"], userID)
	}
	if refreshClaims["email"].(string) != email {
		t.Errorf("Refresh token email mismatch: got %v want %v", refreshClaims["email"], email)
	}
	expUnix := int64(refreshClaims["exp"].(float64))
	if expUnix != refreshExpiry.Unix() {
		t.Errorf("Refresh token expiry mismatch: got %v want %v", expUnix, refreshExpiry.Unix())
	}
}

func TestVerifyAccessToken_InvalidToken(t *testing.T) {
	_, err := util.VerifyAccessToken("invalid.token")
	if err == nil {
		t.Error("Expected error for invalid access token, got nil")
	}
}

func TestVerifyRefreshToken_ExpiredToken(t *testing.T) {
	userID := int32(123)
	email := "user@example.com"
	expiry := time.Now().Add(-1 * time.Hour) // expired

	_, refreshToken, err := util.GenerateJWTTokens(userID, email, expiry)
	if err != nil {
		t.Fatalf("GenerateJWTTokens failed: %v", err)
	}

	_, err = util.VerifyRefreshToken(refreshToken)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Errorf("Expected error containing 'expired', got %v", err)
	}
}

func TestHashAndCheckToken(t *testing.T) {
	token := "some_refresh_token_string"
	hashed := util.HashToken(token)

	if err := util.CheckToken(hashed, token); err != nil {
		t.Errorf("CheckToken failed for matching token: %v", err)
	}

	if err := util.CheckToken(hashed, "different_token"); err == nil {
		t.Error("CheckToken did not fail for mismatched token")
	}
}
