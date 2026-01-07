# SongMartyn Central Stats & Authentication Service

## Overview

A centralized service that collects telemetry from SongMartyn instances, provides license authentication, and offers analytics on usage patterns across the user base.

---

## Goals

1. **Quality Monitoring** - Understand how well SongMartyn performs in the wild
2. **Usage Analytics** - Learn how people use the product to inform development
3. **License Management** - Control authorized vs unauthorized usage
4. **Abuse Prevention** - Identify and block misuse

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    SongMartyn Instances                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ Instance │  │ Instance │  │ Instance │  │ Instance │  ...    │
│  │    A     │  │    B     │  │    C     │  │    D     │        │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘        │
└───────┼─────────────┼─────────────┼─────────────┼───────────────┘
        │             │             │             │
        └─────────────┴──────┬──────┴─────────────┘
                             │ HTTPS
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Stats & Auth Service                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   API       │  │   Auth      │  │  Analytics  │             │
│  │  Gateway    │──│   Engine    │──│   Engine    │             │
│  └──────┬──────┘  └─────────────┘  └──────┬──────┘             │
│         │                                  │                    │
│         ▼                                  ▼                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Redis     │  │ PostgreSQL  │  │ TimescaleDB │             │
│  │  (Cache)    │  │ (Licenses)  │  │  (Metrics)  │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
│  ┌─────────────────────────────────────────────────┐           │
│  │              Admin Dashboard                     │           │
│  │         (React + Grafana embeds)                │           │
│  └─────────────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

---

## Data Collection

### 1. Quality of Service Metrics

| Metric | Description | Frequency |
|--------|-------------|-----------|
| `uptime_seconds` | Time since last restart | On heartbeat |
| `websocket_connections_total` | Total WS connections made | Aggregated |
| `websocket_errors` | Connection failures, disconnects | On occurrence |
| `mpv_health_status` | Player process health (ok/degraded/dead) | Every 30s |
| `mpv_restart_count` | How often mpv needed restarting | Aggregated |
| `api_response_times_ms` | Latency for key operations | Sampled |
| `error_count_by_type` | Errors categorized | Aggregated |
| `memory_usage_mb` | Backend memory consumption | Every 60s |
| `cpu_usage_percent` | Backend CPU load | Every 60s |

### 2. Song Library Metrics

| Metric | Description | Frequency |
|--------|-------------|-----------|
| `library_total_songs` | Total songs in library | On sync |
| `library_songs_by_source` | Breakdown: local, youtube, other | On sync |
| `library_file_types` | Count per extension: mp4, webm, mkv, mp3 | On sync |
| `library_total_size_gb` | Total storage used | On sync |
| `library_growth_rate` | Songs added per period | Daily |
| `library_avg_song_duration_sec` | Average song length | On sync |

### 3. Playback Metrics

| Metric | Description | Frequency |
|--------|-------------|-----------|
| `songs_played_total` | Lifetime plays | On play |
| `songs_played_session` | Plays in current session | On play |
| `song_metadata` | Artist, title, duration, source | On play |
| `skip_events` | Songs skipped before completion | On skip |
| `skip_reason` | User skip, admin skip, error | On skip |
| `queue_additions` | Songs added to queue | On queue |
| `queue_removals` | Songs removed from queue | On remove |
| `average_wait_time_sec` | Time from queue to play | Calculated |

### 4. Session Metrics

| Metric | Description | Frequency |
|--------|-------------|-----------|
| `session_id` | Unique session identifier | On start |
| `session_start_time` | When session began | On start |
| `session_duration_sec` | How long session ran | On end |
| `users_joined_total` | Total unique users in session | Aggregated |
| `users_peak_concurrent` | Max simultaneous users | On change |
| `users_by_role` | Admin vs regular breakdown | On change |
| `room_count` | Number of rooms (future multi-room) | On change |

### 5. Environment Metrics

