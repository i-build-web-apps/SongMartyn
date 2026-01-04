# Project Specification: SongMartyn

## 1. Project Overview

SongMartyn is a high-performance, professional-grade karaoke application built for a workstation environment (Ubuntu Studio). It is designed to replace "vibe-coded" legacy systems with a robust, low-latency engine using Go for the backend, React for the mobile remote, and mpv for media playback.

### Core Philosophy

- **Performance**: Bare-metal execution (no Docker) to utilize full system resources.
- **Stability**: Persistent sessions ("The Martyn Handshake") to prevent disconnection issues.
- **Elegance**: Smooth "swish" transitions between background music and karaoke tracks.

## 2. Technical Stack

| Component | Technology |
|-----------|------------|
| Language | Go (Golang) 1.26+ |
| Frontend | React (Mobile-optimized remote control) |
| Media Engine | mpv (Hardware-accelerated) |
| Audio Server | PipeWire (Low-latency) |
| Database | SQLite (For persistent queue and session data) |
| AI | demucs or uvr5_cli for real-time vocal stem extraction |

## 3. System Architecture

### A. The "Martyn-Nest" (Go Backend)

The backend manages the state of the room, handles WebSocket connections from phones, and controls the mpv instance via IPC.

- **Concurrency**: Use goroutines to separate the "Nest Hub" (WebSockets) from the "Chortle Engine" (AI Splitting).
- **Hardware Control**: Interface with PipeWire for cross-fading audio streams.

### B. The "Martyn-Wing" (React Remote)

A web-based remote accessible via QR code.

- **Session Persistence**: Store a MartynKey (UUID) in localStorage.
- **UI/UX**: High-contrast "Matte" design with neon accents and "Aerial" transitions.

## 4. Key Features

### Feature 1: Vocal Assist ("The Chortle")

Dynamic layering of original vocals over the instrumental track to assist struggling singers.

**Logic**: Mix two audio tracks in mpv using `--lavfi-complex`.

**Levels**:
| Level | Vocal Gain | Purpose |
|-------|------------|---------|
| OFF | 0% | No vocal assistance |
| LOW | 15% | Pitch reference |
| MED | 45% | Melody support |
| HIGH | 80% | Full vocal lead |

### Feature 2: Persistent Session ("The Martyn Handshake")

- **Behavior**: When a phone connects, the backend checks for an existing MartynKey.
- **Resumption**: If found, the user's search history, active song, and "Vocal Assist" preferences are restored immediately even after a browser refresh or Wi-Fi drop.

### Feature 3: Atmosphere Control ("The Dawnsong")

Automated background music management.

- **BGM Mode**: Plays a radio stream or local folder when the queue is empty.
- **Crossfade**: Smoothly transitions from BGM to Karaoke by fading out BGM to 5% while the main track hits 100%.

## 5. Deployment & Optimization (Bare Metal)

### OS Configuration (Ubuntu Studio)

- **Kernel**: Low-latency kernel enabled.
- **CPU Governor**: Set to `performance`.
- **Audio Quantum**: Fixed at 128 or 256 in PipeWire.

### Performance Guardrails

- **Isolation**: Use `runtime.LockOSThread()` for critical audio-timing loops.
- **Memory**: Set `GOGC=200` to minimize Garbage Collection pauses during playback.

## 6. Implementation Checklist

- [ ] Initialize Go project with `mpvipc` and `gorilla/websocket`.
- [ ] Create a PipeWire wrapper to handle volume crossfading via CLI or CGO.
- [ ] Implement the `lavfi-complex` mixing string for mpv to handle dual-audio stems.
- [ ] Setup an async task queue for yt-dlp and AI splitting so the UI never blocks.
- [ ] Build the React frontend with a "Segmented Control" for Vocal Assist levels.

## 7. Visual Identity

**Logo Theme**: Stylized Purple Martin bird constructed from brightly colored triangles.

**Primary Colors**:
| Color | Hex Code | Usage |
|-------|----------|-------|
| Deep Indigo | `#1A1B41` | Primary background |
| Neon Yellow | `#F4D35E` | Accents & highlights |
| Matte Black | `#000000` | Text & UI elements |
