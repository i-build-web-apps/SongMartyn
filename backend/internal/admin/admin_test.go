package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// Manager Initialization Tests
// =============================================================================

func TestNewManager(t *testing.T) {
	m := NewManager("1234")

	if m.pin != "1234" {
		t.Errorf("Expected PIN '1234', got '%s'", m.pin)
	}
	if m.localhostOnly {
		t.Error("Expected localhostOnly to be false when PIN is set")
	}
}

func TestNewManagerWithEmptyPIN(t *testing.T) {
	m := NewManager("")

	if !m.localhostOnly {
		t.Error("Expected localhostOnly to be true when PIN is empty")
	}
}

// =============================================================================
// PIN Validation Tests
// =============================================================================

func TestValidatePIN_Correct(t *testing.T) {
	m := NewManager("1234")

	if !m.ValidatePIN("1234") {
		t.Error("Expected valid PIN to return true")
	}
}

func TestValidatePIN_Incorrect(t *testing.T) {
	m := NewManager("1234")

	if m.ValidatePIN("0000") {
		t.Error("Expected invalid PIN '0000' to return false")
	}
	if m.ValidatePIN("") {
		t.Error("Expected empty PIN to return false")
	}
	if m.ValidatePIN("12345") {
		t.Error("Expected PIN '12345' to return false")
	}
	if m.ValidatePIN("abcd") {
		t.Error("Expected PIN 'abcd' to return false")
	}
}

func TestValidatePIN_CaseSensitive(t *testing.T) {
	m := NewManager("AbCd")

	if !m.ValidatePIN("AbCd") {
		t.Error("Expected exact match to return true")
	}
	if m.ValidatePIN("abcd") {
		t.Error("Expected case mismatch 'abcd' to return false")
	}
	if m.ValidatePIN("ABCD") {
		t.Error("Expected case mismatch 'ABCD' to return false")
	}
}

// =============================================================================
// Token Management Tests
// =============================================================================

func TestGenerateToken(t *testing.T) {
	m := NewManager("1234")

	token1 := m.GenerateToken("user1")
	token2 := m.GenerateToken("user2")

	if token1 == "" {
		t.Error("Expected non-empty token")
	}
	if token1 == token2 {
		t.Error("Expected unique tokens for different users")
	}
}

