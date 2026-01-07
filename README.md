# SongMartyn

**The ultimate self-hosted karaoke party system** — Let guests queue songs from their phones while you focus on the fun.

SongMartyn turns any computer with a TV into a professional karaoke setup. Guests scan a QR code to browse your library and queue songs, while you control the show from the admin panel.

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [System Requirements](#system-requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage Guide](#usage-guide)
- [Admin Panel](#admin-panel)
- [Technical Specification](#technical-specification)
- [Roadmap](#roadmap)

---

## Overview

### How It Works

```
┌─────────────────────────────────────────────────────────────────────┐
│                         LOCAL NETWORK                                │
│                                                                      │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐                          │
│   │  Guest   │  │  Guest   │  │  Admin   │                          │
│   │  Phone   │  │  Tablet  │  │  Phone   │                          │
│   └────┬─────┘  └────┬─────┘  └────┬─────┘                          │
│        │             │             │                                 │
│        └─────────────┼─────────────┘                                 │
│                      │ WebSocket (wss://)                            │
│                      ▼                                               │
│              ┌───────────────┐         ┌─────────────────┐          │
│              │  SongMartyn   │────────►│  TV/Projector   │          │
│              │   Server      │   mpv   │  (Main Display) │          │
│              │   (Go)        │         └─────────────────┘          │
│              └───────────────┘                                       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

1. **Host** runs SongMartyn on a computer connected to a TV/projector
2. **Guests** scan a QR code to open the web interface on their phones
3. **Everyone** can browse the library, search for songs, and add to the queue
4. **Songs play** on the main display with lyrics/video
5. **Admins** control playback, manage users, and moderate the queue

### Key Principles

- **No App Required** — Guests use their phone's browser
- **Self-Hosted** — Your music, your network, your control
- **Real-Time** — All clients stay in sync via WebSocket
- **Bring Your Own Library** — Use your existing karaoke files

---

## Features

### Guest Experience

| Feature | Description |
|---------|-------------|
| **QR Code Join** | Scan to connect instantly, no app download needed |
| **Song Search** | Search your library by title or artist |
| **Queue Songs** | Add songs to the shared queue |
| **View Queue** | See what's coming up and your position |
| **Now Playing** | See current song with progress bar |
| **Personal Settings** | Set your display name and avatar |

### Queue System

| Feature | Description |
|---------|-------------|
| **Shared Queue** | Single queue visible to all connected users |
| **Fair Ordering** | Songs play in the order they were added |
| **Remove Own Songs** | Users can remove their own queued songs |
| **Requeue** | Add a song you've already sung back to the queue |
| **Autoplay** | Optionally auto-advance through the queue |
| **Shuffle** | Admin can shuffle the upcoming queue |

### Playback

| Feature | Description |
|---------|-------------|
| **Countdown Timer** | 10-second countdown before each song starts |
| **Admin Play Control** | When autoplay is off, admin must press Play to start countdown |
| **Skip Song** | Admin can skip the current song |
| **Stop Playback** | Admin can stop and return to holding screen |
| **Volume Control** | Adjust playback volume |
| **Seek** | Jump to any position in the current song |

### User Management

| Feature | Description |
|---------|-------------|
| **Automatic Sessions** | Users get a persistent identity via browser storage |
| **Custom Avatars** | Multiavatar system with customizable parts and colors |
| **Display Names** | Users can set their own name |
| **AFK Status** | Mark users as Away From Keyboard |
| **AFK Song Bump** | AFK users' songs move to end of queue |
| **Kick Users** | Disconnect a user (they can rejoin) |
| **Block Users** | Temporarily or permanently ban a user |
| **Admin Promotion** | Grant admin privileges to trusted users (requires PIN) |

### Holding Screen (Main Display)

When no song is playing, the TV shows:

| Element | Description |
|---------|-------------|
| **QR Code** | Guests scan to connect |
| **Connection URL** | Text URL for manual entry |
| **Next Up** | Shows next singer's name and song |
| **Singer Avatar** | Visual identification of who's next |
| **Custom Message** | Optional venue branding/message (planned) |
| **Custom Logo** | Optional venue logo (planned) |

### Music Library

| Feature | Description |
|---------|-------------|
| **Multiple Locations** | Add multiple folders to your library |
| **Auto-Scan** | Scan folders for supported media files |
| **Format Support** | MP4, WebM, MKV, AVI, MP3, M4A, WAV, FLAC, OGG |
| **CDG Support** | CD+Graphics karaoke format with paired audio |
| **Metadata** | Title, artist, duration tracking |
| **Play Statistics** | Track how often each song is played |
| **Popular Songs** | View most-played songs |

### YouTube Integration (Optional)

| Feature | Description |
|---------|-------------|
| **Search YouTube** | Find karaoke videos on YouTube |
| **Queue YouTube Videos** | Add YouTube videos to the queue |
| **Requires API Key** | Must configure YouTube Data API key |

### Background Music (BGM)

| Feature | Description |
|---------|-------------|
| **Idle Music** | Play music when queue is empty |
| **YouTube Source** | Use a YouTube URL as BGM |
| **Icecast Streams** | Use internet radio as BGM |
| **Separate Volume** | BGM volume independent of karaoke |

### Security

| Feature | Description |
|---------|-------------|
| **HTTPS Required** | All connections encrypted with TLS |
| **Admin PIN** | Remote admin access requires PIN |
| **Localhost Auto-Auth** | Admin panel auto-authenticates from server machine |
| **PIN Confirmation** | Promoting users to admin requires PIN re-entry |

---

## System Requirements

### Hardware

- **Computer**: PC, Mac, or Linux machine capable of driving 2 displays
  - Display 1: Admin/control screen
  - Display 2: TV/projector for karaoke output
- **Network**: WiFi router (guests connect via local network)
- **Audio**: Speakers/PA system connected to the computer

### Software

- **Operating System**: Windows 10+, macOS 10.15+, or Linux
- **mpv**: Media player (required for video playback)
- **Go 1.21+**: For building from source
- **Node.js 18+**: For building the frontend

### Network

- All devices must be on the **same local network**
- SongMartyn does not support internet/remote access
- HTTPS is required (self-signed certificates work)

---

## Installation

### Download Pre-Built Binary

```bash
# Coming soon - check releases page
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/songmartyn.git
cd songmartyn

# Build the backend
cd backend
go build -o songmartyn ./cmd/songmartyn

# Build the frontend
cd ../frontend
npm install
npm run build

# Copy frontend build to backend
cp -r dist ../backend/

# Generate TLS certificates (first run)
cd ../backend
mkdir -p certs
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/key.pem \
  -out certs/cert.pem \
  -days 365 -nodes \
  -subj "/CN=SongMartyn"

# Run
./songmartyn
```

### Install mpv

**macOS:**
```bash
brew install mpv
```

**Ubuntu/Debian:**
```bash
sudo apt install mpv
```

**Windows:**
Download from https://mpv.io/installation/

---

## Configuration

### Environment Variables

Create a `.env` file in the backend directory:

```env
# Server Ports
HTTPS_PORT=8443
HTTP_PORT=8080

# TLS Certificates
TLS_CERT=./certs/cert.pem
TLS_KEY=./certs/key.pem

# Admin PIN (leave empty for localhost-only admin access)
ADMIN_PIN=

# YouTube API Key (optional - enables YouTube search)
YOUTUBE_API_KEY=

# mpv Player Path (auto-detected if not set)
VIDEO_PLAYER=

# Data Directory (defaults to ./data)
DATA_DIR=./data
```

### First Run

1. Start SongMartyn: `./songmartyn`
2. Open `https://localhost:8443/admin` in your browser
3. Accept the self-signed certificate warning
4. Add your music library folders in Settings
5. Scan the folders to index your songs

---

## Usage Guide

### Starting a Karaoke Session

1. **Launch SongMartyn** on your host computer
2. **Open the main display** — Navigate to `https://<your-ip>:8443` on the TV
3. **Show the QR code** — Click the QR button to display the join code
4. **Guests scan and join** — They'll see the song library on their phones
5. **Queue songs** — Guests browse and add songs to the queue
6. **Press Play** — Admin starts the first song (if autoplay is off)
7. **Enjoy!** — Songs play automatically with countdown between singers

### For Guests

1. **Scan the QR code** on the TV screen
2. **Accept the security warning** (self-signed certificate)
3. **Set your name** (optional) — Tap the avatar icon
4. **Search for songs** — Use the search bar
5. **Queue a song** — Tap a song, select vocal assist level, tap "Add to Queue"
6. **Wait your turn** — Watch the queue for your position
7. **Sing!** — When your song comes up, take the mic

### For Admins

1. **Access the admin panel** — Go to `/admin` on the host machine (auto-login) or enter PIN from another device
2. **Manage the queue** — Skip songs, reorder, clear
3. **Manage users** — Kick troublemakers, promote co-hosts
4. **Control playback** — Play, pause, stop, volume
5. **Add music** — Scan new folders into the library

---

## Admin Panel

### Tabs

| Tab | Purpose |
|-----|---------|
| **Queue** | View and manage the song queue |
| **Clients** | See connected users, manage permissions |
| **Library** | Add folders, scan for songs, view stats |
| **Search Logs** | See what users are searching for |
| **Network** | Select which network interface to advertise |
| **Settings** | Configure server, YouTube API, BGM |

### Client Management

| Action | Description |
|--------|-------------|
| **Make Admin** | Promote user to admin (requires PIN confirmation) |
| **Remove Admin** | Demote admin back to regular user |
| **Set AFK** | Mark user as away, bumps their songs to end |
| **Edit Name** | Change a user's display name |
| **Lock Name** | Prevent user from changing their own name |
| **Kick** | Disconnect user (removes their songs from queue) |
| **Block** | Ban user for duration (5 min to permanent) |
| **Unblock** | Remove a user's block |

### Queue Management

| Action | Description |
|--------|-------------|
| **Play Next** | Start countdown for next song |
| **Start Now** | Skip countdown, play immediately |
| **Stop** | Stop playback, show holding screen |
| **Skip** | Skip current song, advance queue |
| **Remove** | Remove a song from queue |
| **Move** | Reorder songs in queue |
| **Shuffle** | Randomize upcoming songs |
| **Clear** | Remove all songs from queue |

---

## Technical Specification

### Architecture

```
backend/
├── cmd/songmartyn/     # Main application entry point
├── internal/
│   ├── admin/          # Admin authentication
│   ├── config/         # Configuration management
│   ├── holdingscreen/  # QR code and idle screen generation
│   ├── library/        # Music library management
│   ├── mpv/            # Media player integration
│   ├── queue/          # Song queue management
│   ├── session/        # User session management
│   └── websocket/      # Real-time communication
└── pkg/models/         # Shared data models

frontend/
├── src/
│   ├── components/     # Reusable UI components
│   ├── pages/          # Main application pages
│   ├── services/       # WebSocket client
│   ├── stores/         # Zustand state stores
│   └── types/          # TypeScript definitions
```

### Data Storage

| Database | Purpose |
|----------|---------|
| `sessions.db` | User sessions, avatars, preferences |
| `library.db` | Song metadata, statistics, locations |
| `queue.db` | Current queue state, history |

### WebSocket Messages

#### Client → Server

| Message | Payload | Description |
|---------|---------|-------------|
| `handshake` | `{martyn_key?, display_name?}` | Initial connection |
| `queue_add` | `{song_id, vocal_assist}` | Add song to queue |
| `queue_remove` | `{song_id, martyn_key}` | Remove song from queue |
| `queue_move` | `{song_id, martyn_key, position}` | Move song in queue |
| `queue_clear` | — | Clear the queue |
| `queue_shuffle` | — | Shuffle upcoming songs |
| `queue_requeue` | `{song_id, martyn_key}` | Requeue a song |
| `play` | — | Resume playback |
| `pause` | — | Pause playback |
| `skip` | — | Skip current song |
| `seek` | `{position}` | Seek to position |
| `volume` | `{level}` | Set volume (0-100) |
| `vocal_assist` | `{level}` | Set vocal assist level |
| `autoplay` | `{enabled}` | Toggle autoplay |
| `set_display_name` | `{name}` | Update display name |
| `set_avatar` | `{config}` | Update avatar |
| `set_afk` | `{is_afk}` | Set own AFK status |

#### Admin → Server

| Message | Payload | Description |
|---------|---------|-------------|
| `admin_set_admin` | `{martyn_key, is_admin}` | Promote/demote user |
| `admin_kick` | `{martyn_key, reason?}` | Kick user |
| `admin_block` | `{martyn_key, duration, reason?}` | Block user |
| `admin_unblock` | `{martyn_key}` | Unblock user |
| `admin_set_afk` | `{martyn_key, is_afk}` | Set user's AFK status |
| `admin_set_name` | `{martyn_key, display_name}` | Change user's name |
| `admin_set_name_lock` | `{martyn_key, locked}` | Lock user's name |
| `admin_play_next` | — | Start countdown |
| `admin_start_now` | — | Play immediately |
| `admin_stop` | — | Stop playback |

#### Server → Client

| Message | Payload | Description |
|---------|---------|-------------|
| `welcome` | `{session, state}` | Connection confirmed |
| `state_update` | `{queue, player, ...}` | Full state update |
| `client_list` | `[{client_info}]` | Admin: connected users |
| `search_result` | `{results}` | Search results |
| `error` | `{error}` | Error message |
| `kicked` | `{reason}` | User was kicked |

### API Endpoints

#### Public

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/status` | System status |
| GET | `/api/features` | Feature flags |
| GET | `/api/connect-url` | Get connection URL |
| GET | `/api/avatar` | Generate avatar SVG |
| GET | `/api/avatar/random` | Random avatar |
| GET | `/api/library/search?q=` | Search library |
| GET | `/api/library/stats` | Library statistics |
| GET | `/api/library/popular` | Popular songs |
| GET | `/api/youtube/search?q=` | Search YouTube |
| WS | `/ws` | WebSocket connection |

#### Admin (requires authentication)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/admin/check` | Check auth status |
| POST | `/api/admin/auth` | Authenticate with PIN |
| GET | `/api/admin/clients` | List all clients |
| POST | `/api/admin/clients/:key/admin` | Set admin status |
| POST | `/api/admin/clients/:key/block` | Block user |
| DELETE | `/api/admin/clients/:key/block` | Unblock user |
| GET | `/api/admin/settings` | Get settings |
| POST | `/api/admin/settings` | Update settings |
| GET | `/api/admin/networks` | List network interfaces |
| GET | `/api/library/locations` | List library locations |
| POST | `/api/library/locations` | Add location |
| POST | `/api/library/locations/:id/scan` | Scan location |

---

## Roadmap

### Planned Features

| Feature | Description | Priority |
|---------|-------------|----------|
| **Custom Branding** | Custom logo, colors, and message on holding screen | High |
| **Telemetry Dashboard** | Analytics on usage, popular songs, etc. | Medium |
| **Lyrics Display** | Show synced lyrics during playback | Low |
| **Audio Separation** | Integrate Demucs/Spleeter for vocal removal | Low |
| **Chromecast Support** | Cast to Chromecast devices | Low |
| **Duet Mode** | Split-screen lyrics for duets | Low |

### Not Planned

| Feature | Reason |
|---------|--------|
| Native mobile apps | Web app provides full functionality, no app store friction |
| Multi-room support | Complexity vs demand |
| Internet/remote access | Security concerns, local-only by design |
| Song downloads | Legal concerns |

---

## Support

- **Issues**: Report bugs via GitHub Issues
- **Documentation**: See `/docs` folder for detailed guides
- **Community**: Join our Discord (coming soon)

---

## License

Source Available License — See [LICENSE](LICENSE) for details.

You may view, fork, and modify this code for personal use. Commercial use requires a license.

---

**SongMartyn** — *Making karaoke nights unforgettable.*
