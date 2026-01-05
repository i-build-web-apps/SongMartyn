package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"songmartyn/internal/admin"
	"songmartyn/internal/avatar"
	"songmartyn/internal/device"
	"songmartyn/internal/holdingscreen"
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
	VideoPlayer   string
	LaunchBrowser bool   // Auto-launch admin page on startup
}

// App holds the application state
type App struct {
	config        Config
	mpv           *mpv.Controller
	hub           *websocket.Hub
	sessions      *session.Manager
	queue         *queue.Manager
	admin         *admin.Manager
	library       *library.Manager
	holdingScreen *holdingscreen.Generator

	// BGM (Background Music) state
	bgmSettings models.BGMSettings
	bgmActive   bool

	// Inter-song countdown state
	countdown       models.CountdownState
	countdownTicker *time.Ticker
	countdownStop   chan struct{}
	countdownMu     sync.Mutex
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns environment variable as bool
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
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
	launchBrowser := flag.Bool("launch-browser", false, "Auto-launch admin page in browser (overrides LAUNCH_BROWSER)")
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
		VideoPlayer:   getEnv("VIDEO_PLAYER", "mpv"),
		LaunchBrowser: getEnvBool("LAUNCH_BROWSER", false),
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
	if *launchBrowser {
		config.LaunchBrowser = true
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

	// Auto-launch browser to admin page if configured
	if config.LaunchBrowser {
		adminURL := fmt.Sprintf("https://localhost:%s/admin", config.Port)
		go func() {
			// Small delay to ensure server is ready
			time.Sleep(500 * time.Millisecond)
			if err := openBrowser(adminURL); err != nil {
				log.Printf("Failed to open browser: %v", err)
			} else {
				log.Printf("Opened admin page in browser: %s", adminURL)
			}
		}()
	}

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
	mpvCtrl := mpv.NewController(config.VideoPlayer)

	// Initialize admin manager
	adminMgr := admin.NewManager(config.AdminPIN)

	// Initialize library manager
	libraryDB := filepath.Join(config.DataDir, "library.db")
	libraryMgr, err := library.NewManager(libraryDB)
	if err != nil {
		return nil, err
	}

	// Initialize holding screen generator
	holdingScreenTempDir := filepath.Join(config.DataDir, "temp")
	avatarAPIURL := fmt.Sprintf("https://localhost:%s", config.Port)
	holdingScreenGen, err := holdingscreen.NewGenerator(holdingScreenTempDir, avatarAPIURL)
	if err != nil {
		log.Printf("Warning: Failed to initialize holding screen: %v", err)
		// Continue without holding screen - it's not critical
	}

	app := &App{
		config:        config,
		mpv:           mpvCtrl,
		hub:           hub,
		sessions:      sessions,
		queue:         queueMgr,
		admin:         adminMgr,
		library:       libraryMgr,
		holdingScreen: holdingScreenGen,
	}

	// Wire up handlers
	app.setupHandlers()

	return app, nil
}

// setupHandlers configures WebSocket message handlers
func (app *App) setupHandlers() {
	app.hub.SetHandlers(websocket.HubHandlers{
		OnHandshake: func(client *websocket.Client, payload websocket.HandshakePayload) (*models.Session, *models.RoomState) {
			// Check if user is blocked before allowing connection
			if payload.MartynKey != "" {
				if blocked, reason := app.sessions.IsBlocked(payload.MartynKey); blocked {
					log.Printf("Blocked user attempted to connect: %s - %s", payload.MartynKey[:8], reason)
					// Send kicked message and reject connection
					app.hub.SendTo(client, websocket.MsgKicked, map[string]string{
						"reason": "You are blocked: " + reason,
					})
					return nil, nil
				}
			}

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

		OnQueueAdd: func(client *websocket.Client, songID string, vocalAssist models.VocalAssistLevel) {
			// Check if this is the first song (queue is empty before adding)
			wasEmpty := app.queue.IsEmpty()

			// Fetch song from library
			libSong, err := app.library.GetSong(songID)
			if err != nil {
				log.Printf("Failed to get song %s: %v", songID, err)
				app.hub.SendTo(client, websocket.MsgError, map[string]string{"error": "Song not found"})
				return
			}

			// Use vocal assist from request, or default to OFF
			if vocalAssist == "" {
				vocalAssist = models.VocalOff
			}

			// Convert LibrarySong to queue Song
			song := models.Song{
				ID:           libSong.ID,
				Title:        libSong.Title,
				Artist:       libSong.Artist,
				Duration:     libSong.Duration,
				ThumbnailURL: libSong.ThumbnailURL,
				VideoURL:     libSong.FilePath, // Use file path as video URL
				VocalPath:    libSong.VocalPath,
				InstrPath:    libSong.InstrPath,
				CDGPath:      libSong.CDGPath,   // CDG graphics file
				AudioPath:    libSong.AudioPath, // Audio for CDG
				VocalAssist:  vocalAssist,
				AddedBy:      client.GetSession().MartynKey,
			}

			// Add to queue
			if err := app.queue.Add(song); err != nil {
				log.Printf("Failed to add song to queue: %v", err)
				app.hub.SendTo(client, websocket.MsgError, map[string]string{"error": "Failed to add to queue"})
				return
			}

			log.Printf("Song '%s' added to queue by %s", song.Title, client.GetSession().DisplayName)

			// Always update holding screen to show "Next Up" info
			app.showHoldingScreen()

			// Auto-start playback if this is the first song and autoplay is enabled
			if wasEmpty && app.queue.GetAutoplay() {
				log.Println("First song added to empty queue - starting playback in 2 seconds")
				// Brief delay to show "Next Up" on holding screen before playing
				go func() {
					time.Sleep(2 * time.Second)
					app.playCurrentSong()
					app.broadcastState()
				}()
			}

			app.broadcastState()
		},

		OnQueueRemove: func(client *websocket.Client, songID string) {
			// Get the current singer's key before removal (for countdown logic)
			currentSingerKey := ""
			if current := app.queue.Current(); current != nil {
				currentSingerKey = current.AddedBy
			}

			currentRemoved, _ := app.queue.Remove(songID)
			log.Printf("Song removed from queue by %s", client.GetSession().DisplayName)

			if currentRemoved {
				// Stop current playback
				app.mpv.StopPlayback()

				// Check if there's a next song to play
				if next := app.queue.Current(); next != nil {
					log.Println("Current song removed - starting countdown for next song")
					app.startCountdown(currentSingerKey)
				} else {
					log.Println("Current song removed - queue empty, showing holding screen")
					app.showHoldingScreen()
				}
			}

			app.broadcastState()
		},

		OnQueueMove: func(client *websocket.Client, from int, to int) {
			if err := app.queue.Move(from, to); err != nil {
				log.Printf("Failed to move song in queue: %v", err)
				return
			}
			log.Printf("Queue reordered by %s: %d -> %d", client.GetSession().DisplayName, from, to)
			app.broadcastState()
		},

		OnQueueClear: func(client *websocket.Client) {
			if err := app.queue.Clear(); err != nil {
				log.Printf("Failed to clear queue: %v", err)
				return
			}
			log.Printf("Queue cleared by %s", client.GetSession().DisplayName)
			// Stop current playback and show holding screen
			app.mpv.StopPlayback()
			app.showHoldingScreen()
			app.broadcastState()
		},

		OnPlay: func(client *websocket.Client) {
			if err := app.mpv.Play(); err != nil {
				log.Printf("Failed to play: %v", err)
			}
			app.broadcastState()
		},

		OnPause: func(client *websocket.Client) {
			if err := app.mpv.Pause(); err != nil {
				log.Printf("Failed to pause: %v", err)
			}
			app.broadcastState()
		},

		OnSkip: func(client *websocket.Client) {
			log.Printf("Skip requested by %s", client.GetSession().DisplayName)
			// Get current singer before advancing queue
			currentSingerKey := ""
			if current := app.queue.Current(); current != nil {
				currentSingerKey = current.AddedBy
			}
			app.mpv.Stop()
			if next := app.queue.Next(); next != nil {
				// Use countdown system for consistent transitions
				app.startCountdown(currentSingerKey)
			} else {
				log.Println("Skip: no more songs in queue")
				app.showHoldingScreen()
			}
			app.broadcastState()
		},

		OnSeek: func(client *websocket.Client, position float64) {
			if err := app.mpv.Seek(position); err != nil {
				log.Printf("Failed to seek to %.2f: %v", position, err)
			}
		},

		OnVocalAssist: func(client *websocket.Client, level models.VocalAssistLevel) {
			if sess := client.GetSession(); sess != nil {
				app.sessions.UpdateVocalAssist(sess.MartynKey, level)
				// Update current playback if this is the current singer
				app.updateVocalMix(level)
			}
		},

		OnVolume: func(client *websocket.Client, volume float64) {
			if err := app.mpv.SetVolume(volume); err != nil {
				log.Printf("Failed to set volume to %.0f: %v", volume, err)
			}
		},

		OnSetDisplayName: func(client *websocket.Client, name string, avatarID string, avatarConfig *models.AvatarConfig) {
			if sess := client.GetSession(); sess != nil {
				// Only update name if provided (non-empty)
				if name != "" {
					sess.DisplayName = name
				}
				sess.AvatarID = avatarID
				sess.AvatarConfig = avatarConfig
				app.sessions.UpdateProfile(sess.MartynKey, sess.DisplayName, avatarID)
				app.sessions.UpdateAvatarConfig(sess.MartynKey, avatarConfig)
				app.broadcastState()
				app.broadcastClientList()
			}
		},

		OnAutoplay: func(client *websocket.Client, enabled bool) {
			app.queue.SetAutoplay(enabled)
			log.Printf("Autoplay set to %v by %s", enabled, client.GetSession().DisplayName)
			app.broadcastState()
		},

		OnQueueShuffle: func(client *websocket.Client) {
			app.queue.Shuffle()
			log.Printf("Queue shuffled by %s", client.GetSession().DisplayName)
			// Refresh holding screen in case next song changed
			app.updateHoldingScreenIfIdle()
			app.broadcastState()
		},

		OnQueueRequeue: func(client *websocket.Client, songID string, martynKey string) {
			if err := app.queue.Requeue(songID, martynKey); err != nil {
				log.Printf("Failed to requeue song %s: %v", songID, err)
				return
			}
			log.Printf("Song %s requeued by %s for user %s",
				songID[:min(8, len(songID))],
				client.GetSession().DisplayName,
				martynKey[:min(8, len(martynKey))])
			app.broadcastState()
		},

		OnSetAFK: func(client *websocket.Client, isAFK bool) {
			sess := client.GetSession()
			if sess == nil {
				return
			}

			// Update session AFK status
			sess.IsAFK = isAFK
			app.sessions.SetAFK(sess.MartynKey, isAFK)

			// If going AFK, bump their songs to end of queue
			if isAFK {
				app.queue.BumpUserToEnd(sess.MartynKey)
				log.Printf("%s went AFK - songs bumped to end", sess.DisplayName)
			} else {
				log.Printf("%s is back from AFK", sess.DisplayName)
			}

			app.broadcastState()
			app.broadcastClientList()
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

			// Remove all of their songs from queue
			currentRemoved, _ := app.queue.RemoveByUser(martynKey)
			if currentRemoved {
				// Stop current playback and skip to next song or show holding screen
				if err := app.mpv.StopPlayback(); err != nil {
					log.Printf("Warning: failed to stop playback: %v", err)
				}
				if next := app.queue.Current(); next != nil {
					app.playCurrentSong()
				} else {
					app.showHoldingScreen()
				}
			}

			app.hub.KickClient(targetClient, reason)
			log.Printf("Admin %s kicked %s: %s", client.GetSession().MartynKey[:8], martynKey[:8], reason)
			app.broadcastState()
			return nil
		},

		OnAdminBlock: func(client *websocket.Client, martynKey string, durationMinutes int, reason string) error {
			// Block the user
			duration := time.Duration(durationMinutes) * time.Minute
			if err := app.sessions.BlockUser(martynKey, duration, reason); err != nil {
				return err
			}

			// Remove all of their songs from queue
			currentRemoved, _ := app.queue.RemoveByUser(martynKey)
			if currentRemoved {
				// Stop current playback and skip to next song or show holding screen
				if err := app.mpv.StopPlayback(); err != nil {
					log.Printf("Warning: failed to stop playback: %v", err)
				}
				if next := app.queue.Current(); next != nil {
					app.playCurrentSong()
				} else {
					app.showHoldingScreen()
				}
			}

			// Also kick them if they're connected
			targetClient := app.hub.FindClientByMartynKey(martynKey)
			if targetClient != nil {
				app.hub.KickClient(targetClient, "You have been blocked: "+reason)
			}

			if durationMinutes == 0 {
				log.Printf("Admin %s permanently blocked %s: %s", client.GetSession().MartynKey[:8], martynKey[:8], reason)
			} else {
				log.Printf("Admin %s blocked %s for %d minutes: %s", client.GetSession().MartynKey[:8], martynKey[:8], durationMinutes, reason)
			}
			app.broadcastClientList()
			app.broadcastState()
			return nil
		},

		OnAdminUnblock: func(client *websocket.Client, martynKey string) error {
			if err := app.sessions.UnblockUser(martynKey); err != nil {
				return err
			}
			log.Printf("Admin %s unblocked %s", client.GetSession().MartynKey[:8], martynKey[:8])
			app.broadcastClientList()
			return nil
		},

		OnAdminSetAFK: func(client *websocket.Client, martynKey string, isAFK bool) error {
			// Update session AFK status
			if err := app.sessions.SetAFK(martynKey, isAFK); err != nil {
				return err
			}

			// Update live session if online
			if targetClient := app.hub.FindClientByMartynKey(martynKey); targetClient != nil {
				if targetSess := targetClient.GetSession(); targetSess != nil {
					targetSess.IsAFK = isAFK
				}
			}

			// If going AFK, bump their songs to end of queue
			if isAFK {
				app.queue.BumpUserToEnd(martynKey)
				log.Printf("Admin %s set %s to AFK - songs bumped to end", client.GetSession().MartynKey[:8], martynKey[:8])
			} else {
				log.Printf("Admin %s set %s back from AFK", client.GetSession().MartynKey[:8], martynKey[:8])
			}

			app.broadcastState()
			app.broadcastClientList()
			return nil
		},

		OnAdminPlayNext: func(client *websocket.Client) error {
			log.Printf("Admin %s triggered play", client.GetSession().MartynKey[:8])
			// Start a countdown before playing (gives singer time to get ready)
			app.startPlayCountdown(10) // 10 second countdown
			return nil
		},

		OnAdminStop: func(client *websocket.Client) error {
			log.Printf("Admin %s stopped playback", client.GetSession().MartynKey[:8])
			// Stop any active countdown
			app.stopCountdown()
			// Stop current playback but keep MPV running
			if err := app.mpv.StopPlayback(); err != nil {
				log.Printf("Warning: failed to stop playback: %v", err)
			}
			// Skip current song (moves it to history)
			app.queue.Skip()
			// Show holding screen with next song info
			app.showHoldingScreen()
			app.broadcastState()
			return nil
		},

		OnAdminSetName: func(client *websocket.Client, martynKey string, displayName string) error {
			log.Printf("Admin %s changed name for %s to '%s'",
				client.GetSession().MartynKey[:8], martynKey[:8], displayName)
			if err := app.sessions.AdminSetDisplayName(martynKey, displayName); err != nil {
				return err
			}
			// Update live session if online
			if targetClient := app.hub.FindClientByMartynKey(martynKey); targetClient != nil {
				if targetSess := targetClient.GetSession(); targetSess != nil {
					targetSess.DisplayName = displayName
				}
			}
			app.broadcastState()
			app.broadcastClientList()
			return nil
		},

		OnAdminSetNameLock: func(client *websocket.Client, martynKey string, locked bool) error {
			log.Printf("Admin %s set name lock for %s to %v",
				client.GetSession().MartynKey[:8], martynKey[:8], locked)
			if err := app.sessions.SetNameLocked(martynKey, locked); err != nil {
				return err
			}
			// Update live session if online
			if targetClient := app.hub.FindClientByMartynKey(martynKey); targetClient != nil {
				if targetSess := targetClient.GetSession(); targetSess != nil {
					targetSess.NameLocked = locked
				}
			}
			app.broadcastState()
			app.broadcastClientList()
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
		// Check if autoplay is enabled
		if !app.queue.GetAutoplay() {
			log.Println("Track ended - autoplay disabled, waiting for manual skip")
			app.broadcastState()
			return
		}

		// Get the current singer before advancing
		currentSong := app.queue.Current()
		currentSingerKey := ""
		if currentSong != nil {
			currentSingerKey = currentSong.AddedBy
		}

		// Advance to next song
		if next := app.queue.Next(); next != nil {
			// Start the inter-song countdown
			app.startCountdown(currentSingerKey)
		} else {
			// Start BGM if enabled
			if app.bgmSettings.Enabled && app.bgmSettings.URL != "" {
				app.startBGM()
			} else {
				log.Println("Queue empty - showing holding screen")
				app.showHoldingScreen()
			}
			app.broadcastState()
		}
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
	playerState.BGMActive = app.bgmActive
	playerState.BGMEnabled = app.bgmSettings.Enabled

	// Get countdown state safely
	app.countdownMu.Lock()
	countdownState := app.countdown
	app.countdownMu.Unlock()

	return models.RoomState{
		Player:    playerState,
		Queue:     app.queue.GetState(),
		Sessions:  app.sessions.GetActiveSessions(),
		Countdown: countdownState,
	}
}

// broadcastState sends the current state to all connected clients
func (app *App) broadcastState() {
	state := app.getRoomState()
	app.hub.BroadcastState(state)
}

// broadcastClientList sends the client list to all admin clients
func (app *App) broadcastClientList() {
	// Get connected clients
	clients := app.hub.GetConnectedClients()

	// Create a map of existing clients for quick lookup
	clientMap := make(map[string]*websocket.ClientInfo)
	for i := range clients {
		clientMap[clients[i].MartynKey] = &clients[i]
	}

	// Add blocked users to the list
	blockedUsers := app.sessions.GetBlockedUsers()
	for _, bu := range blockedUsers {
		if existing, ok := clientMap[bu.MartynKey]; ok {
			// Update existing client with block info
			existing.IsBlocked = true
			existing.BlockReason = bu.Reason
		} else {
			// Add blocked user as offline client
			sess := app.sessions.Get(bu.MartynKey)
			deviceName := ""
			ipAddress := ""
			var avatarConfig *models.AvatarConfig
			if sess != nil {
				deviceName = sess.DeviceName
				ipAddress = sess.IPAddress
				avatarConfig = sess.AvatarConfig
			}
			clients = append(clients, websocket.ClientInfo{
				MartynKey:    bu.MartynKey,
				DisplayName:  bu.DisplayName,
				DeviceName:   deviceName,
				IPAddress:    ipAddress,
				IsAdmin:      false,
				IsOnline:     false,
				IsBlocked:    true,
				BlockReason:  bu.Reason,
				AvatarConfig: avatarConfig,
			})
		}
	}

	app.hub.BroadcastToAdmins(websocket.MsgClientList, clients)
}

// playCurrentSong starts playing the current song in the queue
// startBGM starts background music playback
func (app *App) startBGM() {
	if !app.bgmSettings.Enabled || app.bgmSettings.URL == "" {
		return
	}

	log.Printf("Starting BGM: %s", app.bgmSettings.URL)
	app.bgmActive = true

	// Set BGM volume (typically lower than main content)
	if app.bgmSettings.Volume > 0 {
		app.mpv.SetVolume(app.bgmSettings.Volume)
	}

	// Load BGM URL (works with YouTube via yt-dlp)
	if err := app.mpv.LoadFile(app.bgmSettings.URL); err != nil {
		log.Printf("Failed to load BGM: %v", err)
		app.bgmActive = false
	}
}

// stopBGM stops background music playback
func (app *App) stopBGM() {
	if !app.bgmActive {
		return
	}

	log.Println("Stopping BGM for song playback")
	app.bgmActive = false
}

// showHoldingScreen displays the holding screen with QR code and next-up info
func (app *App) showHoldingScreen() {
	if app.holdingScreen == nil {
		return
	}

	// Get connection URL
	connectURL := app.autoDetectConnectURL()

	// Check for next song in queue
	var nextUp *holdingscreen.NextUpInfo
	queueState := app.queue.GetState()

	// Only show "next up" if there's actually an upcoming song
	// (position must be within bounds - not exhausted/in history)
	if queueState.Position < len(queueState.Songs) {
		nextSong := queueState.Songs[queueState.Position]
		singer := app.sessions.Get(nextSong.AddedBy)

		nextUp = &holdingscreen.NextUpInfo{
			SongTitle:  nextSong.Title,
			SongArtist: nextSong.Artist,
			SingerName: "Unknown",
		}
		if singer != nil {
			nextUp.SingerName = singer.DisplayName
			nextUp.AvatarConfig = singer.AvatarConfig
		}
	}

	// Generate and display the holding screen
	imagePath, err := app.holdingScreen.Generate(connectURL, nextUp)
	if err != nil {
		log.Printf("Failed to generate holding screen: %v", err)
		return
	}

	if err := app.mpv.LoadImage(imagePath); err != nil {
		log.Printf("Failed to load holding screen: %v", err)
		return
	}

	log.Println("Holding screen displayed")
}

// updateHoldingScreenIfIdle refreshes the holding screen if no song is currently playing
func (app *App) updateHoldingScreenIfIdle() {
	// Only update if we're not currently playing a song
	if app.queue.Current() == nil || !app.queue.GetAutoplay() {
		app.showHoldingScreen()
	}
}

// startCountdown starts the inter-song countdown
// currentSingerKey is the MartynKey of the singer who just finished
func (app *App) startCountdown(currentSingerKey string) {
	app.countdownMu.Lock()

	// Stop any existing countdown
	if app.countdownTicker != nil {
		app.countdownTicker.Stop()
		close(app.countdownStop)
	}

	// Get the next song
	nextSong := app.queue.Current()
	if nextSong == nil {
		log.Println("startCountdown: no next song")
		app.countdownMu.Unlock()
		return
	}

	// Check if next singer is same as current
	requiresApproval := nextSong.AddedBy != currentSingerKey

	// Initialize countdown state
	app.countdown = models.CountdownState{
		Active:           true,
		SecondsRemaining: 15,
		NextSongID:       nextSong.ID,
		NextSingerKey:    nextSong.AddedBy,
		RequiresApproval: requiresApproval,
	}

	// Create ticker and stop channel
	app.countdownTicker = time.NewTicker(1 * time.Second)
	app.countdownStop = make(chan struct{})

	log.Printf("Starting 15-second countdown for next song (requires approval: %v)", requiresApproval)

	// Unlock BEFORE calling broadcastState to avoid deadlock
	// (broadcastState -> getRoomState -> countdownMu.Lock would deadlock)
	app.countdownMu.Unlock()

	// Show holding screen with next up info
	app.showHoldingScreen()
	app.broadcastState()

	go func() {
		for {
			select {
			case <-app.countdownStop:
				return
			case <-app.countdownTicker.C:
				app.countdownMu.Lock()
				if !app.countdown.Active {
					app.countdownMu.Unlock()
					return
				}

				app.countdown.SecondsRemaining--

				if app.countdown.SecondsRemaining <= 0 {
					// Countdown finished
					if !app.countdown.RequiresApproval {
						// Same user - auto-play
						log.Println("Countdown finished - auto-playing next song (same user)")
						app.countdown.Active = false
						app.countdownMu.Unlock()
						app.playNextSongNow()
						return
					} else {
						// Different user - wait for admin approval
						log.Println("Countdown finished - waiting for admin approval (different user)")
						app.countdown.SecondsRemaining = 0
						app.countdownMu.Unlock()
						// Broadcast AFTER releasing lock to avoid deadlock
						app.broadcastState()
						return
					}
				}

				app.countdownMu.Unlock()
				// Broadcast AFTER releasing lock to avoid deadlock
				app.broadcastState()
			}
		}
	}()
}

// startPlayCountdown starts a countdown before playing (admin-initiated)
// This gives the singer time to get ready before the song starts
func (app *App) startPlayCountdown(seconds int) {
	// Check if MPV is running, restart if not
	if !app.mpv.IsRunning() {
		log.Println("MPV not running - restarting before countdown")
		if err := app.mpv.Restart(); err != nil {
			log.Printf("Failed to restart MPV: %v", err)
			return
		}
		log.Println("MPV restarted successfully")
	}

	app.countdownMu.Lock()

	// Stop any existing countdown
	if app.countdownTicker != nil {
		app.countdownTicker.Stop()
		close(app.countdownStop)
	}

	// Get the next song
	nextSong := app.queue.Current()
	if nextSong == nil {
		log.Println("startPlayCountdown: no song in queue")
		app.countdownMu.Unlock()
		return
	}

	// Initialize countdown state - RequiresApproval=false means it will auto-play
	app.countdown = models.CountdownState{
		Active:           true,
		SecondsRemaining: seconds,
		NextSongID:       nextSong.ID,
		NextSingerKey:    nextSong.AddedBy,
		RequiresApproval: false, // Admin initiated, will auto-play
	}

	// Create ticker and stop channel
	app.countdownTicker = time.NewTicker(1 * time.Second)
	app.countdownStop = make(chan struct{})

	log.Printf("Starting %d-second play countdown", seconds)

	// Unlock BEFORE calling broadcastState to avoid deadlock
	app.countdownMu.Unlock()

	// Show holding screen with next up info and countdown
	app.showHoldingScreen()
	app.broadcastState()

	go func() {
		for {
			select {
			case <-app.countdownStop:
				return
			case <-app.countdownTicker.C:
				app.countdownMu.Lock()
				if !app.countdown.Active {
					app.countdownMu.Unlock()
					return
				}

				app.countdown.SecondsRemaining--

				if app.countdown.SecondsRemaining <= 0 {
					// Countdown finished - start playing
					log.Println("Play countdown finished - starting song")
					app.countdown.Active = false
					app.countdownMu.Unlock()
					app.playNextSongNow()
					return
				}

				app.countdownMu.Unlock()
				app.broadcastState()
			}
		}
	}()
}

// stopCountdown stops the countdown without playing
func (app *App) stopCountdown() {
	app.countdownMu.Lock()
	defer app.countdownMu.Unlock()

	if app.countdownTicker != nil {
		app.countdownTicker.Stop()
		close(app.countdownStop)
		app.countdownTicker = nil
	}

	app.countdown = models.CountdownState{}
}

// playNextSongNow plays the next song immediately (admin action or auto-play)
func (app *App) playNextSongNow() {
	app.countdownMu.Lock()
	if app.countdownTicker != nil {
		app.countdownTicker.Stop()
		close(app.countdownStop)
		app.countdownTicker = nil
	}
	app.countdown = models.CountdownState{}
	app.countdownMu.Unlock()

	// Play the current song in queue
	app.playCurrentSong()
	app.broadcastState()
}

func (app *App) playCurrentSong() {
	// Stop BGM if active
	app.stopBGM()

	song := app.queue.Current()
	if song == nil {
		log.Println("playCurrentSong: no current song in queue")
		return
	}

	log.Printf("Playing: '%s' by '%s' (file: %s)", song.Title, song.Artist, song.VideoURL)

	// Save current singer's avatar to PNG file for external use
	if singer := app.sessions.Get(song.AddedBy); singer != nil && singer.AvatarConfig != nil {
		if avatarPath, err := app.holdingScreen.SaveCurrentSingerAvatar(singer.AvatarConfig); err != nil {
			log.Printf("Failed to save singer avatar: %v", err)
		} else if avatarPath != "" {
			log.Printf("Saved current singer avatar to: %s", avatarPath)
		}
	}

	// Check for CDG+Audio pair first
	if song.CDGPath != "" && song.AudioPath != "" {
		log.Printf("Using CDG+Audio: cdg=%s, audio=%s", song.CDGPath, song.AudioPath)
		if err := app.mpv.LoadCDG(song.CDGPath, song.AudioPath); err != nil {
			log.Printf("Failed to load CDG '%s': %v", song.CDGPath, err)
		}
		return
	}

	// If stems are available, use vocal mixing
	if song.InstrPath != "" && song.VocalPath != "" {
		gain := models.VocalGainMap[song.VocalAssist]
		log.Printf("Using vocal mix: instr=%s, vocal=%s, gain=%.2f", song.InstrPath, song.VocalPath, gain)
		if err := app.mpv.SetVocalMix(song.InstrPath, song.VocalPath, gain); err != nil {
			log.Printf("Failed to set vocal mix: %v", err)
		}
		return
	}

	// Play original video/audio
	if err := app.mpv.LoadFile(song.VideoURL); err != nil {
		log.Printf("Failed to load file '%s': %v", song.VideoURL, err)
	}
}

// updateVocalMix updates the vocal assist level for current playback
func (app *App) updateVocalMix(level models.VocalAssistLevel) {
	song := app.queue.Current()
	if song == nil || song.InstrPath == "" || song.VocalPath == "" {
		return
	}

	gain := models.VocalGainMap[level]
	log.Printf("Updating vocal mix to level %s (gain: %.2f)", level, gain)
	if err := app.mpv.SetVocalMix(song.InstrPath, song.VocalPath, gain); err != nil {
		log.Printf("Failed to update vocal mix: %v", err)
	}
}

// Run starts the HTTP server and WebSocket hub
func (app *App) Run() {
	// Start WebSocket hub
	go app.hub.Run()

	// Start mpv
	mpvReady := false
	if err := app.mpv.Start(); err != nil {
		log.Printf("Warning: Failed to start mpv: %v", err)
		log.Println("Continuing without video playback...")
	} else {
		log.Println("mpv started successfully")
		mpvReady = true
	}

	// Defer playback to after HTTP server starts (avatar API needs to be available)
	go func() {
		// Wait for HTTP server to be ready
		time.Sleep(500 * time.Millisecond)

		if !mpvReady {
			return
		}

		// Resume playback if queue has songs and autoplay is enabled
		if app.queue.Current() != nil && app.queue.GetAutoplay() {
			log.Println("Queue has songs - resuming playback")
			app.playCurrentSong()
		} else {
			// No songs in queue - show holding screen
			log.Println("Queue empty at startup - showing holding screen")
			app.showHoldingScreen()
		}
	}()

	// Setup routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", app.hub.ServeWS)

	// API endpoints
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"0.1.0"}`))
	})

	// Public status endpoint
	mux.HandleFunc("/api/status", app.handleStatus)

	// Feature flags endpoint (public)
	mux.HandleFunc("/api/features", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"youtube_enabled": app.config.YouTubeAPIKey != "",
			"admin_localhost_only": app.admin.IsLocalhostOnly(),
		})
	})

	// Admin API endpoints
	mux.HandleFunc("/api/admin/auth", app.admin.HandleAuth)
	mux.HandleFunc("/api/admin/check", app.admin.HandleCheckAuth)
	mux.HandleFunc("/api/admin/set-pin", app.admin.HandleSetPIN) // localhost only
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

	// Song selection logging endpoint (public - called when user selects a song)
	mux.HandleFunc("/api/library/select", app.handleSongSelection)

	// Search logs endpoints (admin only)
	mux.HandleFunc("/api/admin/search-logs", app.admin.Middleware(app.handleSearchLogs))
	mux.HandleFunc("/api/admin/search-stats", app.admin.Middleware(app.handleSearchStats))
	mux.HandleFunc("/api/admin/song-selections", app.admin.Middleware(app.handleSongSelections))

	// Settings endpoints (admin only)
	mux.HandleFunc("/api/admin/settings", app.admin.Middleware(app.handleSettings))
	mux.HandleFunc("/api/admin/system-info", app.admin.Middleware(app.handleSystemInfo))
	mux.HandleFunc("/api/admin/networks", app.admin.Middleware(app.handleNetworkEnumeration))
	mux.HandleFunc("/api/admin/player", app.admin.Middleware(app.handlePlayer))
	mux.HandleFunc("/api/admin/database", app.admin.Middleware(app.handleDatabase))
	mux.HandleFunc("/api/admin/bgm", app.admin.Middleware(app.handleBGM))
	mux.HandleFunc("/api/admin/icecast-streams", app.admin.Middleware(app.handleIcecastStreams))
	mux.HandleFunc("/api/connect-url", app.handleConnectURL) // Public - returns selected connection URL

	// Avatar API endpoints
	mux.HandleFunc("/api/avatar", handleAvatar)
	mux.HandleFunc("/api/avatar/random", handleAvatarRandom)

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

	// Log admin access mode
	if app.admin.IsLocalhostOnly() {
		log.Printf("Admin access: localhost only (no PIN configured)")
	} else {
		log.Printf("Admin PIN: %s", app.admin.GetPIN())
	}

	// Log YouTube status
	if app.config.YouTubeAPIKey != "" {
		log.Printf("YouTube search: enabled")
	} else {
		log.Printf("YouTube search: disabled (no API key)")
	}

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

// handleAvatar generates an SVG avatar from config parameters
func handleAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()

	// Parse config from query parameters
	config := avatar.Config{
		Env:   parseIntParam(q.Get("env"), 0),
		Clo:   parseIntParam(q.Get("clo"), 0),
		Head:  parseIntParam(q.Get("head"), 0),
		Mouth: parseIntParam(q.Get("mouth"), 0),
		Eyes:  parseIntParam(q.Get("eyes"), 0),
		Top:   parseIntParam(q.Get("top"), 0),
	}

	// Parse custom colors (optional)
	if hasColorParams(q) {
		config.Colors = &avatar.Colors{
			Env:   q.Get("c_env"),
			Clo:   q.Get("c_clo"),
			Head:  q.Get("c_head"),
			Mouth: q.Get("c_mouth"),
			Eyes:  q.Get("c_eyes"),
			Top:   q.Get("c_top"),
		}
	}

	// Check if JSON config is provided
	if configJSON := q.Get("config"); configJSON != "" {
		if parsed, err := avatar.FromJSON(configJSON); err == nil {
			config = parsed
		}
	}

	config.Normalize()

	// Check if we should include environment (background)
	includeEnv := q.Get("noenv") != "true"

	// Generate SVG
	svg := config.ToSVGWithEnv(includeEnv)

	// Set headers for SVG
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 1 day
	w.Write([]byte(svg))
}

// hasColorParams checks if any color parameters are provided
func hasColorParams(q url.Values) bool {
	return q.Get("c_env") != "" || q.Get("c_clo") != "" || q.Get("c_head") != "" ||
		q.Get("c_mouth") != "" || q.Get("c_eyes") != "" || q.Get("c_top") != ""
}

// handleAvatarRandom generates a random avatar config with colors
func handleAvatarRandom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := avatar.NewRandomWithColors()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// parseIntParam parses an int from string with default
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return defaultVal
	}
	return v
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

// handleStatus handles GET /api/status - public system status page
func (app *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type ServiceStatus struct {
		Status string `json:"status"`
	}

	type StatusResponse struct {
		Database    ServiceStatus `json:"database"`
		WebSocket   ServiceStatus `json:"websocket"`
		Library     ServiceStatus `json:"library"`
		MediaPlayer ServiceStatus `json:"media_player"`
		Internet    ServiceStatus `json:"internet"`
	}

	status := StatusResponse{
		Database:    ServiceStatus{Status: "unavailable"},
		WebSocket:   ServiceStatus{Status: "unavailable"},
		Library:     ServiceStatus{Status: "unavailable"},
		MediaPlayer: ServiceStatus{Status: "unavailable"},
		Internet:    ServiceStatus{Status: "unavailable"},
	}

	// Check database (via library stats)
	if _, _, err := app.library.GetStats(); err == nil {
		status.Database.Status = "connected"
	}

	// Check WebSocket hub
	if app.hub != nil {
		status.WebSocket.Status = "connected"
	}

	// Check library (has locations configured)
	if locations, err := app.library.GetLocations(); err == nil && len(locations) > 0 {
		status.Library.Status = "connected"
	}

	// Check media player (MPV)
	if app.mpv != nil {
		if _, err := app.mpv.GetState(); err == nil {
			status.MediaPlayer.Status = "connected"
		}
	}

	// Check internet connectivity (quick timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Head("https://www.google.com"); err == nil {
		resp.Body.Close()
		status.Internet.Status = "connected"
	}

	json.NewEncoder(w).Encode(status)
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

// handleSongSelection handles POST /api/library/select - logs when a user selects a song
func (app *App) handleSongSelection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SongID      string `json:"song_id"`
		SongTitle   string `json:"song_title"`
		SongArtist  string `json:"song_artist"`
		Source      string `json:"source"`
		SearchQuery string `json:"search_query"`
		MartynKey   string `json:"martyn_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	ipAddress := getClientIP(r)
	if err := app.library.LogSongSelection(req.SongID, req.SongTitle, req.SongArtist, req.Source, req.SearchQuery, req.MartynKey, ipAddress); err != nil {
		log.Printf("Failed to log song selection: %v", err)
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "YouTube search is not available",
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

// handleSongSelections handles GET /api/admin/song-selections
func (app *App) handleSongSelections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	source := r.URL.Query().Get("source")
	selections, err := app.library.GetSongSelections(100, source)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if selections == nil {
		selections = []library.SongSelection{}
	}
	json.NewEncoder(w).Encode(selections)
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

// SettingsPayload represents the settings that can be updated
type SettingsPayload struct {
	HTTPSPort     string `json:"https_port"`
	HTTPPort      string `json:"http_port"`
	AdminPIN      string `json:"admin_pin"`
	YouTubeAPIKey string `json:"youtube_api_key"`
	VideoPlayer   string `json:"video_player"`
	DataDir       string `json:"data_dir"`
}

// handleSettings handles GET/POST /api/admin/settings
func (app *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	envPath := ".env"

	switch r.Method {
	case http.MethodGet:
		// Return current settings (mask sensitive values for display)
		settings := SettingsPayload{
			HTTPSPort:     app.config.Port,
			HTTPPort:      app.config.HTTPPort,
			AdminPIN:      app.config.AdminPIN,
			YouTubeAPIKey: app.config.YouTubeAPIKey,
			VideoPlayer:   app.config.VideoPlayer,
			DataDir:       app.config.DataDir,
		}
		json.NewEncoder(w).Encode(settings)

	case http.MethodPost:
		var settings SettingsPayload
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
			return
		}

		// Check if PIN changed - if so, immediately invalidate non-local admin sessions
		pinChanged := settings.AdminPIN != app.config.AdminPIN
		if pinChanged {
			app.admin.SetPIN(settings.AdminPIN)
			app.config.AdminPIN = settings.AdminPIN
			log.Printf("Admin PIN changed - all non-local admin sessions have been invalidated")
		}

		// Build .env content
		envContent := fmt.Sprintf(`# SongMartyn Configuration
# Updated via admin panel

HTTPS_PORT=%s
HTTP_PORT=%s
ADMIN_PIN=%s
YOUTUBE_API_KEY=%s
VIDEO_PLAYER=%s
DATA_DIR=%s
TLS_CERT=%s
TLS_KEY=%s
`, settings.HTTPSPort, settings.HTTPPort, settings.AdminPIN, settings.YouTubeAPIKey,
			settings.VideoPlayer, settings.DataDir, app.config.CertFile, app.config.KeyFile)

		// Write .env file
		if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save settings: " + err.Error()})
			return
		}

		message := "Settings saved. Restart the server for changes to take effect."
		if pinChanged {
			message = "Settings saved. PIN changed - all non-local admin sessions have been invalidated. Restart server for other changes."
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"message":     message,
			"pin_changed": pinChanged,
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// SystemInfo represents system information
type SystemInfo struct {
	OS           string  `json:"os"`
	Arch         string  `json:"arch"`
	Hostname     string  `json:"hostname"`
	CPUCount     int     `json:"cpu_count"`
	MemoryTotal  uint64  `json:"memory_total"`
	MemoryFree   uint64  `json:"memory_free"`
	MemoryUsed   uint64  `json:"memory_used"`
	DiskTotal    uint64  `json:"disk_total"`
	DiskFree     uint64  `json:"disk_free"`
	DiskUsed     uint64  `json:"disk_used"`
	GoVersion    string  `json:"go_version"`
	ServerUptime string  `json:"server_uptime"`
	NetworkAddrs []string `json:"network_addrs"`
}

var serverStartTime = time.Now()

// handleSystemInfo handles GET /api/admin/system-info
func (app *App) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	hostname, _ := os.Hostname()
	uptime := time.Since(serverStartTime).Round(time.Second).String()

	info := SystemInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Hostname:     hostname,
		CPUCount:     runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		ServerUptime: uptime,
		NetworkAddrs: getNetworkAddresses(),
	}

	// Get memory info (platform-specific)
	info.MemoryTotal, info.MemoryFree, info.MemoryUsed = getMemoryInfo()

	// Get disk info
	info.DiskTotal, info.DiskFree, info.DiskUsed = getDiskInfo(app.config.DataDir)

	json.NewEncoder(w).Encode(info)
}

// getNetworkAddresses returns all network interface addresses
func getNetworkAddresses() []string {
	var addrs []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return addrs
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		ifAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range ifAddrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip IPv6 link-local and loopback
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}

			addrs = append(addrs, fmt.Sprintf("%s (%s)", ip.String(), iface.Name))
		}
	}
	return addrs
}

// getMemoryInfo returns memory statistics (cross-platform basic implementation)
func getMemoryInfo() (total, free, used uint64) {
	// This is a simplified implementation
	// For more accurate info, consider using github.com/shirou/gopsutil
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// These are Go runtime stats, not system-wide
	// Return what we can
	used = m.Sys // Total memory obtained from the OS
	return 0, 0, used
}

// getDiskInfo returns disk statistics for a given path
func getDiskInfo(path string) (total, free, used uint64) {
	// Use syscall for disk info - works on Unix-like systems
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}

	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bavail * uint64(stat.Bsize)
	used = total - free
	return
}

// NetworkInterface represents a network interface with its addresses
type NetworkInterface struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Type         string   `json:"type"`
	MacAddress   string   `json:"mac_address"`
	IPv4         []string `json:"ipv4"`
	IPv6         []string `json:"ipv6"`
	IsUp         bool     `json:"is_up"`
	IsLoopback   bool     `json:"is_loopback"`
	IsWireless   bool     `json:"is_wireless"`
	ConnectURLs  []string `json:"connect_urls"`
}

