import { create } from 'zustand';
import type { LibraryLocation, LibraryStats } from '../types';

const API_BASE = import.meta.env.DEV ? 'https://localhost:8443' : '';

interface LibraryStore {
  locations: LibraryLocation[];
  stats: LibraryStats;
  isLoading: boolean;
  error: string | null;

  // Actions
  fetchLocations: () => Promise<void>;
  fetchStats: () => Promise<void>;
  addLocation: (path: string, name: string) => Promise<boolean>;
  removeLocation: (id: number) => Promise<boolean>;
  scanLocation: (id: number) => Promise<number | null>;
}

const getAuthHeaders = (): Record<string, string> => {
  const token = localStorage.getItem('songmartyn_admin_token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
};

export const useLibraryStore = create<LibraryStore>((set) => ({
  locations: [],
  stats: { total_songs: 0, total_plays: 0 },
  isLoading: false,
  error: null,

  fetchLocations: async () => {
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${API_BASE}/api/library/locations`, {
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        const locations: LibraryLocation[] = await res.json();
        set({ locations, isLoading: false });
      } else {
        set({ error: 'Failed to fetch locations', isLoading: false });
      }
    } catch (err) {
      set({ error: 'Network error', isLoading: false });
    }
  },

  fetchStats: async () => {
    try {
      const res = await fetch(`${API_BASE}/api/library/stats`);
      if (res.ok) {
        const stats: LibraryStats = await res.json();
        set({ stats });
      }
    } catch (err) {
      console.error('Failed to fetch stats:', err);
    }
  },

  addLocation: async (path: string, name: string) => {
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${API_BASE}/api/library/locations`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ path, name }),
      });
      if (res.ok) {
        const location: LibraryLocation = await res.json();
        set((state) => ({
          locations: [...state.locations, location],
          isLoading: false,
        }));
        return true;
      } else {
        const data = await res.json();
        set({ error: data.error || 'Failed to add location', isLoading: false });
        return false;
      }
    } catch (err) {
      set({ error: 'Network error', isLoading: false });
      return false;
    }
  },

  removeLocation: async (id: number) => {
    try {
      const res = await fetch(`${API_BASE}/api/library/locations/${id}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        set((state) => ({
          locations: state.locations.filter((loc) => loc.id !== id),
        }));
        return true;
      }
      return false;
    } catch (err) {
      return false;
    }
  },

  scanLocation: async (id: number) => {
    set({ isLoading: true, error: null });
    try {
      const res = await fetch(`${API_BASE}/api/library/locations/${id}/scan`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });
      if (res.ok) {
        const data = await res.json();
        // Refresh locations to get updated song count
        const locRes = await fetch(`${API_BASE}/api/library/locations`, {
          headers: getAuthHeaders(),
        });
        if (locRes.ok) {
          const locations: LibraryLocation[] = await locRes.json();
          set({ locations, isLoading: false });
        }
        return data.songs_found;
      } else {
        const data = await res.json();
        set({ error: data.error || 'Failed to scan location', isLoading: false });
        return null;
      }
    } catch (err) {
      set({ error: 'Network error', isLoading: false });
      return null;
    }
  },
}));
