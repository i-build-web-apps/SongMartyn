import { create } from 'zustand';
import type { LibrarySong, SongHistory } from '../types';

const API_BASE = import.meta.env.DEV ? 'https://localhost:8443' : '';

// Feature flags
interface Features {
  youtube_enabled: boolean;
  admin_localhost_only: boolean;
}

let featuresCache: Features | null = null;

export async function getFeatures(): Promise<Features> {
  if (featuresCache) return featuresCache;
  try {
    const res = await fetch(`${API_BASE}/api/features`);
    if (res.ok) {
      featuresCache = await res.json();
      return featuresCache!;
    }
  } catch (err) {
    console.error('Failed to fetch features:', err);
  }
  return { youtube_enabled: false, admin_localhost_only: true };
}

// Convert SongHistory to LibrarySong-like format for display
function historyToSong(history: SongHistory): LibrarySong {
  return {
    id: history.song_id,
    title: history.song_title,
    artist: history.song_artist,
    duration: 0,
    file_path: '',
    library_id: 0,
    times_sung: 0,
    added_at: history.sung_at,
  };
}

export type SearchTab = 'search' | 'popular' | 'history' | 'youtube';

interface YouTubeResult {
  id: string;
  title: string;
  channel: string;
  duration: number;
  thumbnail_url: string;
}

// Convert YouTube result to LibrarySong-like format
function youtubeToSong(result: YouTubeResult): LibrarySong {
  return {
    id: `youtube:${result.id}`,
    title: result.title,
    artist: result.channel,
    duration: result.duration,
    file_path: '',
    thumbnail_url: result.thumbnail_url,
    library_id: 0,
    times_sung: 0,
    added_at: '',
  };
}

interface SearchStore {
  // State
  query: string;
  results: LibrarySong[];
  popularSongs: LibrarySong[];
  historySongs: LibrarySong[];
  youtubeResults: LibrarySong[];
  activeTab: SearchTab;
  isLoading: boolean;
  isOpen: boolean;

  // Actions
  setQuery: (query: string) => void;
  setActiveTab: (tab: SearchTab) => void;
  search: () => Promise<void>;
  searchYouTube: () => Promise<void>;
  fetchPopular: () => Promise<void>;
  fetchHistory: () => Promise<void>;
  openSearch: () => void;
  closeSearch: () => void;
  addToQueue: (song: LibrarySong) => void;
}

const getMartynKey = (): string | null => {
  return localStorage.getItem('songmartyn_key');
};

export const useSearchStore = create<SearchStore>((set, get) => ({
  query: '',
  results: [],
  popularSongs: [],
  historySongs: [],
  youtubeResults: [],
  activeTab: 'search',
  isLoading: false,
  isOpen: false,

  setQuery: (query: string) => set({ query }),

  setActiveTab: (tab: SearchTab) => {
    set({ activeTab: tab });
    // Fetch data for the tab if needed
    if (tab === 'popular' && get().popularSongs.length === 0) {
      get().fetchPopular();
    } else if (tab === 'history' && get().historySongs.length === 0) {
      get().fetchHistory();
    }
  },

  search: async () => {
    const query = get().query.trim();
    if (!query) {
      set({ results: [] });
      return;
    }

    set({ isLoading: true });
    try {
      const res = await fetch(`${API_BASE}/api/library/search?q=${encodeURIComponent(query)}`);
      if (res.ok) {
        const songs: LibrarySong[] = await res.json();
        set({ results: songs, isLoading: false });
      } else {
        set({ results: [], isLoading: false });
      }
    } catch (err) {
      console.error('Search failed:', err);
      set({ results: [], isLoading: false });
    }
  },

  searchYouTube: async () => {
    const query = get().query.trim();
    if (!query) {
      set({ youtubeResults: [] });
      return;
    }

    set({ isLoading: true });
    try {
      const res = await fetch(`${API_BASE}/api/youtube/search?q=${encodeURIComponent(query)}`);
      if (res.ok) {
        const results: YouTubeResult[] = await res.json();
        const songs = results.map(youtubeToSong);
        set({ youtubeResults: songs, isLoading: false });
      } else {
        set({ youtubeResults: [], isLoading: false });
      }
    } catch (err) {
      console.error('YouTube search failed:', err);
      set({ youtubeResults: [], isLoading: false });
    }
  },

  fetchPopular: async () => {
    set({ isLoading: true });
    try {
      const res = await fetch(`${API_BASE}/api/library/popular`);
      if (res.ok) {
        const songs: LibrarySong[] = await res.json();
        set({ popularSongs: songs, isLoading: false });
      } else {
        set({ isLoading: false });
      }
    } catch (err) {
      console.error('Failed to fetch popular:', err);
      set({ isLoading: false });
    }
  },

  fetchHistory: async () => {
    const martynKey = getMartynKey();
    if (!martynKey) {
      set({ historySongs: [], isLoading: false });
      return;
    }

    set({ isLoading: true });
    try {
      const res = await fetch(`${API_BASE}/api/library/history?key=${encodeURIComponent(martynKey)}`);
      if (res.ok) {
        const history: SongHistory[] = await res.json();
        // Convert history to song format for consistent display
        const songs = history.map(historyToSong);
        set({ historySongs: songs, isLoading: false });
      } else {
        set({ historySongs: [], isLoading: false });
      }
    } catch (err) {
      console.error('Failed to fetch history:', err);
      set({ historySongs: [], isLoading: false });
    }
  },

  openSearch: () => {
    set({ isOpen: true });
    // Fetch popular songs on open if not already loaded
    if (get().popularSongs.length === 0) {
      get().fetchPopular();
    }
  },

  closeSearch: () => {
    set({ isOpen: false, query: '', results: [], youtubeResults: [] });
  },

  addToQueue: (song: LibrarySong) => {
    // Log the song selection
    const { query, activeTab } = get();
    const source = activeTab === 'youtube' ? 'youtube' : 'library';
    const martynKey = localStorage.getItem('songmartyn_key') || '';

    fetch(`${API_BASE}/api/library/select`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        song_id: String(song.id),
        song_title: song.title,
        song_artist: song.artist || '',
        source,
        search_query: query,
        martyn_key: martynKey,
      }),
    }).catch(err => console.error('Failed to log song selection:', err));

    // This will be handled via WebSocket
    // For now, we'll dispatch a custom event that the websocket service can listen to
    const event = new CustomEvent('songmartyn:queue_add', { detail: song });
    window.dispatchEvent(event);
  },
}));
