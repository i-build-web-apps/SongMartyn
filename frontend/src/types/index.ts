// Vocal Assist Levels (The Chortle)
export type VocalAssistLevel = 'OFF' | 'LOW' | 'MED' | 'HIGH';

export const VOCAL_GAIN_MAP: Record<VocalAssistLevel, number> = {
  OFF: 0,
  LOW: 15,
  MED: 45,
  HIGH: 80,
};

export const VOCAL_LABELS: Record<VocalAssistLevel, string> = {
  OFF: 'Off',
  LOW: 'Pitch',
  MED: 'Melody',
  HIGH: 'Full',
};

// Song in queue
export interface Song {
  id: string;
  title: string;
  artist: string;
  duration: number;
  thumbnail_url: string;
  video_url: string;
  vocal_path?: string;
  instr_path?: string;
  vocal_assist: VocalAssistLevel;
  added_by: string;
  added_at: string;
}

// Avatar custom colors (optional overrides)
export interface AvatarColors {
  env?: string;   // Background color
  clo?: string;   // Clothes primary color
  head?: string;  // Skin color
  mouth?: string; // Mouth color
  eyes?: string;  // Eyes primary color
  top?: string;   // Hair/top color
}

// Avatar configuration
export interface AvatarConfig {
  env: number;
  clo: number;
  head: number;
  mouth: number;
  eyes: number;
  top: number;
  colors?: AvatarColors; // Optional custom colors
}

// Session (The Martyn Handshake)
export interface Session {
  martyn_key: string;
  display_name: string;
  avatar_id?: string;
  avatar_config?: AvatarConfig;
  vocal_assist: VocalAssistLevel;
  search_history: string[];
  current_song_id?: string;
  connected_at: string;
  last_seen_at: string;
  // Admin panel fields
  ip_address: string;
  device_name: string;
  user_agent: string;
  is_admin: boolean;
  is_online: boolean;
  is_afk: boolean;
  name_locked: boolean;
}

// Client info for admin display
export interface ClientInfo {
  martyn_key: string;
  display_name: string;
  device_name: string;
  ip_address: string;
  is_admin: boolean;
  is_online: boolean;
  is_afk: boolean;
  is_blocked: boolean;
  block_reason?: string;
  avatar_config?: AvatarConfig;
  name_locked: boolean;
}

// Player state
export interface PlayerState {
  current_song: Song | null;
  position: number;
  duration: number;
  is_playing: boolean;
  volume: number;
  vocal_assist: VocalAssistLevel;
  bgm_active: boolean;
  bgm_enabled: boolean;
}

// BGM (Background Music) settings
export type BGMSourceType = 'youtube' | 'icecast';

export interface BGMSettings {
  enabled: boolean;
  source_type: BGMSourceType;
  url: string;
  volume: number;
}

// Icecast stream for BGM
export interface IcecastStream {
  name: string;
  url: string;
  genre: string;
  description: string;
  bitrate: number;
  format: string;
}

// Queue state
export interface QueueState {
  songs: Song[];
  position: number;
  autoplay: boolean;
}

// Countdown state (inter-song countdown)
export interface CountdownState {
  active: boolean;
  seconds_remaining: number;
  next_song_id: string;
  next_singer_key: string;
  requires_approval: boolean;
}

// Room state (full sync)
export interface RoomState {
  player: PlayerState;
  queue: QueueState;
  sessions: Session[];
  countdown: CountdownState;
}

// WebSocket message types
export type MessageType =
  | 'handshake'
  | 'search'
  | 'queue_add'
  | 'queue_remove'
  | 'queue_move'
  | 'queue_clear'
  | 'queue_shuffle'
  | 'queue_requeue'
  | 'set_afk'
  | 'play'
  | 'pause'
  | 'skip'
  | 'seek'
  | 'vocal_assist'
  | 'volume'
  | 'set_display_name'
  | 'autoplay'
  | 'admin_set_admin'
  | 'admin_kick'
  | 'admin_block'
  | 'admin_unblock'
  | 'admin_set_afk'
  | 'admin_play_next'
  | 'admin_stop'
  | 'admin_set_name'
  | 'admin_set_name_lock'
  | 'welcome'
  | 'state_update'
  | 'search_result'
  | 'error'
  | 'client_list'
  | 'kicked';

export interface WebSocketMessage<T = unknown> {
  type: MessageType;
  payload: T;
}

export interface WelcomePayload {
  session: Session;
  room_state: RoomState;
}

export interface SearchResult {
  id: string;
  title: string;
  artist: string;
  duration: number;
  thumbnail_url: string;
}

// Admin API types
export interface AdminAuthResponse {
  success: boolean;
  token?: string;
  error?: string;
  is_local: boolean;
}

export interface AdminSetAdminPayload {
  martyn_key: string;
  is_admin: boolean;
}

export interface AdminKickPayload {
  martyn_key: string;
  reason?: string;
}

export interface AdminBlockPayload {
  martyn_key: string;
  duration: number; // Duration in minutes (0 = permanent)
  reason?: string;
}

// Library types
export interface LibraryLocation {
  id: number;
  path: string;
  name: string;
  song_count: number;
  added_at: string;
  last_scan: string;
}

export interface LibrarySong {
  id: string;
  title: string;
  artist: string;
  album?: string;
  duration: number;
  file_path: string;
  thumbnail_url?: string;
  vocal_path?: string;
  instr_path?: string;
  library_id: number;
  times_sung: number;
  last_sung_at?: string;
  last_sung_by?: string;
  added_at: string;
}

export interface LibraryStats {
  total_songs: number;
  total_plays: number;
}

export interface SongHistory {
  id: number;
  song_id: string;
  martyn_key: string;
  sung_at: string;
  song_title: string;
  song_artist: string;
}
