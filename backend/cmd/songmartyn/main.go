package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/joho/godotenv"

	"songmartyn/internal/admin"
	"songmartyn/internal/device"
	"songmartyn/internal/library"
	"songmartyn/internal/mpv"
	"songmartyn/internal/queue"
	"songmartyn/internal/session"
	"songmartyn/internal/websocket"
	"songmartyn/pkg/models"
)

// Re-export library.SearchLog for use in handlers
type SearchLog = library.SearchLog

// Config holds application configuration
type Config struct {
	Port          string // HTTPS port
	HTTPPort      string // HTTP port (for redirect to HTTPS)
	DataDir       string
	StaticDir     string
	DevMode       bool
	AdminPIN      string
	CertFile      string
	KeyFile       string
	YouTubeAPIKey string
}

// App holds the application state
type App struct {
	config   Config
	mpv      *mpv.Controller
	hub      *websocket.Hub
	sessions *session.Manager
	queue    *queue.Manager
	admin    *admin.Manager
	library  *library.Manager
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Performance: Lock OS thread for audio timing on Ubuntu
	if runtime.GOOS == "linux" {
		runtime.LockOSThread()
	}

	// Load .env file (optional - won't error if missing)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables and defaults")
	}

	// Parse flags (flags override .env values)
	port := flag.String("port", "", "HTTPS server port (overrides HTTPS_PORT)")
	httpPort := flag.String("http-port", "", "HTTP port (overrides HTTP_PORT)")
	dataDir := flag.String("data", "", "Data directory (overrides DATA_DIR)")
	staticDir := flag.String("static", "../frontend/dist", "Static files directory")
	devMode := flag.Bool("dev", false, "Development mode (enables CORS)")
	adminPIN := flag.String("pin", "", "Admin PIN (overrides ADMIN_PIN)")
	certFile := flag.String("cert", "", "TLS certificate (overrides TLS_CERT)")
	keyFile := flag.String("key", "", "TLS key (overrides TLS_KEY)")
	youtubeAPIKey := flag.String("youtube-api-key", "", "YouTube API key (overrides YOUTUBE_API_KEY)")
	flag.Parse()

	// Build config: flags > env > defaults
	config := Config{
		Port:          getEnv("HTTPS_PORT", "8443"),
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		DataDir:       getEnv("DATA_DIR", "./data"),
		StaticDir:     *staticDir,
		DevMode:       *devMode,
		AdminPIN:      getEnv("ADMIN_PIN", ""),
		CertFile:      getEnv("TLS_CERT", "./certs/cert.pem"),
		KeyFile:       getEnv("TLS_KEY", "./certs/key.pem"),
		YouTubeAPIKey: getEnv("YOUTUBE_API_KEY", ""),
	}

	// Flags override env values if provided
	if *port != "" {
		config.Port = *port
	}
	if *httpPort != "" {
		config.HTTPPort = *httpPort
	}
	if *dataDir != "" {
		config.DataDir = *dataDir
	}
	if *adminPIN != "" {
		config.AdminPIN = *adminPIN
	}
	if *certFile != "" {
		config.CertFile = *certFile
	}
	if *keyFile != "" {
		config.KeyFile = *keyFile
	}
	if *youtubeAPIKey != "" {
		config.YouTubeAPIKey = *youtubeAPIKey
	}

	// Ensure data directory exists
	os.MkdirAll(config.DataDir, 0755)

	app, err := NewApp(config)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		app.Shutdown()
		os.Exit(0)
	}()

	// Start the server
	app.Run()
}

