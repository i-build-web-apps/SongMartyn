package websocket

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"songmartyn/pkg/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for mobile access
	},
}

// MessageType represents WebSocket message types
type MessageType string

const (
	// Client -> Server
	MsgHandshake      MessageType = "handshake"       // Initial connection with MartynKey
	MsgSearch         MessageType = "search"          // Search for songs
	MsgQueueAdd       MessageType = "queue_add"       // Add song to queue
	MsgQueueRemove    MessageType = "queue_remove"    // Remove song from queue
	MsgPlay           MessageType = "play"            // Play/resume
	MsgPause          MessageType = "pause"           // Pause
	MsgSkip           MessageType = "skip"            // Skip current song
	MsgSeek           MessageType = "seek"            // Seek to position
	MsgVocalAssist    MessageType = "vocal_assist"    // Set vocal assist level
	MsgVolume         MessageType = "volume"          // Set volume
	MsgSetDisplayName MessageType = "set_display_name" // Set custom display name

	// Admin messages (Client -> Server)
	MsgAdminSetAdmin MessageType = "admin_set_admin" // Promote/demote user to admin
	MsgAdminKick     MessageType = "admin_kick"      // Kick a user

	// Server -> Client
	MsgWelcome      MessageType = "welcome"       // Session restored/created
	MsgStateUpdate  MessageType = "state_update"  // Room state update
	MsgSearchResult MessageType = "search_result" // Search results
	MsgError        MessageType = "error"         // Error message
	MsgClientList   MessageType = "client_list"   // List of connected clients (admin)
	MsgKicked       MessageType = "kicked"        // You've been kicked
)

// Message represents a WebSocket message
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// HandshakePayload is sent by client on connection
type HandshakePayload struct {
	MartynKey   string `json:"martyn_key,omitempty"` // Empty for new sessions
	DisplayName string `json:"display_name,omitempty"`
}

// WelcomePayload is sent to client after handshake
type WelcomePayload struct {
	Session   models.Session   `json:"session"`
	RoomState models.RoomState `json:"room_state"`
}

// Client represents a connected WebSocket client
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	session   *models.Session
	ipAddress string
	userAgent string
}

// ClientInfo contains client connection info for admin display
type ClientInfo struct {
	MartynKey   string `json:"martyn_key"`
	DisplayName string `json:"display_name"`
	DeviceName  string `json:"device_name"`
	IPAddress   string `json:"ip_address"`
	IsAdmin     bool   `json:"is_admin"`
	IsOnline    bool   `json:"is_online"`
}

// AdminSetAdminPayload is the payload for setting admin status
type AdminSetAdminPayload struct {
	MartynKey string `json:"martyn_key"`
	IsAdmin   bool   `json:"is_admin"`
}

// AdminKickPayload is the payload for kicking a user
type AdminKickPayload struct {
	MartynKey string `json:"martyn_key"`
	Reason    string `json:"reason,omitempty"`
}

// Hub manages all WebSocket connections (The Nest Hub)
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	// Callbacks for handling messages
	onHandshake      func(client *Client, payload HandshakePayload) (*models.Session, *models.RoomState)
	onSearch         func(client *Client, query string)
	onQueueAdd       func(client *Client, songID string)
	onQueueRemove    func(client *Client, songID string)
	onPlay           func(client *Client)
	onPause          func(client *Client)
	onSkip           func(client *Client)
	onSeek           func(client *Client, position float64)
	onVocalAssist    func(client *Client, level models.VocalAssistLevel)
	onVolume         func(client *Client, volume float64)
	onSetDisplayName func(client *Client, name string)
	onAdminSetAdmin  func(client *Client, martynKey string, isAdmin bool) error
	onAdminKick      func(client *Client, martynKey string, reason string) error
	onClientDisconnect func(client *Client)
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected: %d total clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			_, wasConnected := h.clients[client]
			if wasConnected {
				delete(h.clients, client)
				close(client.send)
			}
			clientCount := len(h.clients)
			h.mu.Unlock()

			// Call disconnect callback AFTER releasing lock to avoid deadlock
			if wasConnected && h.onClientDisconnect != nil {
				h.onClientDisconnect(client)
			}
			log.Printf("Client disconnected: %d total clients", clientCount)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(msgType MessageType, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := Message{
		Type:    msgType,
		Payload: payloadBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.broadcast <- msgBytes
	return nil
}

// BroadcastState sends the current room state to all clients
func (h *Hub) BroadcastState(state models.RoomState) error {
	return h.Broadcast(MsgStateUpdate, state)
}

// SendTo sends a message to a specific client
func (h *Hub) SendTo(client *Client, msgType MessageType, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := Message{
		Type:    msgType,
		Payload: payloadBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case client.send <- msgBytes:
		return nil
	default:
		return nil // Channel full, drop message
	}
}

// ServeWS handles WebSocket upgrade requests
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Debug: Log incoming connection attempt
	ipAddress := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	log.Printf("[WS DEBUG] Connection attempt from %s (Origin: %s, UA: %s)",
		ipAddress, r.Header.Get("Origin"), userAgent[:min(50, len(userAgent))])

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS ERROR] WebSocket upgrade failed for %s: %v", ipAddress, err)
		return
	}

	log.Printf("[WS DEBUG] WebSocket upgraded successfully for %s", ipAddress)

	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 256),
		ipAddress: ipAddress,
		userAgent: userAgent,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

