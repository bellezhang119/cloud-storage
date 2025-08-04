package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_Success(t *testing.T) {
	mockVerifier := func(tokenStr string) (jwt.MapClaims, error) {
		return jwt.MapClaims{
			"user_id": float64(123), // JWT stores numbers as float64
			"email":   "test@example.com",
		}, nil
	}

	middleware := auth.AuthMiddleware(mockVerifier)

	// Dummy handler to verify context values
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(auth.GetUserIDKey())
		email := r.Context().Value(auth.GetUserEmailKey())

		assert.Equal(t, int32(123), userID)
		assert.Equal(t, "test@example.com", email)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer validtoken")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	middleware := auth.AuthMiddleware(func(string) (jwt.MapClaims, error) {
		return nil, nil // Won't be called
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Missing or invalid Authorization header")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	middleware := auth.AuthMiddleware(func(token string) (jwt.MapClaims, error) {
		return nil, jwt.ErrTokenMalformed
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid or expired token")
}

func TestAuthMiddleware_InvalidPayload(t *testing.T) {
	middleware := auth.AuthMiddleware(func(tokenStr string) (jwt.MapClaims, error) {
		return jwt.MapClaims{
			"user_id": "not-a-number",
			"email":   123,
		}, nil
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid token payload")
}