// NewApp creates and initializes the application
func NewApp(config Config) (*App, error) {
	// Initialize session manager
	sessionDB := filepath.Join(config.DataDir, "sessions.db")
	sessions, err := session.NewManager(sessionDB)
	if err != nil {
		return nil, err
	}

	// Initialize queue manager
	queueDB := filepath.Join(config.DataDir, "queue.db")
	queueMgr, err := queue.NewManager(queueDB)
	if err != nil {
		return nil, err
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub()

	// Initialize mpv controller
	mpvCtrl := mpv.NewController()

	// Initialize admin manager
	adminMgr := admin.NewManager(config.AdminPIN)

	// Initialize library manager
	libraryDB := filepath.Join(config.DataDir, "library.db")
	libraryMgr, err := library.NewManager(libraryDB)
	if err != nil {
		return nil, err
	}

	app := &App{
		config:   config,
		mpv:      mpvCtrl,
		hub:      hub,
		sessions: sessions,
		queue:    queueMgr,
		admin:    adminMgr,
		library:  libraryMgr,
	}

	// Wire up handlers
	app.setupHandlers()

	return app, nil
}

// setupHandlers configures WebSocket message handlers
func (app *App) setupHandlers() {
	app.hub.SetHandlers(websocket.HubHandlers{
		OnHandshake: func(client *websocket.Client, payload websocket.HandshakePayload) (*models.Session, *models.RoomState) {
			// The Martyn Handshake - restore or create session
			sess := app.sessions.GetOrCreate(payload.MartynKey, payload.DisplayName)

			// Update device info
			userAgent := client.GetUserAgent()
			deviceName := device.GetFriendlyName(userAgent)
			app.sessions.UpdateDeviceInfo(sess.MartynKey, client.GetIPAddress(), userAgent, deviceName)

			// Mark as online
			sess.IsOnline = true
			sess.IPAddress = client.GetIPAddress()
			sess.UserAgent = userAgent
			sess.DeviceName = deviceName

			roomState := app.getRoomState()
			log.Printf("Session restored/created: %s (%s) from %s [%s]",
				sess.DisplayName, sess.MartynKey[:8], sess.IPAddress, deviceName)

			// Broadcast updated client list to admins
			app.broadcastClientList()

			return sess, &roomState
		},

		OnSearch: func(client *websocket.Client, query string) {
			// TODO: Implement YouTube search via yt-dlp
			log.Printf("Search request: %s", query)
			if sess := client.GetSession(); sess != nil {
				app.sessions.AddSearchHistory(sess.MartynKey, query)
			}
		},

		OnQueueAdd: func(client *websocket.Client, songID string) {
			// TODO: Fetch song metadata and add to queue
			log.Printf("Queue add request: %s", songID)
		},

		OnQueueRemove: func(client *websocket.Client, songID string) {
			app.queue.Remove(songID)
			app.broadcastState()
		},

		OnPlay: func(client *websocket.Client) {
			app.mpv.Play()
			app.broadcastState()
		},

		OnPause: func(client *websocket.Client) {
			app.mpv.Pause()
			app.broadcastState()
		},

		OnSkip: func(client *websocket.Client) {
			if next := app.queue.Next(); next != nil {
				app.playCurrentSong()
			}
			app.broadcastState()
		},

		OnSeek: func(client *websocket.Client, position float64) {
			app.mpv.Seek(position)
		},

		OnVocalAssist: func(client *websocket.Client, level models.VocalAssistLevel) {
			if sess := client.GetSession(); sess != nil {
				app.sessions.UpdateVocalAssist(sess.MartynKey, level)
				// Update current playback if this is the current singer
				app.updateVocalMix(level)
			}
		},

		OnVolume: func(client *websocket.Client, volume float64) {
			app.mpv.SetVolume(volume)
		},

		OnSetDisplayName: func(client *websocket.Client, name string) {
			if sess := client.GetSession(); sess != nil {
				app.sessions.UpdateDisplayName(sess.MartynKey, name)
				sess.DisplayName = name
				app.broadcastState()
				app.broadcastClientList()
			}
		},

		OnAdminSetAdmin: func(client *websocket.Client, martynKey string, isAdmin bool) error {
			if err := app.sessions.SetAdmin(martynKey, isAdmin); err != nil {
				return err
			}
			// Update the target client's session if online
			if targetClient := app.hub.FindClientByMartynKey(martynKey); targetClient != nil {
				if targetSess := targetClient.GetSession(); targetSess != nil {
					targetSess.IsAdmin = isAdmin
				}
			}
			app.broadcastClientList()
			log.Printf("Admin %s set %s admin=%v", client.GetSession().MartynKey[:8], martynKey[:8], isAdmin)
			return nil
		},

		OnAdminKick: func(client *websocket.Client, martynKey string, reason string) error {
			targetClient := app.hub.FindClientByMartynKey(martynKey)
			if targetClient == nil {
				return fmt.Errorf("client not found")
			}
			app.hub.KickClient(targetClient, reason)
			log.Printf("Admin %s kicked %s: %s", client.GetSession().MartynKey[:8], martynKey[:8], reason)
			return nil
		},

		OnClientDisconnect: func(client *websocket.Client) {
			if sess := client.GetSession(); sess != nil {
				app.sessions.SetOnline(sess.MartynKey, false)
				app.broadcastClientList()
			}
		},
	})

	// Queue change callback
	app.queue.OnChange(func() {
		app.broadcastState()
	})

	// mpv track end callback
	app.mpv.OnTrackEnd(func() {
		if next := app.queue.Next(); next != nil {
			app.playCurrentSong()
		} else {
			// TODO: Start Dawnsong BGM mode
			log.Println("Queue empty - Dawnsong mode would start here")
		}
		app.broadcastState()
	})

	// mpv state change callback
	app.mpv.OnStateChange(func(state models.PlayerState) {
		app.broadcastState()
	})
}

// getRoomState returns the current room state
func (app *App) getRoomState() models.RoomState {
	playerState, _ := app.mpv.GetState()
	playerState.CurrentSong = app.queue.Current()

	return models.RoomState{
		Player:   playerState,
		Queue:    app.queue.GetState(),
		Sessions: app.sessions.GetActiveSessions(),
	}
}

// broadcastState sends the current state to all connected clients
func (app *App) broadcastState() {
	state := app.getRoomState()
	app.hub.BroadcastState(state)
}

// broadcastClientList sends the client list to all admin clients
func (app *App) broadcastClientList() {
	clients := app.hub.GetConnectedClients()
	app.hub.BroadcastToAdmins(websocket.MsgClientList, clients)
}

// playCurrentSong starts playing the current song in the queue
func (app *App) playCurrentSong() {
	song := app.queue.Current()
	if song == nil {
		return
	}

	// If stems are available, use vocal mixing
	if song.InstrPath != "" && song.VocalPath != "" {
		gain := models.VocalGainMap[song.VocalAssist]
		app.mpv.SetVocalMix(song.InstrPath, song.VocalPath, gain)
	} else {
		// Play original video/audio
		app.mpv.LoadFile(song.VideoURL)
	}
}

// updateVocalMix updates the vocal assist level for current playback
func (app *App) updateVocalMix(level models.VocalAssistLevel) {
	song := app.queue.Current()
	if song == nil || song.InstrPath == "" || song.VocalPath == "" {
		return
	}

	gain := models.VocalGainMap[level]
	app.mpv.SetVocalMix(song.InstrPath, song.VocalPath, gain)
}

// Run starts the HTTP server and WebSocket hub
func (app *App) Run() {
	// Start WebSocket hub
	go app.hub.Run()

	// Start mpv
	if err := app.mpv.Start(); err != nil {
		log.Printf("Warning: Failed to start mpv: %v", err)
		log.Println("Continuing without video playback...")
	} else {
		log.Println("mpv started successfully")
	}

	// Setup routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", app.hub.ServeWS)

	// API endpoints
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"0.1.0"}`))
	})

	// Admin API endpoints
	mux.HandleFunc("/api/admin/auth", app.admin.HandleAuth)
	mux.HandleFunc("/api/admin/check", app.admin.HandleCheckAuth)
	mux.HandleFunc("/api/admin/clients", app.admin.Middleware(app.handleAdminClients))
	mux.HandleFunc("/api/admin/clients/", app.admin.Middleware(app.handleAdminClientAction))

	// Library API endpoints (admin only)
	mux.HandleFunc("/api/library/locations", app.admin.Middleware(app.handleLibraryLocations))
	mux.HandleFunc("/api/library/locations/", app.admin.Middleware(app.handleLibraryLocationAction))
	mux.HandleFunc("/api/library/search", app.handleLibrarySearch)
	mux.HandleFunc("/api/library/stats", app.handleLibraryStats)
	mux.HandleFunc("/api/library/popular", app.handleLibraryPopular)
	mux.HandleFunc("/api/library/history", app.handleLibraryHistory)

	// YouTube search endpoint
	mux.HandleFunc("/api/youtube/search", app.handleYouTubeSearch)

	// Search logs endpoints (admin only)
	mux.HandleFunc("/api/admin/search-logs", app.admin.Middleware(app.handleSearchLogs))
	mux.HandleFunc("/api/admin/search-stats", app.admin.Middleware(app.handleSearchStats))

	// Static files (frontend build) with SPA fallback
	if _, err := os.Stat(app.config.StaticDir); err == nil {
		mux.HandleFunc("/", spaHandler(app.config.StaticDir))
	}

	// CORS middleware for dev mode
	var handler http.Handler = mux
	if app.config.DevMode {
		handler = corsMiddleware(mux)
		log.Println("Development mode enabled - CORS allowed from all origins")
	}

	httpsAddr := ":" + app.config.Port
	httpAddr := ":" + app.config.HTTPPort

	log.Printf("SongMartyn starting on https://localhost%s", httpsAddr)
	log.Printf("HTTP redirect server on http://localhost%s", httpAddr)
	log.Printf("WebSocket endpoint: wss://localhost%s/ws", httpsAddr)
	log.Printf("Admin panel: https://localhost%s/admin", httpsAddr)
	log.Printf("Admin PIN: %s", app.admin.GetPIN())

	// Start HTTP redirect server in background
	go func() {
		redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Build HTTPS URL
			host := r.Host
			// Replace HTTP port with HTTPS port in host
			if app.config.HTTPPort != "80" {
				host = strings.TrimSuffix(host, ":"+app.config.HTTPPort)
			}
			if app.config.Port != "443" {
				host = host + ":" + app.config.Port
			}
			target := "https://" + host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})
		log.Printf("HTTP redirect server listening on %s", httpAddr)
		if err := http.ListenAndServe(httpAddr, redirectHandler); err != nil {
			log.Printf("HTTP redirect server error: %v", err)
		}
	}()

	// Start HTTPS server
	log.Printf("TLS enabled with cert: %s, key: %s", app.config.CertFile, app.config.KeyFile)
	if err := http.ListenAndServeTLS(httpsAddr, app.config.CertFile, app.config.KeyFile, handler); err != nil {
		log.Fatalf("HTTPS server failed: %v", err)
	}
}

// Shutdown gracefully shuts down the application
func (app *App) Shutdown() {
	app.mpv.Stop()
	app.sessions.Close()
	app.queue.Close()
	app.library.Close()
}

// spaHandler serves static files with SPA fallback to index.html
func spaHandler(staticDir string) http.HandlerFunc {
	fs := http.Dir(staticDir)
	fileServer := http.FileServer(fs)

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to open the file
		f, err := fs.Open(path)
		if err != nil {
			// File doesn't exist, serve index.html for SPA routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		// Check if it's a directory
		stat, err := os.Stat(filepath.Join(staticDir, path))
		if err == nil && stat.IsDir() {
			// Check for index.html in directory
			indexPath := filepath.Join(staticDir, path, "index.html")
			if _, err := os.Stat(indexPath); err != nil {
				// No index.html, serve root index.html
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Serve the file
		fileServer.ServeHTTP(w, r)
	}
}

// corsMiddleware adds CORS headers for development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleAdminClients handles GET /api/admin/clients - list all clients
func (app *App) handleAdminClients(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	clients := app.hub.GetConnectedClients()
	json.NewEncoder(w).Encode(clients)
}

// handleAdminClientAction handles POST/DELETE /api/admin/clients/{key}/...
func (app *App) handleAdminClientAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/admin/clients/{key}/{action}
	path := r.URL.Path
	// Remove prefix "/api/admin/clients/"
	path = path[len("/api/admin/clients/"):]

	parts := splitPath(path)
	if len(parts) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid path"})
		return
	}

	martynKey := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case action == "admin" && r.Method == http.MethodPost:
		// Toggle admin status
		var req struct {
			IsAdmin bool `json:"is_admin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
			return
		}

		if err := app.sessions.SetAdmin(martynKey, req.IsAdmin); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Update live session if online
		if client := app.hub.FindClientByMartynKey(martynKey); client != nil {
			if sess := client.GetSession(); sess != nil {
				sess.IsAdmin = req.IsAdmin
			}
		}

		app.broadcastClientList()
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case action == "" && r.Method == http.MethodDelete:
		// Kick client
		client := app.hub.FindClientByMartynKey(martynKey)
		if client == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Client not found"})
			return
		}

		app.hub.KickClient(client, "Kicked by admin")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown action"})
	}
}

