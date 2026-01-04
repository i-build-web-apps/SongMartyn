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

// Session (The Martyn Handshake)
export interface Session {
  martyn_key: string;
  display_name: string;
  avatar_id?: string;
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
}

// Client info for admin display
export interface ClientInfo {
  martyn_key: string;
  display_name: string;
  device_name: string;
  ip_address: string;
  is_admin: boolean;
  is_online: boolean;
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
}

// Queue state
export interface QueueState {
  songs: Song[];
  position: number;
}

// Room state (full sync)
export interface RoomState {
  player: PlayerState;
  queue: QueueState;
  sessions: Session[];
}

// WebSocket message types
export type MessageType =
  | 'handshake'
  | 'search'
  | 'queue_add'
  | 'queue_remove'
  | 'play'
  | 'pause'
  | 'skip'
  | 'seek'
  | 'vocal_assist'
  | 'volume'
  | 'set_display_name'
  | 'admin_set_admin'
  | 'admin_kick'
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