| Metric | Description | Frequency |
|--------|-------------|-----------|
| `instance_id` | Unique installation identifier | On startup |
| `version` | SongMartyn version | On startup |
| `os_platform` | linux, darwin, windows | On startup |
| `os_version` | Specific OS version | On startup |
| `go_version` | Go runtime version | On startup |
| `mpv_version` | mpv player version | On startup |
| `cpu_cores` | Available CPU cores | On startup |
| `ram_total_gb` | Total system memory | On startup |

---

## Authentication & Licensing

### License Tiers

| Tier | Monthly | Annual | Features | Limits |
|------|---------|--------|----------|--------|
| **Starter** | $0 | $0 | Basic functionality, community support | 25 songs, 5 guests |
| **Party Host** | $9.99 | $95.88 (20% off) | Full features, YouTube import, email support | Unlimited songs, 50 guests |
| **Venue** | $49 | $470 (20% off) | Multi-room, white-label, priority support + SLA | Unlimited everything |

**Competitive Positioning**: SongMartyn Venue at $49/month is ~75% cheaper than KaraFun Business at $199/month per room.

### License Key Format

```
SM-{TIER}-{TIMESTAMP}-{CHECKSUM}
Example: SM-PARTY-20260106-A7B3C9D2
```

### Authentication Flow

```
┌─────────────┐                          ┌─────────────┐
│ SongMartyn  │                          │ Auth Server │
│  Instance   │                          │             │
└──────┬──────┘                          └──────┬──────┘
       │                                        │
       │  1. POST /auth/validate               │
       │  {license_key, instance_id, hwid}     │
       │──────────────────────────────────────►│
       │                                        │
       │  2. Response                          │
       │  {valid: bool, tier, expires,         │
       │   features: [], limits: {}}           │
       │◄──────────────────────────────────────│
       │                                        │
       │  3. Periodic heartbeat (every 1h)     │
       │  POST /auth/heartbeat                 │
       │  {instance_id, session_stats}         │
       │──────────────────────────────────────►│
       │                                        │
       │  4. Heartbeat response                │
       │  {continue: bool, message: ""}        │
       │◄──────────────────────────────────────│
       │                                        │
```

### Enforcement Modes

| Mode | Behavior | Use Case |
|------|----------|----------|
| **Strict** | Block functionality if auth fails | Paid tiers |
| **Graceful** | Warn but allow, with limits | Free tier |
| **Offline** | Cache license, validate when online | All tiers |

### Hardware ID (HWID) Generation

To prevent license sharing, generate a stable hardware ID:

```go
func GenerateHWID() string {
    data := []string{
        getMACAddress(),
        getCPUID(),
        getDiskSerial(),
        getHostname(),
    }
    hash := sha256.Sum256([]byte(strings.Join(data, "|")))
    return hex.EncodeToString(hash[:16])
}
```

---

## API Endpoints

### Stats Ingestion

```
POST /v1/stats/batch
Content-Type: application/json
Authorization: Bearer {instance_token}

{
  "instance_id": "abc123",
  "timestamp": "2026-01-06T12:00:00Z",
  "metrics": [
    {"name": "songs_played_total", "value": 42, "tags": {"source": "youtube"}},
    {"name": "users_peak_concurrent", "value": 15},
    ...
  ]
}

Response: 202 Accepted
```

### License Validation

```
POST /v1/auth/validate
Content-Type: application/json

{
  "license_key": "SM-PARTY-20260106-A7B3C9D2",
  "instance_id": "abc123",
  "hwid": "0a1b2c3d4e5f6789",
  "version": "1.2.0"
}

Response: 200 OK
{
  "valid": true,
  "tier": "party",
  "expires": "2027-01-06T00:00:00Z",
  "features": ["multi-room", "custom-branding", "priority-support"],
  "limits": {
    "max_songs": 2000,
    "max_users": 100,
    "max_rooms": 5
  },
  "message": null
}
```

### Instance Registration

```
POST /v1/instances/register
Content-Type: application/json

{
  "hwid": "0a1b2c3d4e5f6789",
  "platform": "darwin",
  "version": "1.2.0",
  "owner_email": "user@example.com"  // Optional for free tier
}

Response: 201 Created
{
  "instance_id": "abc123",
  "instance_token": "eyJ...",  // JWT for API auth
  "tier": "free",
  "registered_at": "2026-01-06T12:00:00Z"
}
```

