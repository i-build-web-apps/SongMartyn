# SongMartyn

**The ultimate self-hosted karaoke party system** - Turn any device into a karaoke machine with real-time multiplayer support, intelligent vocal assistance, and a beautiful mobile-first interface.

## Features

### Party-Ready Experience
- **Instant Guest Access** - Guests scan a QR code to join instantly, no app installation required
- **Auto-Generated Singer Names** - Fun randomly-generated names (famous musician first + last names) for anonymous guests
- **Pixel Avatars** - Choose from 8 unique pixel art avatars to personalize your profile
- **Real-Time Queue** - Everyone sees the same queue, updated instantly across all devices
- **Mobile-First Design** - Beautiful dark theme UI optimized for phones and tablets

### Vocal Assist Technology (The Chortle)
- **Off** - Pure karaoke, you're on your own!
- **Pitch** - Subtle pitch reference (15% vocal guide)
- **Melody** - Moderate melody support (45% vocal guide)
- **Full** - Full vocal lead for learning (80% vocal guide)
- **Per-Song Settings** - Each singer can set their own vocal assist preference

### Admin Control Center
- **PIN-Protected Remote Access** - Secure admin panel with automatic localhost authentication
- **Connected Clients View** - See all connected devices with IP, device type, and online status
- **Network Configuration** - Select which network address to advertise via QR code
- **Library Management** - Add folders, scan for media, view song statistics
- **Search Analytics** - Track what guests are searching for, identify missing songs

### Technical Excellence
- **WebSocket Real-Time Sync** - Sub-second updates across all connected clients
- **Session Persistence** - The "Martyn Handshake" remembers returning guests
- **YouTube Integration** - Search and queue directly from YouTube (optional API key)
- **Local Library Support** - Play from your own music/video collection
- **TLS by Default** - Secure HTTPS/WSS connections out of the box

## Architecture

```
┌─────────────────┐     WebSocket      ┌──────────────────┐
│   Guest Phone   │◄──────────────────►│                  │
└─────────────────┘                    │   SongMartyn     │
                                       │   Backend (Go)   │
┌─────────────────┐     WebSocket      │                  │
│   Guest Tablet  │◄──────────────────►│   - Queue Mgmt   │
└─────────────────┘                    │   - MPV Player   │
                                       │   - Audio Split  │
┌─────────────────┐     WebSocket      │   - Session Mgr  │
│   Admin Panel   │◄──────────────────►│                  │
└─────────────────┘                    └────────┬─────────┘
                                                │
                                       ┌────────▼─────────┐
                                       │   Video Output   │
                                       │   (TV/Projector) │
                                       └──────────────────┘
```

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- mpv media player (optional, for video playback)

### Installation

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

# Generate TLS certificates (first run only)
cd ../backend
mkdir -p certs
openssl req -x509 -newkey rsa:4096 -keyout certs/key.pem -out certs/cert.pem -days 365 -nodes -subj "/CN=localhost"

# Run
./songmartyn
```

### Configuration

Create a `.env` file in the backend directory:

```env
# Admin PIN (leave empty for localhost-only admin access)
ADMIN_PIN=

# YouTube API (optional)
YOUTUBE_API_KEY=your_api_key_here

# Port configuration
PORT=8443
HTTP_PORT=8080

# TLS certificates
TLS_CERT=./certs/cert.pem
TLS_KEY=./certs/key.pem
```

## Usage

1. **Start the server** - Run `./songmartyn` in the backend directory
2. **Connect your TV** - Open `https://localhost:8443` on your karaoke display
3. **Invite guests** - Click the QR code button to show the connection QR code
4. **Add music** - Use the admin panel to add library folders
5. **Start singing!** - Guests search and queue songs from their phones

## Mobile Interface

The guest interface is designed for one-handed phone use:

- **Now Playing** - See current song with progress
- **Vocal Assist** - Quick toggle between assistance levels
- **Queue View** - See what's coming up
- **Search** - Full-screen search with tabs for Library, YouTube, Popular, and My Songs

## Admin Panel

Access at `/admin` (automatically authenticated on localhost):

- **Clients** - Monitor connected guests, grant admin privileges, kick users
- **Library** - Manage music folders and scan for new songs
- **Search Logs** - See what guests are searching for
- **Network** - Configure which IP address to advertise
- **Settings** - System configuration

## API Reference

### WebSocket Messages

| Type | Direction | Description |
|------|-----------|-------------|
| `handshake` | Client → Server | Initial connection with session key |
| `welcome` | Server → Client | Session confirmed, current state |
| `search` | Client → Server | Search query |
| `queue_add` | Client → Server | Add song to queue |
| `state_update` | Server → Client | Room state changed |
| `vocal_assist` | Client → Server | Set vocal assist level |

### HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/library/songs` | List all songs |
| GET | `/api/library/search` | Search songs |
| GET | `/api/admin/clients` | List connected clients |
| GET | `/api/admin/networks` | List network interfaces |
| GET/POST | `/api/connect-url` | Get/set QR code URL |

## Tech Stack

**Backend**
- Go 1.21+
- Gorilla WebSocket
- SQLite (song library & session storage)
- mpv (media playback)

**Frontend**
- React 18+
- TypeScript
- Vite
- Tailwind CSS
- Zustand (state management)

## Roadmap

- [ ] Audio separation (Demucs/Spleeter integration)
- [ ] Lyrics display with timing
- [ ] Multi-room support
- [ ] Chromecast/AirPlay output
- [ ] Song rating system
- [ ] Playlist templates
- [ ] Duet mode

## Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests to the `main` branch.

## License

MIT License - See [LICENSE](LICENSE) for details.

---

**Made with love for karaoke nights everywhere.**

*Named after the legendary karaoke host who insisted every song needs the perfect amount of vocal support.*
