import { create } from 'zustand';
import type { ClientInfo, AdminAuthResponse } from '../types';
import { wsService } from '../services/websocket';

const ADMIN_TOKEN_KEY = 'songmartyn_admin_token';
const API_BASE = import.meta.env.DEV ? 'https://localhost:8443' : '';

interface AdminStore {
  // Auth state
  isAuthenticated: boolean;
  isLocal: boolean;
  token: string | null;
  authError: string | null;

  // Client list
  clients: ClientInfo[];

  // Actions
  checkAuth: () => Promise<void>;
  authenticate: (pin: string) => Promise<boolean>;
  logout: () => void;
  setClients: (clients: ClientInfo[]) => void;
  fetchClients: () => Promise<void>;
  setAdminStatus: (martynKey: string, isAdmin: boolean) => Promise<boolean>;
  setAFKStatus: (martynKey: string, isAFK: boolean) => Promise<boolean>;
  kickClient: (martynKey: string, reason?: string) => Promise<boolean>;
  blockClient: (martynKey: string, durationMinutes: number, reason?: string) => Promise<boolean>;
  unblockClient: (martynKey: string) => Promise<boolean>;
  setClientName: (martynKey: string, displayName: string) => Promise<boolean>;
  setClientNameLock: (martynKey: string, locked: boolean) => Promise<boolean>;
}

export const useAdminStore = create<AdminStore>((set, get) => ({
  isAuthenticated: false,
  isLocal: false,
  token: localStorage.getItem(ADMIN_TOKEN_KEY),
  authError: null,
  clients: [],

  checkAuth: async () => {
    try {
      const token = get().token;
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }

      const res = await fetch(`${API_BASE}/api/admin/check`, { headers });
      const data: AdminAuthResponse = await res.json();

      if (data.success) {
        set({ isAuthenticated: true, isLocal: data.is_local, authError: null });
        // If local and no token, get one
        if (data.is_local && !token) {
          const authRes = await fetch(`${API_BASE}/api/admin/auth`);
          const authData: AdminAuthResponse = await authRes.json();
          if (authData.token) {
            localStorage.setItem(ADMIN_TOKEN_KEY, authData.token);
            set({ token: authData.token });
          }
        }
      } else {
        set({ isAuthenticated: false });
      }
    } catch {
      set({ isAuthenticated: false, authError: 'Failed to check auth' });
    }
  },

  authenticate: async (pin: string) => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/auth`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pin }),
      });
      const data: AdminAuthResponse = await res.json();

      if (data.success && data.token) {
        localStorage.setItem(ADMIN_TOKEN_KEY, data.token);
        set({
          isAuthenticated: true,
          isLocal: data.is_local,
          token: data.token,
          authError: null,
        });
        return true;
      } else {
        set({ authError: data.error || 'Authentication failed' });
        return false;
      }
    } catch {
      set({ authError: 'Failed to authenticate' });
      return false;
    }
  },

  logout: () => {
    localStorage.removeItem(ADMIN_TOKEN_KEY);
    set({ isAuthenticated: false, token: null });
  },

  setClients: (clients) => set({ clients }),

  fetchClients: async () => {
    try {
      const token = get().token;
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      const res = await fetch(`${API_BASE}/api/admin/clients`, { headers });
      if (res.ok) {
        const clients: ClientInfo[] = await res.json();
        set({ clients });
      }
    } catch (err) {
      console.error('Failed to fetch clients:', err);
    }
  },

  setAdminStatus: async (martynKey: string, isAdmin: boolean) => {
    try {
      const token = get().token;
      const res = await fetch(`${API_BASE}/api/admin/clients/${martynKey}/admin`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ is_admin: isAdmin }),
      });
      return res.ok;
    } catch {
      return false;
    }
  },

  setAFKStatus: async (martynKey: string, isAFK: boolean) => {
    try {
      wsService.adminSetAFK(martynKey, isAFK);
      return true;
    } catch {
      return false;
    }
  },

  kickClient: async (martynKey: string, reason?: string) => {
    try {
      wsService.adminKick(martynKey, reason);
      return true;
    } catch {
      return false;
    }
  },

  blockClient: async (martynKey: string, durationMinutes: number, reason?: string) => {
    try {
      wsService.adminBlock(martynKey, durationMinutes, reason);
      return true;
    } catch {
      return false;
    }
  },

  unblockClient: async (martynKey: string) => {
    try {
      wsService.adminUnblock(martynKey);
      return true;
    } catch {
      return false;
    }
  },

  setClientName: async (martynKey: string, displayName: string) => {
    try {
      wsService.adminSetName(martynKey, displayName);
      return true;
    } catch {
      return false;
    }
  },

  setClientNameLock: async (martynKey: string, locked: boolean) => {
    try {
      wsService.adminSetNameLock(martynKey, locked);
      return true;
    } catch {
      return false;
    }
  },
}));