// handleNetworkEnumeration handles GET /api/admin/networks
func (app *App) handleNetworkEnumeration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var result []NetworkInterface
	for _, iface := range interfaces {
		ni := NetworkInterface{
			Name:       iface.Name,
			MacAddress: iface.HardwareAddr.String(),
			IsUp:       iface.Flags&net.FlagUp != 0,
			IsLoopback: iface.Flags&net.FlagLoopback != 0,
			IPv4:       []string{},
			IPv6:       []string{},
			ConnectURLs: []string{},
		}

		// Generate display name
		ni.DisplayName = getInterfaceDisplayName(iface.Name)
		ni.Type = getInterfaceType(iface.Name)
		ni.IsWireless = isWirelessInterface(iface.Name)

		// Get addresses
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLinkLocalUnicast() {
				continue
			}

			if ip.To4() != nil {
				ni.IPv4 = append(ni.IPv4, ip.String())
				// Generate connect URL for non-loopback IPv4
				if !ni.IsLoopback {
					connectURL := fmt.Sprintf("https://%s:%s", ip.String(), app.config.Port)
					ni.ConnectURLs = append(ni.ConnectURLs, connectURL)
				}
			} else {
				ni.IPv6 = append(ni.IPv6, ip.String())
			}
		}

		// Only include interfaces that are up and have addresses
		if ni.IsUp && (len(ni.IPv4) > 0 || len(ni.IPv6) > 0) {
			result = append(result, ni)
		}
	}

	json.NewEncoder(w).Encode(result)
}

