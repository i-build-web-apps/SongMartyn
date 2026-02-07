package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDiagnosticsEndpointGET tests GET /api/admin/diagnostics returns cached diagnostics
func TestDiagnosticsEndpointGET(t *testing.T) {
	// Create a minimal app for testing
	app := &App{
		config: Config{
			Port:     "8443",
			HTTPPort: "8080",
		},
	}

	// Initialize diagnostics
	app.diagnostics = DiagnosticsInfo{
		PortChecks: []PortCheck{
			{Port: 8443, Protocol: "tcp", Description: "HTTPS", Status: "open"},
			{Port: 8080, Protocol: "tcp", Description: "HTTP", Status: "open"},
		},
		Displays: []DisplayInfo{
			{Name: "Test Display", Resolution: "1920x1080", Type: "internal", Connection: "builtin", Main: true},
		},
		FirewallEnabled: false,
		FirewallStatus:  "disabled",
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/admin/diagnostics", nil)
	w := httptest.NewRecorder()

	// Call handler directly (bypassing middleware for unit test)
	app.handleDiagnostics(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var result DiagnosticsInfo
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify diagnostics data
	if len(result.PortChecks) != 2 {
		t.Errorf("Expected 2 port checks, got %d", len(result.PortChecks))
	}
	if len(result.Displays) != 1 {
		t.Errorf("Expected 1 display, got %d", len(result.Displays))
	}
	if result.Displays[0].Name != "Test Display" {
		t.Errorf("Expected display name 'Test Display', got '%s'", result.Displays[0].Name)
	}
}

// TestDiagnosticsEndpointPOST tests POST /api/admin/diagnostics refreshes diagnostics
func TestDiagnosticsEndpointPOST(t *testing.T) {
	app := &App{
		config: Config{
			Port:     "8443",
			HTTPPort: "8080",
		},
	}

	// Create POST request
	req := httptest.NewRequest(http.MethodPost, "/api/admin/diagnostics", nil)
	w := httptest.NewRecorder()

	// Call handler (this will refresh diagnostics)
	app.handleDiagnostics(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var result DiagnosticsInfo
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// After POST, diagnostics should be populated
	if result.PortChecks == nil {
		t.Error("Expected port checks to be populated after refresh")
	}
}

// TestSystemInfoEndpoint tests GET /api/admin/system-info
func TestSystemInfoEndpoint(t *testing.T) {
	app := &App{
		config: Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/system-info", nil)
	w := httptest.NewRecorder()

	app.handleSystemInfo(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result SystemInfo
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify required fields are populated
	if result.OS == "" {
		t.Error("Expected OS to be populated")
	}
	if result.Arch == "" {
		t.Error("Expected Arch to be populated")
	}
	if result.GoVersion == "" {
		t.Error("Expected GoVersion to be populated")
	}
	if result.CPUCount <= 0 {
		t.Error("Expected CPUCount to be greater than 0")
	}
}

// TestIcecastStreamsEndpoint tests GET /api/admin/icecast-streams
func TestIcecastStreamsEndpoint(t *testing.T) {
	app := &App{
		config: Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/icecast-streams", nil)
	w := httptest.NewRecorder()

	app.handleIcecastStreams(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var streams []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&streams); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify streams are returned
	if len(streams) == 0 {
		t.Error("Expected at least one icecast stream")
	}

	// Verify first stream has required fields
	if len(streams) > 0 {
		first := streams[0]
		if _, ok := first["name"]; !ok {
			t.Error("Expected stream to have 'name' field")
		}
		if _, ok := first["url"]; !ok {
			t.Error("Expected stream to have 'url' field")
		}
		if _, ok := first["genre"]; !ok {
			t.Error("Expected stream to have 'genre' field")
		}
	}
}

// TestIcecastStreamsMethodNotAllowed tests POST to icecast-streams returns 405
func TestIcecastStreamsMethodNotAllowed(t *testing.T) {
	app := &App{
		config: Config{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/icecast-streams", nil)
	w := httptest.NewRecorder()

	app.handleIcecastStreams(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestPortCheckStructure tests the PortCheck struct serialization
func TestPortCheckStructure(t *testing.T) {
	pc := PortCheck{
		Port:        8443,
		Protocol:    "tcp",
		Description: "HTTPS Server",
		Status:      "open",
		Error:       "",
	}

	data, err := json.Marshal(pc)
	if err != nil {
		t.Fatalf("Failed to marshal PortCheck: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result["port"].(float64) != 8443 {
		t.Errorf("Expected port 8443, got %v", result["port"])
	}
	if result["status"].(string) != "open" {
		t.Errorf("Expected status 'open', got %v", result["status"])
	}
}

// TestDisplayInfoStructure tests the DisplayInfo struct serialization
func TestDisplayInfoStructure(t *testing.T) {
	di := DisplayInfo{
		Name:       "Dell U2715H",
		Resolution: "2560x1440",
		Type:       "external",
		Connection: "DisplayPort",
		Main:       false,
	}

	data, err := json.Marshal(di)
	if err != nil {
		t.Fatalf("Failed to marshal DisplayInfo: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result["name"].(string) != "Dell U2715H" {
		t.Errorf("Expected name 'Dell U2715H', got %v", result["name"])
	}
	if result["main"].(bool) != false {
		t.Errorf("Expected main false, got %v", result["main"])
	}
}

// TestDiagnosticsInfoStructure tests the DiagnosticsInfo struct serialization
func TestDiagnosticsInfoStructure(t *testing.T) {
	di := DiagnosticsInfo{
		PortChecks: []PortCheck{
			{Port: 8443, Protocol: "tcp", Status: "open"},
		},
		Displays: []DisplayInfo{
			{Name: "Test", Main: true},
		},
		FirewallEnabled: true,
		FirewallStatus:  "enabled",
	}

	data, err := json.Marshal(di)
	if err != nil {
		t.Fatalf("Failed to marshal DiagnosticsInfo: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result["firewall_enabled"].(bool) != true {
		t.Errorf("Expected firewall_enabled true, got %v", result["firewall_enabled"])
	}
	if result["firewall_status"].(string) != "enabled" {
		t.Errorf("Expected firewall_status 'enabled', got %v", result["firewall_status"])
	}
}
