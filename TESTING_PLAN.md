# SongMartyn Testing Plan

This document provides a comprehensive testing checklist for all major and minor features of SongMartyn. Run through this plan before launching any major versions.

## Prerequisites

- Backend server running (`./songmartyn`)
- Frontend built and served (npm run build in frontend/)
- MPV player installed
- Test songs in the library
- At least 2 browser sessions available for testing

---

## 1. Connection & Session Management

### 1.1 Initial Connection
- [ ] New user can connect and gets auto-generated name
- [ ] WebSocket connection established (check logs)
- [ ] User receives welcome message with session info
- [ ] MartynKey is persisted in localStorage

### 1.2 Session Persistence
- [ ] Refreshing page restores session
- [ ] Display name is preserved
- [ ] Avatar configuration is preserved
- [ ] Admin status is preserved

### 1.3 Multiple Clients
- [ ] Multiple browsers can connect simultaneously
- [ ] Each gets unique session unless same MartynKey
- [ ] Client list updates in real-time

---

## 2. User Profile & Avatar

### 2.1 Avatar Creation
- [ ] Avatar creator opens from header
- [ ] All avatar components can be changed (head, eyes, mouth, clothing, etc.)
- [ ] Custom colors can be selected
- [ ] Preview updates in real-time
- [ ] Save persists avatar config

### 2.2 Display Name
- [ ] Can change display name
- [ ] Name updates across all views
- [ ] Name visible in queue and admin panel

---

## 3. Admin Panel

### 3.1 Admin Access
- [ ] Admin panel accessible at /admin
- [ ] PIN protection works (default: 1234)
- [ ] Local network access granted automatically
- [ ] Non-admins see "Not Authorized"

### 3.2 Client Management
- [ ] Online clients displayed correctly
- [ ] Offline clients shown separately
- [ ] Can grant/revoke admin status
- [ ] Can set users to AFK
- [ ] Can kick users
- [ ] Can block users with duration
- [ ] Can unblock users

### 3.3 AFK Functionality
- [ ] Setting user AFK moves their songs to end of queue
- [ ] AFK badge shown on user
- [ ] AFK modal appears for affected user
- [ ] "I'm Back" button clears AFK status

---

## 4. Song Library & Search

### 4.1 Library Management
- [ ] Library locations displayed
- [ ] Can add new library location
- [ ] Rescan finds new songs
- [ ] Song count accurate

### 4.2 Search
- [ ] Search opens from main page
- [ ] Results show matching songs
- [ ] Thumbnails display correctly
- [ ] Can add song to queue from results
- [ ] Vocal assist level selectable when adding

---

## 5. Queue Management

### 5.1 User Queue View
- [ ] Only shows current song and upcoming songs (no history)
- [ ] Song count reflects upcoming only
- [ ] Current song highlighted
- [ ] Can remove own songs
- [ ] Cannot remove others' songs

### 5.2 Admin Queue View
- [ ] "Up Next" tab shows current and upcoming
- [ ] "History" tab shows previously played songs
- [ ] History in reverse chronological order
- [ ] Can remove any song
- [ ] Can reorder songs via drag and drop
- [ ] Shuffle button works (shuffles upcoming only)
- [ ] Clear queue works

### 5.3 Queue Updates
- [ ] Queue changes sync across all clients
- [ ] Position updates when songs complete

---

## 6. Playback Control

### 6.1 Basic Playback
- [ ] Song starts playing in MPV
- [ ] Play/Pause works
- [ ] Seek works
- [ ] Volume control works

### 6.2 Vocal Assist
- [ ] OFF - Original audio plays
- [ ] LOW - Vocals at low level
- [ ] MED - Vocals at medium level
- [ ] HIGH - Full vocal track
- [ ] Can change level during playback

### 6.3 Now Playing Display
- [ ] Song title and artist shown
- [ ] Singer avatar and name shown
- [ ] Playing/Paused status indicator
- [ ] Vocal assist level badge

---

## 7. Countdown & Approval System

### 7.1 Same User Auto-Play
- [ ] When song ends and next song is same user:
  - [ ] 15-second countdown starts
  - [ ] Countdown shows cyan indicator in admin
  - [ ] Song auto-plays after countdown
  - [ ] "Start Now" skips countdown

### 7.2 Different User Approval
- [ ] When song ends and next song is different user:
  - [ ] 15-second countdown starts
  - [ ] Countdown shows orange indicator in admin
  - [ ] "Waiting for Admin Approval" message
  - [ ] Song does NOT auto-play
  - [ ] "Start Now" begins playback

### 7.3 Skip Behavior
- [ ] Skip during playback triggers countdown
- [ ] Countdown uses same logic (same/different user)

---

## 8. Holding Screen

### 8.1 Display Triggers
- [ ] Shows when queue is empty
- [ ] Shows when current song removed
- [ ] Shows between songs during countdown

### 8.2 Content
- [ ] Logo displayed as background
- [ ] QR code for connection URL
- [ ] "Scan to join!" text visible
- [ ] "Next Up" card shows when song queued:
  - [ ] Singer avatar
  - [ ] Singer name
  - [ ] Song title
  - [ ] Artist name

### 8.3 Avatar PNG Export
- [ ] Avatar saved to temp/current-singer-avatar.png
- [ ] File updates when song changes
- [ ] PNG has transparency

---

## 9. Background Music (BGM)

### 9.1 BGM Settings
- [ ] Can enable/disable BGM in settings
- [ ] Can set Icecast stream URL
- [ ] Settings persist

### 9.2 BGM Playback
- [ ] BGM plays when queue empty (if enabled)
- [ ] BGM stops when song starts
- [ ] BGM indicator shown in UI

---

## 10. Autoplay

### 10.1 Toggle
- [ ] Autoplay toggle in admin queue
- [ ] State persists across restarts

### 10.2 Behavior
- [ ] When ON: Next song plays after countdown
- [ ] When OFF: Playback stops, manual skip required

---

## 11. WebSocket Communication

### 11.1 State Sync
- [ ] All clients receive state updates
- [ ] No stale data after operations
- [ ] Reconnection restores state

### 11.2 Error Handling
- [ ] Invalid commands return errors
- [ ] Unauthorized actions rejected
- [ ] Client handles disconnects gracefully

---

## 12. Edge Cases

### 12.1 Queue Edge Cases
- [ ] Empty queue handling
- [ ] Single song queue
- [ ] Remove currently playing song
- [ ] Remove all songs by one user

### 12.2 Session Edge Cases
- [ ] Admin disconnects during countdown
- [ ] User goes AFK mid-song
- [ ] Blocked user tries to reconnect

### 12.3 Playback Edge Cases
- [ ] Song file not found
- [ ] Corrupted audio file
- [ ] Very short song (<10s)
- [ ] Very long song (>10min)

---

## Test Execution Notes

### Running Tests
1. Start fresh - clear database and restart server
2. Open admin panel in Browser A
3. Open user view in Browser B (incognito)
4. Work through sections sequentially
5. Note any failures with reproduction steps

### Reporting Issues
- Create GitHub issue for each failure
- Include:
  - Section and test case
  - Expected behavior
  - Actual behavior
  - Console/server logs
  - Steps to reproduce