// min returns the smaller of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getClientIP extracts the real client IP from a request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (if behind proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// SetHandlers sets the message handler callbacks
func (h *Hub) SetHandlers(handlers HubHandlers) {
	h.onHandshake = handlers.OnHandshake
	h.onSearch = handlers.OnSearch
	h.onQueueAdd = handlers.OnQueueAdd
	h.onQueueRemove = handlers.OnQueueRemove
	h.onPlay = handlers.OnPlay
	h.onPause = handlers.OnPause
	h.onSkip = handlers.OnSkip
	h.onSeek = handlers.OnSeek
	h.onVocalAssist = handlers.OnVocalAssist
	h.onVolume = handlers.OnVolume
	h.onSetDisplayName = handlers.OnSetDisplayName
	h.onAdminSetAdmin = handlers.OnAdminSetAdmin
	h.onAdminKick = handlers.OnAdminKick
	h.onClientDisconnect = handlers.OnClientDisconnect
}

// HubHandlers contains all handler callbacks
type HubHandlers struct {
	OnHandshake        func(client *Client, payload HandshakePayload) (*models.Session, *models.RoomState)
	OnSearch           func(client *Client, query string)
	OnQueueAdd         func(client *Client, songID string)
	OnQueueRemove      func(client *Client, songID string)
	OnPlay             func(client *Client)
	OnPause            func(client *Client)
	OnSkip             func(client *Client)
	OnSeek             func(client *Client, position float64)
	OnVocalAssist      func(client *Client, level models.VocalAssistLevel)
	OnVolume           func(client *Client, volume float64)
	OnSetDisplayName   func(client *Client, name string)
	OnAdminSetAdmin    func(client *Client, martynKey string, isAdmin bool) error
	OnAdminKick        func(client *Client, martynKey string, reason string) error
	OnClientDisconnect func(client *Client)
}

// readPump pumps messages from the WebSocket to the hub
func (c *Client) readPump() {
	clientID := c.ipAddress // Use IP for logging before session is established
	log.Printf("[WS DEBUG] readPump started for %s", clientID)

	defer func() {
		log.Printf("[WS DEBUG] readPump closing for %s (session: %v)", clientID, c.session != nil)
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			// Log all close errors with details
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("[WS ERROR] Unexpected close for %s: %v", clientID, err)
			} else {
				log.Printf("[WS DEBUG] Connection closed for %s: %v", clientID, err)
			}
			break
		}

		log.Printf("[WS DEBUG] Received message type %d from %s (%d bytes)", messageType, clientID, len(data))

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[WS ERROR] Invalid message format from %s: %v (data: %s)", clientID, err, string(data[:min(100, len(data))]))
			continue
		}

		log.Printf("[WS DEBUG] Processing message '%s' from %s", msg.Type, clientID)
		c.handleMessage(msg)
	}
}

// writePump pumps messages from the hub to the WebSocket
func (c *Client) writePump() {
	clientID := c.ipAddress
	log.Printf("[WS DEBUG] writePump started for %s", clientID)

	defer func() {
		log.Printf("[WS DEBUG] writePump closing for %s", clientID)
		c.conn.Close()
	}()

	for message := range c.send {
		log.Printf("[WS DEBUG] Sending message to %s (%d bytes)", clientID, len(message))
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("[WS ERROR] Write failed for %s: %v", clientID, err)
			return
		}
	}
	log.Printf("[WS DEBUG] Send channel closed for %s", clientID)
}