// getInterfaceDisplayName returns a friendly name for the interface
func getInterfaceDisplayName(name string) string {
	// Common interface name patterns
	switch {
	case strings.HasPrefix(name, "en"):
		if strings.HasPrefix(name, "en0") {
			return "Wi-Fi / Ethernet (Primary)"
		}
		return "Ethernet"
	case strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "wlp"):
		return "Wi-Fi"
	case strings.HasPrefix(name, "eth"):
		return "Ethernet"
	case strings.HasPrefix(name, "lo"):
		return "Loopback"
	case strings.HasPrefix(name, "docker"):
		return "Docker Network"
	case strings.HasPrefix(name, "br-"):
		return "Bridge Network"
	case strings.HasPrefix(name, "veth"):
		return "Virtual Ethernet"
	case strings.HasPrefix(name, "utun"):
		return "VPN Tunnel"
	case strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "tap"):
		return "VPN"
	case strings.HasPrefix(name, "awdl"):
		return "Apple Wireless Direct Link"
	case strings.HasPrefix(name, "bridge"):
		return "Network Bridge"
	default:
		return name
	}
}

// getInterfaceType returns the type of interface
func getInterfaceType(name string) string {
	switch {
	case strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "wlp"):
		return "wireless"
	case strings.HasPrefix(name, "en0") && runtime.GOOS == "darwin":
		return "wifi_or_ethernet"
	case strings.HasPrefix(name, "en") || strings.HasPrefix(name, "eth"):
		return "ethernet"
	case strings.HasPrefix(name, "lo"):
		return "loopback"
	case strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "veth"):
		return "virtual"
	case strings.HasPrefix(name, "utun") || strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "tap"):
		return "vpn"
	default:
		return "other"
	}
}

