import { create } from 'zustand';
import type {
  Session,
  RoomState,
  PlayerState,
  QueueState,
  CountdownState,
  Song,
  VocalAssistLevel,
} from '../types';

export type NotificationType = 'success' | 'info' | 'warning' | 'error';

export interface Notification {
  id: string;
  type: NotificationType;
  message: string;
  timestamp: number;
}

interface RoomStore {
  // Connection state
  isConnected: boolean;
  isConnecting: boolean;
  isBlocked: boolean;
  blockReason: string;

  // Session (The Martyn Handshake)
  session: Session | null;

  // Player state
  player: PlayerState;

  // Queue state
  queue: QueueState;

  // Countdown state (inter-song)
  countdown: CountdownState;

  // Other connected sessions
  sessions: Session[];

  // Notifications
  notifications: Notification[];

  // Actions
  setConnected: (connected: boolean) => void;
  setConnecting: (connecting: boolean) => void;
  setBlocked: (blocked: boolean, reason?: string) => void;
  setSession: (session: Session) => void;
  updateState: (state: RoomState) => void;
  updatePlayer: (player: Partial<PlayerState>) => void;
  updateQueue: (queue: QueueState) => void;
  setVocalAssist: (level: VocalAssistLevel) => void;
  addNotification: (type: NotificationType, message: string) => void;
  removeNotification: (id: string) => void;
  clearNotifications: () => void;
}

const initialPlayerState: PlayerState = {
  current_song: null,
  position: 0,
  duration: 0,
  is_playing: false,
  volume: 100,
  vocal_assist: 'OFF',
  bgm_active: false,
  bgm_enabled: false,
};

const initialQueueState: QueueState = {
  songs: [],
  position: 0,
  autoplay: false,
};

const initialCountdownState: CountdownState = {
  active: false,
  seconds_remaining: 0,
  next_song_id: '',
  next_singer_key: '',
  requires_approval: false,
};

export const useRoomStore = create<RoomStore>((set) => ({
  // Initial state
  isConnected: false,
  isConnecting: false,
  isBlocked: false,
  blockReason: '',
  session: null,
  player: initialPlayerState,
  queue: initialQueueState,
  countdown: initialCountdownState,
  sessions: [],
  notifications: [],

  // Actions
  setConnected: (connected) => set({ isConnected: connected }),

  setConnecting: (connecting) => set({ isConnecting: connecting }),

  setBlocked: (blocked, reason = '') => set({ isBlocked: blocked, blockReason: reason }),

  setSession: (session) => set({ session }),

  updateState: (state) =>
    set((currentState) => {
      // Sync the user's session with server state (e.g., when admin sets AFK)
      let updatedSession = currentState.session;
      if (currentState.session) {
        const serverSession = state.sessions.find(
          (s) => s.martyn_key === currentState.session?.martyn_key
        );
        if (serverSession) {
          updatedSession = {
            ...currentState.session,
            is_afk: serverSession.is_afk,
            is_admin: serverSession.is_admin,
          };
        }
      }

      return {
        player: state.player,
        queue: state.queue,
        countdown: state.countdown || initialCountdownState,
        sessions: state.sessions,
        session: updatedSession,
      };
    }),

  updatePlayer: (player) =>
    set((state) => ({
      player: { ...state.player, ...player },
    })),

  updateQueue: (queue) => set({ queue }),

  setVocalAssist: (level) =>
    set((state) => ({
      player: { ...state.player, vocal_assist: level },
      session: state.session
        ? { ...state.session, vocal_assist: level }
        : null,
    })),

  addNotification: (type, message) =>
    set((state) => ({
      notifications: [
        ...state.notifications,
        {
          id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
          type,
          message,
          timestamp: Date.now(),
        },
      ].slice(-5), // Keep only the last 5 notifications
    })),

  removeNotification: (id) =>
    set((state) => ({
      notifications: state.notifications.filter((n) => n.id !== id),
    })),

  clearNotifications: () => set({ notifications: [] }),
}));

// Selectors
export const selectCurrentSong = (state: RoomStore): Song | null =>
  state.player.current_song;

export const selectIsPlaying = (state: RoomStore): boolean =>
  state.player.is_playing;

export const selectVocalAssist = (state: RoomStore): VocalAssistLevel =>
  state.player.vocal_assist;

export const selectQueue = (state: RoomStore): Song[] => state.queue.songs;

export const selectQueuePosition = (state: RoomStore): number =>
  state.queue.position;

export const selectAutoplay = (state: RoomStore): boolean =>
  state.queue.autoplay;

export const selectCountdown = (state: RoomStore): CountdownState =>
  state.countdown;

export const selectSession = (state: RoomStore): Session | null =>
  state.session;

export const selectActiveSessions = (state: RoomStore): Session[] =>
  state.sessions;

export const selectNotifications = (state: RoomStore): Notification[] =>
  state.notifications;