// handleMessage processes incoming messages
func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case MsgHandshake:
		var payload HandshakePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Invalid handshake payload: %v", err)
			return
		}
		if c.hub.onHandshake != nil {
			session, roomState := c.hub.onHandshake(c, payload)
			if session != nil {
				c.session = session
				c.hub.SendTo(c, MsgWelcome, WelcomePayload{
					Session:   *session,
					RoomState: *roomState,
				})
			}
		}

	case MsgSearch:
		var query string
		if err := json.Unmarshal(msg.Payload, &query); err != nil {
			return
		}
		if c.hub.onSearch != nil {
			c.hub.onSearch(c, query)
		}

	case MsgQueueAdd:
		var songID string
		if err := json.Unmarshal(msg.Payload, &songID); err != nil {
			return
		}
		if c.hub.onQueueAdd != nil {
			c.hub.onQueueAdd(c, songID)
		}

	case MsgQueueRemove:
		var songID string
		if err := json.Unmarshal(msg.Payload, &songID); err != nil {
			return
		}
		if c.hub.onQueueRemove != nil {
			c.hub.onQueueRemove(c, songID)
		}

	case MsgPlay:
		if c.hub.onPlay != nil {
			c.hub.onPlay(c)
		}

	case MsgPause:
		if c.hub.onPause != nil {
			c.hub.onPause(c)
		}

	case MsgSkip:
		if c.hub.onSkip != nil {
			c.hub.onSkip(c)
		}

	case MsgSeek:
		var position float64
		if err := json.Unmarshal(msg.Payload, &position); err != nil {
			return
		}
		if c.hub.onSeek != nil {
			c.hub.onSeek(c, position)
		}

	case MsgVocalAssist:
		var level models.VocalAssistLevel
		if err := json.Unmarshal(msg.Payload, &level); err != nil {
			return
		}
		if c.hub.onVocalAssist != nil {
			c.hub.onVocalAssist(c, level)
		}

	case MsgVolume:
		var volume float64
		if err := json.Unmarshal(msg.Payload, &volume); err != nil {
			return
		}
		if c.hub.onVolume != nil {
			c.hub.onVolume(c, volume)
		}

	case MsgSetDisplayName:
		var name string
		if err := json.Unmarshal(msg.Payload, &name); err != nil {
			return
		}
		if c.hub.onSetDisplayName != nil {
			c.hub.onSetDisplayName(c, name)
		}

	case MsgAdminSetAdmin:
		// Check if client is admin
		if c.session == nil || !c.session.IsAdmin {
			c.hub.SendTo(c, MsgError, map[string]string{"error": "Not authorized"})
			return
		}
		var payload AdminSetAdminPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		if c.hub.onAdminSetAdmin != nil {
			if err := c.hub.onAdminSetAdmin(c, payload.MartynKey, payload.IsAdmin); err != nil {
				c.hub.SendTo(c, MsgError, map[string]string{"error": err.Error()})
			}
		}

	case MsgAdminKick:
		// Check if client is admin
		if c.session == nil || !c.session.IsAdmin {
			c.hub.SendTo(c, MsgError, map[string]string{"error": "Not authorized"})
			return
		}
		var payload AdminKickPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		if c.hub.onAdminKick != nil {
			if err := c.hub.onAdminKick(c, payload.MartynKey, payload.Reason); err != nil {
				c.hub.SendTo(c, MsgError, map[string]string{"error": err.Error()})
			}
		}
	}
}

// GetSession returns the client's session
func (c *Client) GetSession() *models.Session {
	return c.session
}

// GetIPAddress returns the client's IP address
func (c *Client) GetIPAddress() string {
	return c.ipAddress
}

// GetUserAgent returns the client's User-Agent string
func (c *Client) GetUserAgent() string {
	return c.userAgent
}

// GetConnectedClients returns info about all connected clients
func (h *Hub) GetConnectedClients() []ClientInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]ClientInfo, 0, len(h.clients))
	for client := range h.clients {
		if client.session != nil {
			clients = append(clients, ClientInfo{
				MartynKey:   client.session.MartynKey,
				DisplayName: client.session.DisplayName,
				DeviceName:  client.session.DeviceName,
				IPAddress:   client.ipAddress,
				IsAdmin:     client.session.IsAdmin,
				IsOnline:    true,
			})
		}
	}
	return clients
}

// FindClientByMartynKey finds a connected client by their MartynKey
func (h *Hub) FindClientByMartynKey(martynKey string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.session != nil && client.session.MartynKey == martynKey {
			return client
		}
	}
	return nil
}

// KickClient disconnects a client and sends them a kicked message
func (h *Hub) KickClient(client *Client, reason string) {
	if client == nil {
		return
	}

	// Send kicked message
	h.SendTo(client, MsgKicked, map[string]string{"reason": reason})

	// Close the connection
	client.conn.Close()
}

// BroadcastToAdmins sends a message only to admin clients
func (h *Hub) BroadcastToAdmins(msgType MessageType, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := Message{
		Type:    msgType,
		Payload: payloadBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.session != nil && client.session.IsAdmin {
			select {
			case client.send <- msgBytes:
			default:
				// Channel full, skip
			}
		}
	}
	return nil
}