---

## Client SDK Integration

### Backend Changes Required

```
backend/
├── internal/
│   ├── telemetry/
│   │   ├── collector.go      # Gathers metrics
│   │   ├── sender.go         # Batches and sends to server
│   │   ├── buffer.go         # Offline buffering
│   │   └── privacy.go        # Anonymization helpers
│   ├── licensing/
│   │   ├── license.go        # License validation
│   │   ├── hwid.go           # Hardware ID generation
│   │   └── enforcement.go    # Feature gating
│   └── ...
├── cmd/songmartyn/
│   └── main.go               # Initialize telemetry
```

### Collector Interface

```go
type MetricsCollector interface {
    // Quality metrics
    RecordError(errorType string, err error)
    RecordLatency(operation string, duration time.Duration)
    RecordMpvHealth(status MpvHealthStatus)

    // Playback metrics
    RecordSongPlayed(song *models.Song)
    RecordSongSkipped(song *models.Song, reason string)
    RecordQueueEvent(eventType string, song *models.Song)

    // Session metrics
    RecordUserJoined(userID string, isAdmin bool)
    RecordUserLeft(userID string)

    // Library metrics
    SyncLibraryStats(songs []models.Song)

    // Flush
    Flush() error
}
```

### Privacy Controls

```go
type PrivacyConfig struct {
    // What to collect
    CollectQualityMetrics  bool `json:"collect_quality"`
    CollectPlaybackMetrics bool `json:"collect_playback"`
    CollectSessionMetrics  bool `json:"collect_session"`
    CollectLibraryMetrics  bool `json:"collect_library"`

    // Anonymization
    AnonymizeSongTitles    bool `json:"anonymize_songs"`    // Hash instead of plaintext
    AnonymizeUserIDs       bool `json:"anonymize_users"`    // Hash user identifiers
    ExcludeLocalFiles      bool `json:"exclude_local"`      // Don't report local file names

    // Transmission
    SendInterval           time.Duration `json:"send_interval"`    // Default: 5 minutes
    MaxBatchSize           int           `json:"max_batch_size"`   // Default: 1000
    RetryCount             int           `json:"retry_count"`      // Default: 3
}
```

---

## Database Schema

### PostgreSQL (Licenses & Instances)

```sql
-- Instances
CREATE TABLE instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hwid VARCHAR(64) UNIQUE NOT NULL,
    instance_token TEXT NOT NULL,
    owner_email VARCHAR(255),
    platform VARCHAR(32),
    version VARCHAR(32),
    registered_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ,
    is_banned BOOLEAN DEFAULT FALSE,
    ban_reason TEXT
);

-- Licenses
CREATE TABLE licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_key VARCHAR(64) UNIQUE NOT NULL,
    tier VARCHAR(32) NOT NULL,
    owner_email VARCHAR(255) NOT NULL,
    instance_id UUID REFERENCES instances(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_revoked BOOLEAN DEFAULT FALSE,
    revoke_reason TEXT
);

-- License activations (audit trail)
CREATE TABLE license_activations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id UUID REFERENCES licenses(id),
    instance_id UUID REFERENCES instances(id),
    activated_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);
```

### TimescaleDB (Metrics)