// splitPath splits a URL path into segments
func splitPath(path string) []string {
	var parts []string
	for _, p := range splitOn(path, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// splitOn splits a string on a separator
func splitOn(s string, sep rune) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == sep {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}

// handleLibraryLocations handles GET/POST /api/library/locations
func (app *App) handleLibraryLocations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		locations, err := app.library.GetLocations()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if locations == nil {
			locations = []models.LibraryLocation{}
		}
		json.NewEncoder(w).Encode(locations)

	case http.MethodPost:
		var req struct {
			Path string `json:"path"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
			return
		}

		location, err := app.library.AddLocation(req.Path, req.Name)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		json.NewEncoder(w).Encode(location)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleLibraryLocationAction handles actions on specific locations
func (app *App) handleLibraryLocationAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse path: /api/library/locations/{id}/{action}
	path := r.URL.Path[len("/api/library/locations/"):]
	parts := splitPath(path)
	if len(parts) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid path"})
		return
	}

	var locationID int64
	fmt.Sscanf(parts[0], "%d", &locationID)

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case action == "scan" && r.Method == http.MethodPost:
		// Scan location for media files
		count, err := app.library.ScanLocation(locationID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"songs_found": count,
		})

	case action == "" && r.Method == http.MethodDelete:
		// Delete location
		if err := app.library.RemoveLocation(locationID); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown action"})
	}
}

// handleLibrarySearch handles GET /api/library/search?q=query
func (app *App) handleLibrarySearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode([]models.LibrarySong{})
		return
	}

	songs, err := app.library.SearchSongs(query, 50)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if songs == nil {
		songs = []models.LibrarySong{}
	}

	// Log the search
	ipAddress := getClientIP(r)
	martynKey := r.URL.Query().Get("key") // Optional martyn key from client
	app.library.LogSearch(query, "library", len(songs), martynKey, ipAddress)

	json.NewEncoder(w).Encode(songs)
}

// handleLibraryStats handles GET /api/library/stats
func (app *App) handleLibraryStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	totalSongs, totalPlays, err := app.library.GetStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]int{
		"total_songs": totalSongs,
		"total_plays": totalPlays,
	})
}

// handleLibraryPopular handles GET /api/library/popular
func (app *App) handleLibraryPopular(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	songs, err := app.library.GetPopularSongs(20)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if songs == nil {
		songs = []models.LibrarySong{}
	}
	json.NewEncoder(w).Encode(songs)
}

// handleLibraryHistory handles GET /api/library/history?key=martynKey
func (app *App) handleLibraryHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	martynKey := r.URL.Query().Get("key")
	if martynKey == "" {
		json.NewEncoder(w).Encode([]models.SongHistory{})
		return
	}

	history, err := app.library.GetUserHistory(martynKey, 50)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if history == nil {
		history = []models.SongHistory{}
	}
	json.NewEncoder(w).Encode(history)
}

// YouTubeResult represents a YouTube search result
type YouTubeResult struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Channel      string `json:"channel"`
	Duration     int    `json:"duration"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// YouTubeAPIResponse represents the YouTube Data API search response
type YouTubeAPIResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			ChannelTitle string `json:"channelTitle"`
			Thumbnails   struct {
				Medium struct {
					URL string `json:"url"`
				} `json:"medium"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

// YouTubeVideoDetailsResponse represents the YouTube video details response
type YouTubeVideoDetailsResponse struct {
	Items []struct {
		ID             string `json:"id"`
		ContentDetails struct {
			Duration string `json:"duration"`
		} `json:"contentDetails"`
	} `json:"items"`
}

// parseDuration converts ISO 8601 duration to seconds (e.g., PT4M30S -> 270)
func parseDuration(isoDuration string) int {
	// Remove PT prefix
	d := strings.TrimPrefix(isoDuration, "PT")

	var hours, minutes, seconds int

	// Parse hours
	if idx := strings.Index(d, "H"); idx != -1 {
		fmt.Sscanf(d[:idx], "%d", &hours)
		d = d[idx+1:]
	}

	// Parse minutes
	if idx := strings.Index(d, "M"); idx != -1 {
		fmt.Sscanf(d[:idx], "%d", &minutes)
		d = d[idx+1:]
	}

	// Parse seconds
	if idx := strings.Index(d, "S"); idx != -1 {
		fmt.Sscanf(d[:idx], "%d", &seconds)
	}

	return hours*3600 + minutes*60 + seconds
}

// handleYouTubeSearch handles GET /api/youtube/search?q=query
func (app *App) handleYouTubeSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode([]YouTubeResult{})
		return
	}

	// Check if API key is configured
	if app.config.YouTubeAPIKey == "" {
		log.Printf("YouTube search requested: %s (API key not configured)", query)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "YouTube API key not configured. Set YOUTUBE_API_KEY env or use -youtube-api-key flag.",
		})
		return
	}

	// Build YouTube Data API search URL
	searchURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?part=snippet&type=video&maxResults=20&q=%s&key=%s",
		url.QueryEscape(query),
		app.config.YouTubeAPIKey,
	)

	// Make the search request
	resp, err := http.Get(searchURL)
	if err != nil {
		log.Printf("YouTube API request failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "YouTube API request failed"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("YouTube API error (status %d): %s", resp.StatusCode, string(body))
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "YouTube API returned an error"})
		return
	}

	var searchResp YouTubeAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		log.Printf("Failed to parse YouTube API response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse YouTube response"})
		return
	}

	// Collect video IDs to fetch durations
	var videoIDs []string
	for _, item := range searchResp.Items {
		if item.ID.VideoID != "" {
			videoIDs = append(videoIDs, item.ID.VideoID)
		}
	}

	// Fetch video durations in a single request
	durations := make(map[string]int)
	if len(videoIDs) > 0 {
		detailsURL := fmt.Sprintf(
			"https://www.googleapis.com/youtube/v3/videos?part=contentDetails&id=%s&key=%s",
			url.QueryEscape(strings.Join(videoIDs, ",")),
			app.config.YouTubeAPIKey,
		)

		detailsResp, err := http.Get(detailsURL)
		if err == nil {
			defer detailsResp.Body.Close()
			if detailsResp.StatusCode == http.StatusOK {
				var videoDetails YouTubeVideoDetailsResponse
				if json.NewDecoder(detailsResp.Body).Decode(&videoDetails) == nil {
					for _, item := range videoDetails.Items {
						durations[item.ID] = parseDuration(item.ContentDetails.Duration)
					}
				}
			}
		}
	}

	// Build results
	var results []YouTubeResult
	for _, item := range searchResp.Items {
		if item.ID.VideoID == "" {
			continue
		}
		results = append(results, YouTubeResult{
			ID:           item.ID.VideoID,
			Title:        item.Snippet.Title,
			Channel:      item.Snippet.ChannelTitle,
			Duration:     durations[item.ID.VideoID],
			ThumbnailURL: item.Snippet.Thumbnails.Medium.URL,
		})
	}

	// Log the search
	ipAddress := getClientIP(r)
	martynKey := r.URL.Query().Get("key")
	app.library.LogSearch(query, "youtube", len(results), martynKey, ipAddress)

	log.Printf("YouTube search: %q returned %d results", query, len(results))
	json.NewEncoder(w).Encode(results)
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check for forwarded headers first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// handleSearchLogs handles GET /api/admin/search-logs
func (app *App) handleSearchLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodDelete {
		if err := app.library.ClearSearchLogs(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	source := r.URL.Query().Get("source")
	logs, err := app.library.GetSearchLogs(100, source)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if logs == nil {
		logs = []library.SearchLog{}
	}
	json.NewEncoder(w).Encode(logs)
}

// handleSearchStats handles GET /api/admin/search-stats
func (app *App) handleSearchStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	totalSearches, uniqueQueries, notFoundCount, topQueries, err := app.library.GetSearchStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_searches":  totalSearches,
		"unique_queries":  uniqueQueries,
		"not_found_count": notFoundCount,
		"top_not_found":   topQueries,
	})
}
