package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	testVersion := "1.0.0-test"
	handler := Health(testVersion)
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	status, ok := response["status"].(string)
	if !ok || status != "ok" {
		t.Errorf("Expected status 'ok', got %v", status)
	}

	version, ok := response["version"].(string)
	if !ok || version != testVersion {
		t.Errorf("Expected version '%s', got %v", testVersion, version)
	}
}