```sql
-- Enable TimescaleDB
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Quality metrics
CREATE TABLE metrics_quality (
    time TIMESTAMPTZ NOT NULL,
    instance_id UUID NOT NULL,
    metric_name VARCHAR(64) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    tags JSONB
);
SELECT create_hypertable('metrics_quality', 'time');

-- Playback events
CREATE TABLE events_playback (
    time TIMESTAMPTZ NOT NULL,
    instance_id UUID NOT NULL,
    event_type VARCHAR(32) NOT NULL,  -- played, skipped, queued, removed
    song_hash VARCHAR(64),            -- Anonymized song identifier
    song_source VARCHAR(32),          -- youtube, local, etc
    song_duration_sec INTEGER,
    metadata JSONB
);
SELECT create_hypertable('events_playback', 'time');

-- Session events
CREATE TABLE events_session (
    time TIMESTAMPTZ NOT NULL,
    instance_id UUID NOT NULL,
    session_id UUID NOT NULL,
    event_type VARCHAR(32) NOT NULL,  -- started, ended, user_joined, user_left
    user_count INTEGER,
    metadata JSONB
);
SELECT create_hypertable('events_session', 'time');

-- Library snapshots
CREATE TABLE snapshots_library (
    time TIMESTAMPTZ NOT NULL,
    instance_id UUID NOT NULL,
    total_songs INTEGER,
    total_size_bytes BIGINT,
    songs_by_source JSONB,
    songs_by_type JSONB
);
SELECT create_hypertable('snapshots_library', 'time');

-- Continuous aggregates for dashboards
CREATE MATERIALIZED VIEW daily_playback_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS day,
    instance_id,
    COUNT(*) FILTER (WHERE event_type = 'played') AS songs_played,
    COUNT(*) FILTER (WHERE event_type = 'skipped') AS songs_skipped,
    COUNT(DISTINCT song_hash) AS unique_songs
FROM events_playback
GROUP BY day, instance_id;
```

---

## Tech Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **API Server** | Go + Chi router | Matches existing backend, high performance |
| **Auth** | JWT + Redis sessions | Stateless, scalable |
| **Database** | PostgreSQL | Reliable, feature-rich |
| **Time Series** | TimescaleDB | PostgreSQL extension, easy integration |
| **Cache** | Redis | Rate limiting, session cache, hot data |
| **Dashboard** | React + Grafana | React for custom UI, Grafana for metrics |
| **Deployment** | Docker + Fly.io | Easy scaling, global edge |

---

## Privacy & Compliance

### Data Classification

| Level | Data Type | Handling |
|-------|-----------|----------|
| **Public** | Version numbers, platform | No restrictions |
| **Internal** | Aggregate stats, error rates | Anonymized |
| **Sensitive** | Song titles, user counts | Hashed or opt-in only |
| **PII** | Email addresses, IPs | Encrypted, consent required |

### GDPR Considerations

1. **Consent** - Clear opt-in during setup for telemetry
2. **Purpose Limitation** - Only collect what's needed
3. **Data Minimization** - Aggregate when possible
4. **Right to Erasure** - Endpoint to delete instance data
5. **Data Portability** - Export endpoint for user data
6. **Retention Policy** - Auto-delete after 90 days for free tier

### Opt-In Flow

