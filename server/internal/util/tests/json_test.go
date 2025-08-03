package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

func TestRespondWithError(t *testing.T) {
	recorder := httptest.NewRecorder()
	util.RespondWithError(recorder, http.StatusBadRequest, "bad request error")

	res := recorder.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, res.StatusCode)
	}

	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}

	body, _ := io.ReadAll(res.Body)

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if payload["error"] != "bad request error" {
		t.Errorf("Expected error message 'bad request error', got '%s'", payload["error"])
	}
}

func TestRespondWithJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	payload := map[string]string{
		"message": "success",
	}

	util.RespondWithJSON(recorder, http.StatusOK, payload)

	res := recorder.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", ct)
	}

	body, _ := io.ReadAll(res.Body)

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got '%s'", result["message"])
	}
}

func TestRespondWithJSON_MarshalError(t *testing.T) {
	recorder := httptest.NewRecorder()

	payload := make(chan int)

	util.RespondWithJSON(recorder, http.StatusOK, payload)

	res := recorder.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)

	if len(strings.TrimSpace(string(body))) != 0 {
		t.Errorf("Expected empty body, got %s", string(body))
	}
}
