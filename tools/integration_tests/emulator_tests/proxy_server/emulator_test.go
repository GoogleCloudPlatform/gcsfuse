package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockHTTPClient helps simulate HTTP responses for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// TestGetRetryID tests the GetRetryID function
func TestGetRetryID(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/retry_test", r.URL.Path, "Unexpected URL path")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "test-id-123"})
	}))
	defer mockServer.Close()

	hostURL, _ := url.Parse(mockServer.URL)
	et := &emulatorTest{host: hostURL}

	instructions := map[string][]string{"retry": {"retry-instruction"}}
	testID, err := et.GetRetryID(instructions, "http")
	assert.NoError(t, err)
	assert.Equal(t, "test-id-123", testID, "Unexpected test ID returned")
}

// TestCreateRetryTest tests the CreateRetryTest function
func TestCreateRetryTest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/retry_test", r.URL.Path, "Unexpected URL path")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "test-id-123"})
	}))
	defer mockServer.Close()

	instructions := map[string][]string{"retry": {"retry-instruction"}}
	testID, err := CreateRetryTest(mockServer.URL, instructions)
	assert.NoError(t, err)
	assert.Equal(t, "test-id-123", testID, "Unexpected test ID returned")

	// Test with empty instructions
	testID, err = CreateRetryTest(mockServer.URL, map[string][]string{})
	assert.NoError(t, err)
	assert.Equal(t, "", testID, "Expected empty test ID for empty instructions")
}

// TestAddRetryID tests the AddRetryID function
func TestAddRetryID(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "test-id-123"})
	}))
	defer mockServer.Close()

	gConfig = &Config{TargetHost: mockServer.URL}
	gOpManager = &OperationManager{
		retryConfigs: map[RequestType][]RetryConfig{
			"TestType": {{Method: "TestType", RetryInstruction: "retry-instruction", RetryCount: 1, SkipCount: 0}},
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	r := RequestTypeAndInstruction{
		RequestType: "TestType",
		Instruction: "retry",
	}

	err := AddRetryID(req, r)
	assert.NoError(t, err)
	assert.Equal(t, "test-id-123", req.Header.Get("x-retry-test-id"), "Unexpected x-retry-test-id header value")
}