```
┌────────────────────────────────────────────────────┐
│          Help Us Improve SongMartyn                │
│                                                    │
│  We'd love to collect anonymous usage data to     │
│  make SongMartyn better. You can change this      │
│  anytime in Settings.                             │
│                                                    │
│  What we collect:                                 │
│  ✓ Error reports and crash logs                  │
│  ✓ Feature usage statistics                       │
│  ✓ Performance metrics                            │
│                                                    │
│  What we DON'T collect:                           │
│  ✗ Song titles or filenames                       │
│  ✗ Personal information                           │
│  ✗ IP addresses (beyond country)                  │
│                                                    │
│  [Share Anonymous Data]  [No Thanks]              │
└────────────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 1: Foundation (MVP)

**Goal**: Basic stats collection and viewing

- [ ] Set up stats server with Go + Chi
- [ ] PostgreSQL schema for instances
- [ ] TimescaleDB schema for metrics
- [ ] `/v1/stats/batch` endpoint
- [ ] Instance registration endpoint
- [ ] Basic collector in SongMartyn backend
- [ ] Offline buffering (SQLite queue)
- [ ] Simple admin dashboard (list instances, basic charts)

**Deliverables**:
- Stats server deployed
- SongMartyn sends heartbeats
- Can view active instances

### Phase 2: Analytics

**Goal**: Meaningful insights from collected data

- [ ] Continuous aggregates for common queries
- [ ] Grafana dashboards for:
  - Daily/weekly active instances
  - Songs played over time
  - Error rates by version
  - Popular file types
  - Geographic distribution
- [ ] Alerting for anomalies (spike in errors)
- [ ] Weekly digest email for admins

**Deliverables**:
- Production-ready dashboards
- Actionable insights

### Phase 3: Licensing

**Goal**: Control authorized usage

- [ ] License generation system
- [ ] HWID generation in client
- [ ] License validation endpoint
- [ ] Feature gating in SongMartyn
- [ ] Grace period handling
- [ ] License portal (user-facing)
- [ ] Payment integration (Stripe/LemonSqueezy)

**Deliverables**:
- Paid tiers functional
- Self-service license management

### Phase 4: Advanced Features

**Goal**: Enterprise readiness

- [ ] Multi-region deployment
- [ ] SLA monitoring for venue tier
- [ ] Custom reporting
- [ ] Webhook integrations
- [ ] API for third-party analytics
- [ ] White-label support

---

## Security Considerations

### API Security

- All endpoints over HTTPS
- Rate limiting: 100 req/min per instance
- Request signing for stats submission
- JWT with short expiry (1 hour), refresh tokens

### Data Security

- Encryption at rest (PostgreSQL TDE)
- Encryption in transit (TLS 1.3)
- Separate read/write database users
- Audit logging for admin actions

### Abuse Prevention

```go
type AbuseDetector struct {
    // Thresholds
    MaxInstancesPerHWID     int           // Prevent cloning
    MaxRequestsPerMinute    int           // Rate limiting
    MaxErrorReportsPerHour  int           // Spam prevention

    // Patterns
    SuspiciousPatterns      []regexp.Regexp  // Known attack patterns
    BannedHWIDs             map[string]bool  // Blacklist
}

func (a *AbuseDetector) Check(req *StatsRequest) error {
    // Implementation
}
```

---

## Monitoring the Monitor

### Health Checks

- `/health/live` - Is the process running?
- `/health/ready` - Can it serve requests?
- `/health/db` - Database connectivity
- `/health/redis` - Cache connectivity

### Observability

| Aspect | Tool |
|--------|------|
| Metrics | Prometheus + Grafana |
| Logs | Structured JSON, aggregated to Loki |
| Traces | OpenTelemetry (optional) |
| Errors | Sentry |
| Uptime | Better Uptime / Checkly |

---

## Cost Estimation

### Infrastructure (Monthly)

| Resource | Specification | Cost |
|----------|---------------|------|
| API Server | 2 vCPU, 4GB RAM | $30 |
| PostgreSQL | 2 vCPU, 4GB RAM, 100GB | $50 |
| Redis | 1GB | $15 |
| CDN/Edge | Cloudflare Pro | $20 |
| Monitoring | Grafana Cloud (free tier) | $0 |
| **Total** | | **~$115/month** |

### Scaling Triggers

| Metric | Threshold | Action |
|--------|-----------|--------|
| API latency | >500ms p95 | Scale API horizontally |
| DB connections | >80% max | Scale DB vertically |
| Storage | >70% capacity | Increase disk, archive old data |
| Instances | >10,000 active | Consider sharding |

---

## Open Questions

1. **Naming**: Should this be a separate product ("SongMartyn Cloud") or just "Stats Server"?
2. **Self-hosting**: Should enterprises be able to run their own stats server?
3. **Data sharing**: Should anonymized aggregate data be public (e.g., "500,000 songs played this month")?
4. **Pricing model**: Per-instance? Per-song? Flat rate?
5. **Offline-first**: How long should cached license be valid without phone-home?

---

## Next Steps

1. Review and approve this plan
2. Set up repository for stats server
3. Define API contracts (OpenAPI spec)
4. Implement Phase 1 MVP
5. Beta test with select users
6. Iterate based on feedback

---

*Document created: 2026-01-06*
*Author: Claude (with human guidance)*
*Status: Draft - Pending Review*
