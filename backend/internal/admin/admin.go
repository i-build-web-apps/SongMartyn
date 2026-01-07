package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Manager handles admin authentication and authorization
type Manager struct {
	pin           string
	localhostOnly bool // If true, only localhost can access admin
	tokens        map[string]tokenInfo // token -> info
	mu            sync.RWMutex
	tokenExpiry   time.Duration
	onAdminAuth   func(martynKey string) // Callback when user authenticates as admin
}

type tokenInfo struct {
	MartynKey string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// NewManager creates a new admin manager
// If pin is empty, admin access is restricted to localhost only
func NewManager(pin string) *Manager {
	localhostOnly := pin == ""
	return &Manager{
		pin:           pin,
		localhostOnly: localhostOnly,
		tokens:        make(map[string]tokenInfo),
		tokenExpiry:   24 * time.Hour,
	}
}

// IsLocalhostOnly returns true if admin access is restricted to localhost
func (m *Manager) IsLocalhostOnly() bool {
	return m.localhostOnly
}

// SetOnAdminAuth sets a callback that's called when a user authenticates as admin
func (m *Manager) SetOnAdminAuth(callback func(martynKey string)) {
	m.onAdminAuth = callback
}

// GetPIN returns the admin PIN (for display on startup)
// Returns empty string if localhost-only mode
func (m *Manager) GetPIN() string {
	if m.localhostOnly {
		return ""
	}
	return m.pin
}

// ValidatePIN checks if the provided PIN is correct
func (m *Manager) ValidatePIN(pin string) bool {
	return m.pin == pin
}

// GenerateToken creates a new admin token for a session
func (m *Manager) GenerateToken(martynKey string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate random token
	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	m.tokens[token] = tokenInfo{
		MartynKey: martynKey,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(m.tokenExpiry),
	}

	return token
}

// ValidateToken checks if a token is valid
func (m *Manager) ValidateToken(token string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.tokens[token]
	if !ok {
		return "", false
	}

	if time.Now().After(info.ExpiresAt) {
		return "", false
	}

	return info.MartynKey, true
}

// RevokeToken removes a token
func (m *Manager) RevokeToken(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, token)
}

// SetPIN updates the admin PIN and invalidates all non-local tokens
// This forces all remote admin users to re-authenticate with the new PIN
func (m *Manager) SetPIN(newPIN string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pin = newPIN
	m.localhostOnly = newPIN == ""

	// Clear all tokens except local ones
	for token, info := range m.tokens {
		if info.MartynKey != "local" {
			delete(m.tokens, token)
		}
	}
}

// RevokeAllNonLocalTokens invalidates all remote admin tokens
func (m *Manager) RevokeAllNonLocalTokens() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for token, info := range m.tokens {
		if info.MartynKey != "local" {
			delete(m.tokens, token)
			count++
		}
	}
	return count
}

// CleanupExpiredTokens removes expired tokens
func (m *Manager) CleanupExpiredTokens() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for token, info := range m.tokens {
		if now.After(info.ExpiresAt) {
			delete(m.tokens, token)
		}
	}
}

// IsLocalRequest checks if a request is from localhost
func IsLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ip.IsLoopback()
}

// GetClientIP extracts the real client IP from a request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (if behind proxy)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// AuthResponse is the response for auth endpoints
type AuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Error   string `json:"error,omitempty"`
	IsLocal bool   `json:"is_local"`
}

// Middleware provides admin authentication middleware
func (m *Manager) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Allow localhost without authentication
		if IsLocalRequest(r) {
			next(w, r)
			return
		}

		// If localhost-only mode, reject all non-local requests
		if m.localhostOnly {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(AuthResponse{
				Success: false,
				Error:   "Admin access is restricted to localhost only",
			})
			return
		}

		// Check for Bearer token
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(AuthResponse{
				Success: false,
				Error:   "Authentication required",
			})
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if _, valid := m.ValidateToken(token); !valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(AuthResponse{
				Success: false,
				Error:   "Invalid or expired token",
			})
			return
		}

		next(w, r)
	}
}

// HandleAuth handles PIN authentication requests
func (m *Manager) HandleAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get martyn_key from query string (works for both GET and POST)
	martynKey := r.URL.Query().Get("martyn_key")

	// If local, auto-authenticate
	if IsLocalRequest(r) {
		token := m.GenerateToken(martynKey)
		// Mark session as admin if martyn_key provided
		if martynKey != "" && m.onAdminAuth != nil {
			m.onAdminAuth(martynKey)
		}
		json.NewEncoder(w).Encode(AuthResponse{
			Success: true,
			Token:   token,
			IsLocal: true,
		})
		return
	}

	// If localhost-only mode, reject remote auth attempts
	if m.localhostOnly {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Error:   "Admin access is restricted to localhost only",
			IsLocal: false,
		})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	var req struct {
		PIN       string `json:"pin"`
		MartynKey string `json:"martyn_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	if !m.ValidatePIN(req.PIN) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Error:   "Invalid PIN",
		})
		return
	}

	// Use martyn_key from body if provided, otherwise from query
	if req.MartynKey != "" {
		martynKey = req.MartynKey
	}

	// Mark session as admin
	if martynKey != "" && m.onAdminAuth != nil {
		m.onAdminAuth(martynKey)
	}

	token := m.GenerateToken(martynKey)
	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Token:   token,
		IsLocal: false,
	})
}

// IsAuthorized checks if a request is authorized (for inline auth checks)
func (m *Manager) IsAuthorized(r *http.Request) bool {
	// Localhost is always authorized
	if IsLocalRequest(r) {
		return true
	}

	// If localhost-only mode, non-local requests are not authorized
	if m.localhostOnly {
		return false
	}

	// Check for Bearer token
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	_, valid := m.ValidateToken(token)
	return valid
}

// HandleSetPIN handles PIN update requests (localhost only)
func (m *Manager) HandleSetPIN(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only localhost can change the PIN
	if !IsLocalRequest(r) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Error:   "Only localhost can change the admin PIN",
		})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	var req struct {
		PIN string `json:"pin"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AuthResponse{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	m.SetPIN(req.PIN)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"localhost_only": m.localhostOnly,
		"message":        "PIN updated, all remote admin sessions have been invalidated",
	})
}

// HandleCheckAuth checks if current auth is valid
func (m *Manager) HandleCheckAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	isLocal := IsLocalRequest(r)

	// Check for existing token
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		if _, valid := m.ValidateToken(token); valid {
			json.NewEncoder(w).Encode(AuthResponse{
				Success: true,
				IsLocal: isLocal,
			})
			return
		}
	}

	// Local access is auto-authenticated
	if isLocal {
		json.NewEncoder(w).Encode(AuthResponse{
			Success: true,
			IsLocal: true,
		})
		return
	}

	json.NewEncoder(w).Encode(AuthResponse{
		Success: false,
		IsLocal: false,
	})
}
