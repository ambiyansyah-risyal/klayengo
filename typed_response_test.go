package klayengo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestUser represents a test user struct for unmarshaling
type TestUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TestAPIResponse represents a test API response struct
type TestAPIResponse struct {
	Success bool     `json:"success"`
	Data    TestUser `json:"data"`
	Message string   `json:"message"`
}

func TestGetJSON(t *testing.T) {
	expectedUser := TestUser{
		ID:    123,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedUser); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New()
	var user TestUser
	err := client.GetJSON(context.Background(), server.URL, &user)

	if err != nil {
		t.Fatalf("GetJSON() returned error: %v", err)
	}

	if user.ID != expectedUser.ID {
		t.Errorf("Expected ID %d, got %d", expectedUser.ID, user.ID)
	}
	if user.Name != expectedUser.Name {
		t.Errorf("Expected Name %s, got %s", expectedUser.Name, user.Name)
	}
	if user.Email != expectedUser.Email {
		t.Errorf("Expected Email %s, got %s", expectedUser.Email, user.Email)
	}
}

func TestPostJSON(t *testing.T) {
	requestUser := TestUser{
		Name:  "Jane Doe",
		Email: "jane@example.com",
	}
	expectedResponse := TestAPIResponse{
		Success: true,
		Data: TestUser{
			ID:    456,
			Name:  requestUser.Name,
			Email: requestUser.Email,
		},
		Message: "User created successfully",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var receivedUser TestUser
		if err := json.NewDecoder(r.Body).Decode(&receivedUser); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if receivedUser.Name != requestUser.Name {
			t.Errorf("Expected request Name %s, got %s", requestUser.Name, receivedUser.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New()
	var response TestAPIResponse
	err := client.PostJSON(context.Background(), server.URL, requestUser, &response)

	if err != nil {
		t.Fatalf("PostJSON() returned error: %v", err)
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if response.Data.ID != expectedResponse.Data.ID {
		t.Errorf("Expected Data.ID %d, got %d", expectedResponse.Data.ID, response.Data.ID)
	}
	if response.Message != expectedResponse.Message {
		t.Errorf("Expected Message %s, got %s", expectedResponse.Message, response.Message)
	}
}

func TestGetTyped(t *testing.T) {
	expectedUser := TestUser{
		ID:    789,
		Name:  "Alice Smith",
		Email: "alice@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedUser); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New()
	var user TestUser
	resp, err := client.GetTyped(context.Background(), server.URL, &user)

	if err != nil {
		t.Fatalf("GetTyped() returned error: %v", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Response.StatusCode)
	}
	if user.ID != expectedUser.ID {
		t.Errorf("Expected ID %d, got %d", expectedUser.ID, user.ID)
	}
	if user.Name != expectedUser.Name {
		t.Errorf("Expected Name %s, got %s", expectedUser.Name, user.Name)
	}
	if user.Email != expectedUser.Email {
		t.Errorf("Expected Email %s, got %s", expectedUser.Email, user.Email)
	}
}

func TestPostTyped(t *testing.T) {
	requestUser := TestUser{
		Name:  "Bob Wilson",
		Email: "bob@example.com",
	}
	expectedResponse := TestAPIResponse{
		Success: true,
		Data: TestUser{
			ID:    999,
			Name:  requestUser.Name,
			Email: requestUser.Email,
		},
		Message: "User updated successfully",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New()
	var response TestAPIResponse
	resp, err := client.PostTyped(context.Background(), server.URL, requestUser, &response)

	if err != nil {
		t.Fatalf("PostTyped() returned error: %v", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Response.StatusCode)
	}
	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if response.Data.ID != expectedResponse.Data.ID {
		t.Errorf("Expected Data.Data.ID %d, got %d", expectedResponse.Data.ID, response.Data.ID)
	}
}

func TestDoJSON(t *testing.T) {
	expectedUser := TestUser{
		ID:    555,
		Name:  "Carol Brown",
		Email: "carol@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedUser); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	req, err := http.NewRequestWithContext(context.Background(), "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	client := New()
	var user TestUser
	err = client.DoJSON(req, &user)

	if err != nil {
		t.Fatalf("DoJSON() returned error: %v", err)
	}

	if user.ID != expectedUser.ID {
		t.Errorf("Expected ID %d, got %d", expectedUser.ID, user.ID)
	}
	if user.Name != expectedUser.Name {
		t.Errorf("Expected Name %s, got %s", expectedUser.Name, user.Name)
	}
	if user.Email != expectedUser.Email {
		t.Errorf("Expected Email %s, got %s", expectedUser.Email, user.Email)
	}
}

func TestJSONErrorHandling(t *testing.T) {
	// Test HTTP error status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid request"}`))
	}))
	defer server.Close()

	client := New()
	var user TestUser
	err := client.GetJSON(context.Background(), server.URL, &user)

	if err == nil {
		t.Fatal("Expected error for 400 status, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP error 400") {
		t.Errorf("Expected error to contain 'HTTP error 400', got: %s", err.Error())
	}
}

func TestJSONInvalidJSONResponse(t *testing.T) {
	// Test invalid JSON response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := New()
	var user TestUser
	err := client.GetJSON(context.Background(), server.URL, &user)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal response") {
		t.Errorf("Expected error to contain 'failed to unmarshal response', got: %s", err.Error())
	}
}

func TestJSONEmptyResponse(t *testing.T) {
	// Test empty response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty response body
	}))
	defer server.Close()

	client := New()
	var user TestUser
	err := client.GetJSON(context.Background(), server.URL, &user)

	// Empty response should not error, target should remain unchanged
	if err != nil {
		t.Fatalf("Unexpected error for empty response: %v", err)
	}
}

// CustomUnmarshaler for testing - doubles numeric values
type CustomUnmarshaler struct{}

func (u *CustomUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	// First unmarshal normally
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	
	// Then double the ID if it's a TestUser
	if user, ok := v.(*TestUser); ok {
		user.ID = user.ID * 2
	}
	return nil
}

func TestCustomUnmarshaler(t *testing.T) {

	expectedUser := TestUser{
		ID:    100,
		Name:  "David Lee",
		Email: "david@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expectedUser); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New(WithUnmarshaler(&CustomUnmarshaler{}))
	var user TestUser
	err := client.GetJSON(context.Background(), server.URL, &user)

	if err != nil {
		t.Fatalf("GetJSON() returned error: %v", err)
	}

	// ID should be doubled by custom unmarshaler
	if user.ID != expectedUser.ID*2 {
		t.Errorf("Expected ID %d (doubled), got %d", expectedUser.ID*2, user.ID)
	}
	if user.Name != expectedUser.Name {
		t.Errorf("Expected Name %s, got %s", expectedUser.Name, user.Name)
	}
}

func TestPostJSONWithNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := New()
	var response map[string]bool
	err := client.PostJSON(context.Background(), server.URL, nil, &response)

	if err != nil {
		t.Fatalf("PostJSON() with nil body returned error: %v", err)
	}

	if !response["success"] {
		t.Error("Expected success to be true")
	}
}