// isWirelessInterface checks if the interface is wireless
func isWirelessInterface(name string) bool {
	return strings.HasPrefix(name, "wlan") ||
		strings.HasPrefix(name, "wlp") ||
		(strings.HasPrefix(name, "en0") && runtime.GOOS == "darwin") // macOS primary is often Wi-Fi
}

// handleConnectURL handles GET/POST /api/connect-url
// GET returns the selected connection URL for QR codes
// POST (admin only) sets the preferred connection URL
func (app *App) handleConnectURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	connectURLFile := filepath.Join(app.config.DataDir, "connect_url.txt")

	switch r.Method {
	case http.MethodGet:
		// Return stored URL or auto-detect
		data, err := os.ReadFile(connectURLFile)
		if err == nil && len(data) > 0 {
			json.NewEncoder(w).Encode(map[string]string{"url": strings.TrimSpace(string(data))})
			return
		}

		// Auto-detect: find first non-loopback IPv4 address
		url := app.autoDetectConnectURL()
		json.NewEncoder(w).Encode(map[string]string{"url": url})

	case http.MethodPost:
		// Check admin auth for POST
		if !app.admin.IsAuthorized(r) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
			return
		}

		// Save to file
		if err := os.WriteFile(connectURLFile, []byte(req.URL), 0644); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "url": req.URL})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// autoDetectConnectURL finds the best connection URL
