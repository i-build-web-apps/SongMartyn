import { create } from 'zustand';
import type {
  Session,
  RoomState,
  PlayerState,
  QueueState,
  Song,
  VocalAssistLevel,
} from '../types';

interface RoomStore {
  // Connection state
  isConnected: boolean;
  isConnecting: boolean;

  // Session (The Martyn Handshake)
  session: Session | null;

  // Player state
  player: PlayerState;

  // Queue state
  queue: QueueState;

  // Other connected sessions
  sessions: Session[];

  // Actions
  setConnected: (connected: boolean) => void;
  setConnecting: (connecting: boolean) => void;
  setSession: (session: Session) => void;
  updateState: (state: RoomState) => void;
  updatePlayer: (player: Partial<PlayerState>) => void;
  updateQueue: (queue: QueueState) => void;
  setVocalAssist: (level: VocalAssistLevel) => void;
}

const initialPlayerState: PlayerState = {
  current_song: null,
  position: 0,
  duration: 0,
  is_playing: false,
  volume: 100,
  vocal_assist: 'OFF',
  bgm_active: false,
};

const initialQueueState: QueueState = {
  songs: [],
  position: 0,
};

export const useRoomStore = create<RoomStore>((set) => ({
  // Initial state
  isConnected: false,
  isConnecting: false,
  session: null,
  player: initialPlayerState,
  queue: initialQueueState,
  sessions: [],

  // Actions
  setConnected: (connected) => set({ isConnected: connected }),

  setConnecting: (connecting) => set({ isConnecting: connecting }),

  setSession: (session) => set({ session }),

  updateState: (state) =>
    set({
      player: state.player,
      queue: state.queue,
      sessions: state.sessions,
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

export const selectSession = (state: RoomStore): Session | null =>
  state.session;

export const selectActiveSessions = (state: RoomStore): Session[] =>
  state.sessions;
