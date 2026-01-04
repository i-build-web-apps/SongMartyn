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

// generateRandomPIN generates a 6-digit random PIN
func generateRandomPIN() string {
	bytes := make([]byte, 3)
	rand.Read(bytes)
	// Convert to 6-digit number
	num := int(bytes[0])<<16 | int(bytes[1])<<8 | int(bytes[2])
	num = num % 1000000
	return strings.Repeat("0", 6-len(string(rune(num)))) + string(rune(num))
}

// GetPIN returns the admin PIN (for display on startup)
func (m *Manager) GetPIN() string {
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

	// If local, auto-authenticate
	if IsLocalRequest(r) {
		token := m.GenerateToken("local")
		json.NewEncoder(w).Encode(AuthResponse{
			Success: true,
			Token:   token,
			IsLocal: true,
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

	token := m.GenerateToken(req.MartynKey)
	json.NewEncoder(w).Encode(AuthResponse{
		Success: true,
		Token:   token,
		IsLocal: false,
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