func (app *App) autoDetectConnectURL() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Sprintf("https://localhost:%s", app.config.Port)
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Only IPv4, non-loopback
			if ip != nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				return fmt.Sprintf("https://%s:%s", ip.String(), app.config.Port)
			}
		}
	}

	return fmt.Sprintf("https://localhost:%s", app.config.Port)
}

// handlePlayer handles GET/POST /api/admin/player
func (app *App) handlePlayer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Return player status
		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_running": app.mpv.IsRunning(),
		})

	case http.MethodPost:
		// Launch or restart the player
		var req struct {
			Action string `json:"action"` // "launch" or "restart"
		}
		json.NewDecoder(r.Body).Decode(&req)

		if req.Action == "" {
			req.Action = "launch"
		}

		var err error
		switch req.Action {
		case "restart":
			err = app.mpv.Restart()
		case "launch":
			if app.mpv.IsRunning() {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":     "ok",
					"message":    "Player is already running",
					"is_running": true,
				})
				return
			}
			err = app.mpv.Start()
		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid action"})
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to " + req.Action + " player: " + err.Error()})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"message":    "Player " + req.Action + "ed successfully",
			"is_running": app.mpv.IsRunning(),
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleDatabase handles GET/POST /api/admin/database - database management
func (app *App) handleDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Return database stats
		queueState := app.queue.GetState()
		stats := map[string]interface{}{
			"sessions": map[string]interface{}{
				"count": app.sessions.GetSessionCount(),
			},
			"blocked_users": map[string]interface{}{
				"count": app.sessions.GetBlockedUserCount(),
			},
			"search_logs": map[string]interface{}{
				"count": app.library.GetSearchLogCount(),
			},
			"song_history": map[string]interface{}{
				"count": app.library.GetSongHistoryCount(),
			},
			"queue": map[string]interface{}{
				"count": len(queueState.Songs),
			},
		}
		json.NewEncoder(w).Encode(stats)

	case http.MethodPost:
		// Perform database operation
		var req struct {
			Action string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
			return
		}

		var err error
		var message string

		switch req.Action {
		case "flush_sessions":
			err = app.sessions.FlushSessions()
			message = "All sessions cleared"
			// Disconnect all WebSocket clients
			app.hub.DisconnectAll("Session data cleared by admin")

		case "flush_blocked":
			err = app.sessions.FlushBlockedUsers()
			message = "All blocked users cleared"

		case "flush_queue":
			app.queue.Clear()
			message = "Queue cleared"

		case "flush_search_logs":
			err = app.library.ClearSearchLogs()
			message = "Search logs cleared"

		case "flush_song_history":
			err = app.library.ClearSongHistory()
			message = "Song history cleared"

		case "flush_all":
			// Flush everything except library songs
			app.sessions.FlushSessions()
			app.sessions.FlushBlockedUsers()
			app.queue.Clear()
			app.library.ClearSearchLogs()
			app.library.ClearSongHistory()
			app.library.ClearSongSelections()
			app.hub.DisconnectAll("All data cleared by admin")
			message = "All data cleared (library songs preserved)"

		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unknown action: " + req.Action})
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": message,
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleBGM handles GET/POST /api/admin/bgm - background music settings
func (app *App) handleBGM(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Return current BGM settings
		json.NewEncoder(w).Encode(app.bgmSettings)

	case http.MethodPost:
		// Update BGM settings
		var settings models.BGMSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
			return
		}

		// Default to YouTube if source type not specified
		if settings.SourceType == "" {
			settings.SourceType = models.BGMSourceYouTube
		}

		// Update settings
		app.bgmSettings = settings
		log.Printf("BGM settings updated: enabled=%v, source=%s, url=%s, volume=%.0f",
			settings.Enabled, settings.SourceType, settings.URL, settings.Volume)

		// If BGM was disabled, stop any active BGM
		if !settings.Enabled && app.bgmActive {
			app.stopBGM()
			app.broadcastState()
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "ok",
			"settings": app.bgmSettings,
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleIcecastStreams handles GET /api/admin/icecast-streams - returns popular Icecast music streams
func (app *App) handleIcecastStreams(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Curated list of popular music-only Icecast streams (no presenters/talk)
	// These are reliable, high-quality streams focused on instrumental/background music
	streams := []models.IcecastStream{
		// SomaFM streams - excellent quality, no ads, instrumental/ambient
		{
			Name:        "SomaFM - Groove Salad",
			URL:         "https://ice1.somafm.com/groovesalad-128-mp3",
			Genre:       "Ambient/Downtempo",
			Description: "Chill ambient/downtempo beats, perfect background music",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Drone Zone",
			URL:         "https://ice1.somafm.com/dronezone-128-mp3",
			Genre:       "Ambient/Space",
			Description: "Ambient soundscapes for space exploration",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Space Station",
			URL:         "https://ice1.somafm.com/spacestation-128-mp3",
			Genre:       "Ambient/Electronic",
			Description: "Spaced-out ambient and mid-tempo electronic",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Deep Space One",
			URL:         "https://ice1.somafm.com/deepspaceone-128-mp3",
			Genre:       "Ambient/Experimental",
			Description: "Deep ambient electronic exploration",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Suburbs of Goa",
			URL:         "https://ice1.somafm.com/suburbsofgoa-128-mp3",
			Genre:       "World/Ambient",
			Description: "Desi-influenced Asian world beats and beyond",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Lush",
			URL:         "https://ice1.somafm.com/lush-128-mp3",
			Genre:       "Electronic/Downtempo",
			Description: "Sensuous, mellow electronic vocals",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Boot Liquor",
			URL:         "https://ice1.somafm.com/bootliquor-128-mp3",
			Genre:       "Americana/Country",
			Description: "Americana roots music for dusty back roads",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Secret Agent",
			URL:         "https://ice1.somafm.com/secretagent-128-mp3",
			Genre:       "Lounge/Jazz",
			Description: "Spy-themed lounge, exotica and jazz",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Illinois Street Lounge",
			URL:         "https://ice1.somafm.com/illstreet-128-mp3",
			Genre:       "Lounge/Exotica",
			Description: "Classic bachelor pad, retro-modern exotica",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Cliqhop IDM",
			URL:         "https://ice1.somafm.com/cliqhop-128-mp3",
			Genre:       "Electronic/IDM",
			Description: "Blips and bleeps with a beat",
			Bitrate:     128,
			Format:      "MP3",
		},
		// Radio Paradise - high quality eclectic mix
		{
			Name:        "Radio Paradise - Main Mix",
			URL:         "https://stream.radioparadise.com/aac-128",
			Genre:       "Eclectic/Rock",
			Description: "Eclectic mix of rock, world, and electronic",
			Bitrate:     128,
			Format:      "AAC",
		},
		{
			Name:        "Radio Paradise - Mellow",
			URL:         "https://stream.radioparadise.com/mellow-128",
			Genre:       "Mellow/Acoustic",
			Description: "Softer, acoustic selections",
			Bitrate:     128,
			Format:      "AAC",
		},
		// Jazz streams
		{
			Name:        "SomaFM - Jazz Riot",
			URL:         "https://ice1.somafm.com/jazzriot-128-mp3",
			Genre:       "Jazz",
			Description: "Jazz from the classic era to modern",
			Bitrate:     128,
			Format:      "MP3",
		},
		// Classical
		{
			Name:        "WCPE Classical",
			URL:         "https://audio-ogg.ibiblio.org:8000/wcpe.ogg",
			Genre:       "Classical",
			Description: "24/7 classical music, no interruptions",
			Bitrate:     128,
			Format:      "OGG",
		},
		// Chillout/Lo-fi
		{
			Name:        "SomaFM - DEF CON Radio",
			URL:         "https://ice1.somafm.com/defcon-128-mp3",
			Genre:       "Electronic/Hacker",
			Description: "Music for hacking - electronic and synth",
			Bitrate:     128,
			Format:      "MP3",
		},
		{
			Name:        "SomaFM - Beat Blender",
			URL:         "https://ice1.somafm.com/beatblender-128-mp3",
			Genre:       "Electronic/Trip-hop",
			Description: "Deep house and downtempo chill",
			Bitrate:     128,
			Format:      "MP3",
		},
	}

	json.NewEncoder(w).Encode(streams)
}