func TestValidateToken_Valid(t *testing.T) {
	m := NewManager("1234")

	token := m.GenerateToken("user1")
	martynKey, valid := m.ValidateToken(token)

	if !valid {
		t.Error("Expected valid token to return true")
	}
	if martynKey != "user1" {
		t.Errorf("Expected martyn_key 'user1', got '%s'", martynKey)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	m := NewManager("1234")

	_, valid := m.ValidateToken("invalid-token")
	if valid {
		t.Error("Expected invalid token to return false")
	}
}

func TestRevokeToken(t *testing.T) {
	m := NewManager("1234")

	token := m.GenerateToken("user1")

	// Token should be valid initially
	_, valid := m.ValidateToken(token)
	if !valid {
		t.Error("Token should be valid before revocation")
	}

	m.RevokeToken(token)

	// Token should be invalid after revocation
	_, valid = m.ValidateToken(token)
	if valid {
		t.Error("Token should be invalid after revocation")
	}
}

// =============================================================================
// SetPIN Tests
// =============================================================================

func TestSetPIN(t *testing.T) {
	m := NewManager("1234")

	m.SetPIN("5678")

	if !m.ValidatePIN("5678") {
		t.Error("Expected new PIN '5678' to be valid")
	}
	if m.ValidatePIN("1234") {
		t.Error("Expected old PIN '1234' to be invalid")
	}
}

func TestSetPIN_InvalidatesRemoteTokens(t *testing.T) {
	m := NewManager("1234")

	// Generate a token for a remote user
	remoteToken := m.GenerateToken("remote-user")

	// Verify token is valid
	_, valid := m.ValidateToken(remoteToken)
	if !valid {
		t.Error("Token should be valid before PIN change")
	}

	// Change PIN
	m.SetPIN("5678")

	// Remote token should be invalidated
	_, valid = m.ValidateToken(remoteToken)
	if valid {
		t.Error("Remote token should be invalidated after PIN change")
	}
}

func TestSetPIN_ToEmpty(t *testing.T) {
	m := NewManager("1234")

	m.SetPIN("")

	if !m.localhostOnly {
		t.Error("Expected localhostOnly to be true after setting empty PIN")
	}
}

// =============================================================================
// HTTP Handler Tests - Remote Authentication
// =============================================================================

// mockRemoteRequest creates a request that appears to be from a remote IP
func mockRemoteRequest(method, url string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	// Set RemoteAddr to a non-loopback IP to simulate remote request
	req.RemoteAddr = "192.168.1.100:12345"
	return req
}

// mockLocalRequest creates a request that appears to be from localhost
func mockLocalRequest(method, url string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	// Default RemoteAddr is 192.0.2.1:1234, need to override for localhost
	req.RemoteAddr = "127.0.0.1:12345"
	return req
}

func TestHandleAuth_RemoteWithValidPIN(t *testing.T) {
	m := NewManager("1234")

	body, _ := json.Marshal(map[string]string{
		"pin":        "1234",
		"martyn_key": "test-user",
	})

	req := mockRemoteRequest("POST", "/api/admin/auth", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.HandleAuth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if !resp.Success {
		t.Errorf("Expected success=true, got error: %s", resp.Error)
	}
	if resp.Token == "" {
		t.Error("Expected token to be returned")
	}
	if resp.IsLocal {
		t.Error("Expected IsLocal to be false for remote request")
	}
}

func TestHandleAuth_RemoteWithInvalidPIN(t *testing.T) {
	m := NewManager("1234")

	body, _ := json.Marshal(map[string]string{
		"pin":        "0000",
		"martyn_key": "test-user",
	})

	req := mockRemoteRequest("POST", "/api/admin/auth", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.HandleAuth(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.Success {
		t.Error("Expected success=false for invalid PIN")
	}
	if resp.Error != "Invalid PIN" {
		t.Errorf("Expected error 'Invalid PIN', got '%s'", resp.Error)
	}
}

func TestHandleAuth_RemoteWithEmptyPIN(t *testing.T) {
	m := NewManager("1234")

	body, _ := json.Marshal(map[string]string{
		"pin":        "",
		"martyn_key": "test-user",
	})

	req := mockRemoteRequest("POST", "/api/admin/auth", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.HandleAuth(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.Success {
		t.Error("Expected success=false for empty PIN")
	}
}

func TestHandleAuth_LocalAutoAuth(t *testing.T) {
	m := NewManager("1234")

	req := mockLocalRequest("GET", "/api/admin/auth?martyn_key=local-user", nil)

	rr := httptest.NewRecorder()
	m.HandleAuth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if !resp.Success {
		t.Errorf("Expected success=true for local request, got error: %s", resp.Error)
	}
	if resp.Token == "" {
		t.Error("Expected token to be returned for local request")
	}
	if !resp.IsLocal {
		t.Error("Expected IsLocal to be true for local request")
	}
}

func TestHandleAuth_RemoteWithLocalhostOnlyMode(t *testing.T) {
	m := NewManager("") // Empty PIN = localhost-only mode

	body, _ := json.Marshal(map[string]string{
		"pin":        "1234",
		"martyn_key": "test-user",
	})

	req := mockRemoteRequest("POST", "/api/admin/auth", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.HandleAuth(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.Success {
		t.Error("Expected success=false in localhost-only mode")
	}
}

// =============================================================================
// Middleware Tests
// =============================================================================

func TestMiddleware_LocalRequest(t *testing.T) {
	m := NewManager("1234")

	handlerCalled := false
	handler := m.Middleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := mockLocalRequest("GET", "/api/admin/test", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if !handlerCalled {
		t.Error("Expected handler to be called for local request")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_RemoteWithValidToken(t *testing.T) {
	m := NewManager("1234")

	// Generate a valid token
	token := m.GenerateToken("test-user")

	handlerCalled := false
	handler := m.Middleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := mockRemoteRequest("GET", "/api/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if !handlerCalled {
		t.Error("Expected handler to be called with valid token")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_RemoteWithInvalidToken(t *testing.T) {
	m := NewManager("1234")

	handlerCalled := false
	handler := m.Middleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := mockRemoteRequest("GET", "/api/admin/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()

	handler(rr, req)

	if handlerCalled {
		t.Error("Expected handler NOT to be called with invalid token")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestMiddleware_RemoteWithNoToken(t *testing.T) {
	m := NewManager("1234")

	handlerCalled := false
	handler := m.Middleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := mockRemoteRequest("GET", "/api/admin/test", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if handlerCalled {
		t.Error("Expected handler NOT to be called without token")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestMiddleware_RemoteInLocalhostOnlyMode(t *testing.T) {
	m := NewManager("") // localhost-only mode

	handlerCalled := false
	handler := m.Middleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := mockRemoteRequest("GET", "/api/admin/test", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if handlerCalled {
		t.Error("Expected handler NOT to be called in localhost-only mode")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}
}

// =============================================================================
// IsAuthorized Tests
// =============================================================================

func TestIsAuthorized_LocalRequest(t *testing.T) {
	m := NewManager("1234")

	req := mockLocalRequest("GET", "/test", nil)

	if !m.IsAuthorized(req) {
		t.Error("Expected local request to be authorized")
	}
}

func TestIsAuthorized_RemoteWithValidToken(t *testing.T) {
	m := NewManager("1234")

	token := m.GenerateToken("test-user")
	req := mockRemoteRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	if !m.IsAuthorized(req) {
		t.Error("Expected remote request with valid token to be authorized")
	}
}

func TestIsAuthorized_RemoteWithInvalidToken(t *testing.T) {
	m := NewManager("1234")

	req := mockRemoteRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	if m.IsAuthorized(req) {
		t.Error("Expected remote request with invalid token to NOT be authorized")
	}
}

func TestIsAuthorized_RemoteWithNoToken(t *testing.T) {
	m := NewManager("1234")

	req := mockRemoteRequest("GET", "/test", nil)

	if m.IsAuthorized(req) {
		t.Error("Expected remote request without token to NOT be authorized")
	}
}

// =============================================================================
// Admin Callback Tests
// =============================================================================

func TestOnAdminAuth_CallbackInvoked(t *testing.T) {
	m := NewManager("1234")

	callbackCalled := false
	callbackMartynKey := ""

	m.SetOnAdminAuth(func(martynKey string) {
		callbackCalled = true
		callbackMartynKey = martynKey
	})

	body, _ := json.Marshal(map[string]string{
		"pin":        "1234",
		"martyn_key": "test-user",
	})

	req := mockRemoteRequest("POST", "/api/admin/auth", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.HandleAuth(rr, req)

	if !callbackCalled {
		t.Error("Expected onAdminAuth callback to be invoked")
	}
	if callbackMartynKey != "test-user" {
		t.Errorf("Expected martyn_key 'test-user', got '%s'", callbackMartynKey)
	}
}

// =============================================================================
// HandleSetPIN Tests (localhost only)
// =============================================================================

func TestHandleSetPIN_LocalRequest(t *testing.T) {
	m := NewManager("1234")

	body, _ := json.Marshal(map[string]string{"pin": "5678"})
	req := mockLocalRequest("POST", "/api/admin/pin", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.HandleSetPIN(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Verify PIN was changed
	if !m.ValidatePIN("5678") {
		t.Error("Expected new PIN to be set")
	}
}

func TestHandleSetPIN_RemoteRequest(t *testing.T) {
	m := NewManager("1234")

	body, _ := json.Marshal(map[string]string{"pin": "5678"})
	req := mockRemoteRequest("POST", "/api/admin/pin", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	m.HandleSetPIN(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for remote PIN change, got %d", rr.Code)
	}

	// Verify PIN was NOT changed
	if !m.ValidatePIN("1234") {
		t.Error("Expected PIN to remain unchanged")
	}
}

// =============================================================================
// HandleCheckAuth Tests
// =============================================================================

func TestHandleCheckAuth_LocalRequest(t *testing.T) {
	m := NewManager("1234")

	req := mockLocalRequest("GET", "/api/admin/check", nil)
	rr := httptest.NewRecorder()

	m.HandleCheckAuth(rr, req)

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if !resp.Success {
		t.Error("Expected success=true for local check")
	}
	if !resp.IsLocal {
		t.Error("Expected IsLocal=true for local request")
	}
}

func TestHandleCheckAuth_RemoteWithValidToken(t *testing.T) {
	m := NewManager("1234")

	token := m.GenerateToken("test-user")
	req := mockRemoteRequest("GET", "/api/admin/check", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	m.HandleCheckAuth(rr, req)

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if !resp.Success {
		t.Error("Expected success=true with valid token")
	}
	if resp.IsLocal {
		t.Error("Expected IsLocal=false for remote request")
	}
}

func TestHandleCheckAuth_RemoteWithNoToken(t *testing.T) {
	m := NewManager("1234")

	req := mockRemoteRequest("GET", "/api/admin/check", nil)
	rr := httptest.NewRecorder()

	m.HandleCheckAuth(rr, req)

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.Success {
		t.Error("Expected success=false without token")
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestRevokeAllNonLocalTokens(t *testing.T) {
	m := NewManager("1234")

	// Generate multiple tokens
	token1 := m.GenerateToken("user1")
	token2 := m.GenerateToken("user2")
	localToken := m.GenerateToken("local")

	// Revoke all non-local tokens
	count := m.RevokeAllNonLocalTokens()

	if count != 2 {
		t.Errorf("Expected 2 tokens revoked, got %d", count)
	}

	// Non-local tokens should be invalid
	if _, valid := m.ValidateToken(token1); valid {
		t.Error("Expected token1 to be invalid")
	}
	if _, valid := m.ValidateToken(token2); valid {
		t.Error("Expected token2 to be invalid")
	}

	// Local token should still be valid
	if _, valid := m.ValidateToken(localToken); !valid {
		t.Error("Expected local token to still be valid")
	}
}
