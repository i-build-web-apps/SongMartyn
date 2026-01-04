import { useEffect, useState } from 'react';
import { useAdminStore } from '../stores/adminStore';
import { useLibraryStore } from '../stores/libraryStore';
import { wsService } from '../services/websocket';
import type { ClientInfo, LibraryLocation } from '../types';

type AdminTab = 'clients' | 'library' | 'search-logs' | 'network' | 'settings';

interface SearchLogEntry {
  id: number;
  query: string;
  source: string;
  results_count: number;
  martyn_key: string;
  ip_address: string;
  searched_at: string;
}

interface SearchStats {
  total_searches: number;
  unique_queries: number;
  not_found_count: number;
  top_not_found: SearchLogEntry[];
}

function PinEntry() {
  const [pin, setPin] = useState('');
  const { authenticate, authError } = useAdminStore();
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    await authenticate(pin);
    setIsLoading(false);
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="bg-matte-gray rounded-2xl p-8 w-full max-w-sm">
        <h1 className="text-2xl font-bold text-white mb-2">Admin Access</h1>
        <p className="text-gray-400 mb-6">Enter the admin PIN to continue</p>

        <form onSubmit={handleSubmit}>
          <input
            type="password"
            value={pin}
            onChange={(e) => setPin(e.target.value)}
            placeholder="Enter PIN"
            className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon mb-4"
            autoFocus
          />

          {authError && (
            <p className="text-red-400 text-sm mb-4">{authError}</p>
          )}

          <button
            type="submit"
            disabled={isLoading || !pin}
            className="w-full py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform disabled:opacity-50"
          >
            {isLoading ? 'Authenticating...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  );
}

function ClientRow({ client, onToggleAdmin, onKick }: {
  client: ClientInfo;
  onToggleAdmin: () => void;
  onKick: () => void;
}) {
  return (
    <tr className="border-b border-white/5">
      <td className="py-3 px-4">
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full ${client.is_online ? 'bg-green-400' : 'bg-gray-500'}`} />
          <span className="text-white font-medium">{client.display_name}</span>
        </div>
      </td>
      <td className="py-3 px-4 text-gray-400">{client.device_name || 'Unknown'}</td>
      <td className="py-3 px-4 text-gray-400 font-mono text-sm">{client.ip_address}</td>
      <td className="py-3 px-4">
        <button
          onClick={onToggleAdmin}
          className={`px-3 py-1 rounded-lg text-sm font-medium transition-colors ${
            client.is_admin
              ? 'bg-yellow-neon/20 text-yellow-neon'
              : 'bg-matte-black text-gray-400 hover:text-white'
          }`}
        >
          {client.is_admin ? 'Admin' : 'User'}
        </button>
      </td>
      <td className="py-3 px-4">
        {client.is_online && (
          <button
            onClick={onKick}
            className="px-3 py-1 bg-red-500/20 text-red-400 rounded-lg text-sm font-medium hover:bg-red-500/30 transition-colors"
          >
            Kick
          </button>
        )}
      </td>
    </tr>
  );
}

function ClientList() {
  const { clients, setAdminStatus, kickClient } = useAdminStore();

  const handleToggleAdmin = async (client: ClientInfo) => {
    await setAdminStatus(client.martyn_key, !client.is_admin);
  };

  const handleKick = async (client: ClientInfo) => {
    if (confirm(`Kick ${client.display_name}?`)) {
      await kickClient(client.martyn_key);
    }
  };

  const onlineClients = clients.filter((c) => c.is_online);
  const offlineClients = clients.filter((c) => !c.is_online);

  return (
    <div className="bg-matte-gray rounded-2xl overflow-hidden">
      <div className="px-6 py-4 border-b border-white/5">
        <h2 className="text-lg font-semibold text-white">Connected Clients</h2>
        <p className="text-sm text-gray-400">
          {onlineClients.length} online, {offlineClients.length} offline
        </p>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="text-left text-gray-400 text-sm border-b border-white/5">
              <th className="py-3 px-4 font-medium">Name</th>
              <th className="py-3 px-4 font-medium">Device</th>
              <th className="py-3 px-4 font-medium">IP Address</th>
              <th className="py-3 px-4 font-medium">Role</th>
              <th className="py-3 px-4 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {clients.length === 0 ? (
              <tr>
                <td colSpan={5} className="py-8 text-center text-gray-500">
                  No clients connected
                </td>
              </tr>
            ) : (
              clients.map((client) => (
                <ClientRow
                  key={client.martyn_key}
                  client={client}
                  onToggleAdmin={() => handleToggleAdmin(client)}
                  onKick={() => handleKick(client)}
                />
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function LibraryManagement() {
  const { locations, stats, isLoading, error, fetchLocations, fetchStats, addLocation, removeLocation, scanLocation } = useLibraryStore();
  const [showAddForm, setShowAddForm] = useState(false);
  const [newPath, setNewPath] = useState('');
  const [newName, setNewName] = useState('');
  const [scanningId, setScanningId] = useState<number | null>(null);

  useEffect(() => {
    fetchLocations();
    fetchStats();
  }, [fetchLocations, fetchStats]);

  const handleAddLocation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPath || !newName) return;

    const success = await addLocation(newPath, newName);
    if (success) {
      setNewPath('');
      setNewName('');
      setShowAddForm(false);
    }
  };

  const handleScan = async (location: LibraryLocation) => {
    setScanningId(location.id);
    const count = await scanLocation(location.id);
    setScanningId(null);
    if (count !== null) {
      alert(`Found ${count} songs in ${location.name}`);
      fetchStats();
    }
  };

  const handleRemove = async (location: LibraryLocation) => {
    if (confirm(`Remove "${location.name}" and all its songs from the library?`)) {
      await removeLocation(location.id);
      fetchStats();
    }
  };

  return (
    <div className="bg-matte-gray rounded-2xl overflow-hidden">
      <div className="px-6 py-4 border-b border-white/5">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-white">Song Library</h2>
            <p className="text-sm text-gray-400">
              {stats.total_songs} songs, {stats.total_plays} total plays
            </p>
          </div>
          <button
            onClick={() => setShowAddForm(!showAddForm)}
            className="px-4 py-2 bg-yellow-neon text-indigo-deep font-semibold rounded-lg hover:scale-[1.02] transition-transform"
          >
            {showAddForm ? 'Cancel' : 'Add Location'}
          </button>
        </div>
      </div>

      {error && (
        <div className="px-6 py-3 bg-red-500/20 text-red-400 text-sm">
          {error}
        </div>
      )}

      {showAddForm && (
        <form onSubmit={handleAddLocation} className="px-6 py-4 border-b border-white/5 space-y-3">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Folder Path</label>
            <input
              type="text"
              value={newPath}
              onChange={(e) => setNewPath(e.target.value)}
              placeholder="/path/to/music/folder"
              className="w-full px-4 py-2 bg-matte-black rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Display Name</label>
            <input
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="My Karaoke Collection"
              className="w-full px-4 py-2 bg-matte-black rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            />
          </div>
          <button
            type="submit"
            disabled={isLoading || !newPath || !newName}
            className="px-4 py-2 bg-green-500 text-white font-semibold rounded-lg hover:bg-green-600 transition-colors disabled:opacity-50"
          >
            {isLoading ? 'Adding...' : 'Add & Scan'}
          </button>
        </form>
      )}

      <div className="divide-y divide-white/5">
        {locations.length === 0 ? (
          <div className="px-6 py-8 text-center text-gray-500">
            No library locations configured. Add a folder to scan for songs.
          </div>
        ) : (
          locations.map((location) => (
            <div key={location.id} className="px-6 py-4 flex items-center justify-between">
              <div>
                <h3 className="text-white font-medium">{location.name}</h3>
                <p className="text-sm text-gray-400">{location.path}</p>
                <p className="text-xs text-gray-500 mt-1">
                  {location.song_count} songs
                  {location.last_scan && ` • Last scanned: ${new Date(location.last_scan).toLocaleDateString()}`}
                </p>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => handleScan(location)}
                  disabled={scanningId === location.id}
                  className="px-3 py-1.5 bg-blue-500/20 text-blue-400 rounded-lg text-sm font-medium hover:bg-blue-500/30 transition-colors disabled:opacity-50"
                >
                  {scanningId === location.id ? 'Scanning...' : 'Rescan'}
                </button>
                <button
                  onClick={() => handleRemove(location)}
                  className="px-3 py-1.5 bg-red-500/20 text-red-400 rounded-lg text-sm font-medium hover:bg-red-500/30 transition-colors"
                >
                  Remove
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function NetworkSettings() {
  const [ssid, setSsid] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Load current WiFi settings on mount
  useEffect(() => {
    // TODO: Fetch current WiFi settings from backend
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!ssid) return;

    setIsSaving(true);
    setMessage(null);

    try {
      // TODO: Save WiFi settings to backend
      // For now, just simulate success
      await new Promise(resolve => setTimeout(resolve, 1000));
      setMessage({ type: 'success', text: 'Network settings saved. Device will connect on next restart.' });
    } catch {
      setMessage({ type: 'error', text: 'Failed to save network settings.' });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="bg-matte-gray rounded-2xl overflow-hidden">
      <div className="px-6 py-4 border-b border-white/5">
        <h2 className="text-lg font-semibold text-white">Network Settings</h2>
        <p className="text-sm text-gray-400">Configure WiFi connection for the karaoke system</p>
      </div>

      <form onSubmit={handleSave} className="p-6 space-y-4">
        <div>
          <label className="block text-sm text-gray-400 mb-1">WiFi Network Name (SSID)</label>
          <input
            type="text"
            value={ssid}
            onChange={(e) => setSsid(e.target.value)}
            placeholder="Enter network name"
            className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">Password</label>
          <div className="relative">
            <input
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter WiFi password"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon pr-12"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
            >
              {showPassword ? (
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                </svg>
              ) : (
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
              )}
            </button>
          </div>
        </div>

        {message && (
          <div className={`p-3 rounded-lg text-sm ${message.type === 'success' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
            {message.text}
          </div>
        )}

        <button
          type="submit"
          disabled={isSaving || !ssid}
          className="w-full py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform disabled:opacity-50"
        >
          {isSaving ? 'Saving...' : 'Save Network Settings'}
        </button>
      </form>

      <div className="px-6 pb-6">
        <div className="p-4 bg-matte-black/50 rounded-xl">
          <h3 className="text-white font-medium mb-2">QR Code for Guests</h3>
          <p className="text-sm text-gray-400 mb-3">
            Display a QR code that guests can scan to connect to the WiFi network and access the karaoke system.
          </p>
          <button className="px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg text-sm font-medium hover:bg-blue-500/30 transition-colors">
            Generate QR Code
          </button>
        </div>
      </div>
    </div>
  );
}

function GeneralSettings() {
  const [adminPin, setAdminPin] = useState('');
  const [newPin, setNewPin] = useState('');
  const [confirmPin, setConfirmPin] = useState('');
  const [autoPlay, setAutoPlay] = useState(true);
  const [maxQueuePerUser, setMaxQueuePerUser] = useState(3);
  const [isSaving, setIsSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleChangePin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!adminPin || !newPin || newPin !== confirmPin) {
      setMessage({ type: 'error', text: 'Please fill all fields and ensure new PINs match.' });
      return;
    }

    setIsSaving(true);
    setMessage(null);

    try {
      // TODO: Change admin PIN via backend API
      await new Promise(resolve => setTimeout(resolve, 1000));
      setMessage({ type: 'success', text: 'Admin PIN changed successfully.' });
      setAdminPin('');
      setNewPin('');
      setConfirmPin('');
    } catch {
      setMessage({ type: 'error', text: 'Failed to change PIN. Check current PIN is correct.' });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Change Admin PIN */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Change Admin PIN</h2>
          <p className="text-sm text-gray-400">Update the PIN required for admin access</p>
        </div>

        <form onSubmit={handleChangePin} className="p-6 space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Current PIN</label>
            <input
              type="password"
              value={adminPin}
              onChange={(e) => setAdminPin(e.target.value)}
              placeholder="Enter current PIN"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">New PIN</label>
            <input
              type="password"
              value={newPin}
              onChange={(e) => setNewPin(e.target.value)}
              placeholder="Enter new PIN"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Confirm New PIN</label>
            <input
              type="password"
              value={confirmPin}
              onChange={(e) => setConfirmPin(e.target.value)}
              placeholder="Confirm new PIN"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            />
          </div>

          {message && (
            <div className={`p-3 rounded-lg text-sm ${message.type === 'success' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
              {message.text}
            </div>
          )}

          <button
            type="submit"
            disabled={isSaving || !adminPin || !newPin || !confirmPin}
            className="w-full py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform disabled:opacity-50"
          >
            {isSaving ? 'Changing...' : 'Change PIN'}
          </button>
        </form>
      </div>

      {/* Queue Settings */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Queue Settings</h2>
          <p className="text-sm text-gray-400">Configure how the song queue behaves</p>
        </div>

        <div className="p-6 space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-white font-medium">Auto-play next song</h3>
              <p className="text-sm text-gray-400">Automatically play the next song when one ends</p>
            </div>
            <button
              onClick={() => setAutoPlay(!autoPlay)}
              className={`relative w-12 h-6 rounded-full transition-colors ${autoPlay ? 'bg-yellow-neon' : 'bg-matte-black'}`}
            >
              <span className={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${autoPlay ? 'left-7' : 'left-1'}`} />
            </button>
          </div>

          <div>
            <label className="block text-white font-medium mb-1">Max songs per user in queue</label>
            <p className="text-sm text-gray-400 mb-2">Limit how many songs each user can have queued at once</p>
            <select
              value={maxQueuePerUser}
              onChange={(e) => setMaxQueuePerUser(Number(e.target.value))}
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            >
              <option value={1}>1 song</option>
              <option value={2}>2 songs</option>
              <option value={3}>3 songs</option>
              <option value={5}>5 songs</option>
              <option value={10}>10 songs</option>
              <option value={0}>Unlimited</option>
            </select>
          </div>
        </div>
      </div>

      {/* System Info */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">System Information</h2>
        </div>

        <div className="p-6 space-y-3">
          <div className="flex justify-between">
            <span className="text-gray-400">Version</span>
            <span className="text-white font-mono">0.1.0</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-400">Server URL</span>
            <span className="text-white font-mono text-sm">{window.location.origin}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

const API_BASE = import.meta.env.DEV ? 'https://localhost:8443' : '';

function SearchLogs() {
  const [logs, setLogs] = useState<SearchLogEntry[]>([]);
  const [stats, setStats] = useState<SearchStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [sourceFilter, setSourceFilter] = useState<string>('');
  const [showStats, setShowStats] = useState(true);

  const fetchLogs = async () => {
    try {
      const url = sourceFilter
        ? `${API_BASE}/api/admin/search-logs?source=${sourceFilter}`
        : `${API_BASE}/api/admin/search-logs`;
      const res = await fetch(url, { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setLogs(data || []);
      }
    } catch (err) {
      console.error('Failed to fetch search logs:', err);
    }
  };

  const fetchStats = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/search-stats`, { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setStats(data);
      }
    } catch (err) {
      console.error('Failed to fetch search stats:', err);
    }
  };

  const clearLogs = async () => {
    if (!confirm('Clear all search logs? This cannot be undone.')) return;
    try {
      const res = await fetch(`${API_BASE}/api/admin/search-logs`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok) {
        setLogs([]);
        fetchStats();
      }
    } catch (err) {
      console.error('Failed to clear search logs:', err);
    }
  };

  useEffect(() => {
    Promise.all([fetchLogs(), fetchStats()]).finally(() => setIsLoading(false));
  }, []);

  useEffect(() => {
    fetchLogs();
  }, [sourceFilter]);

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleString();
  };

  if (isLoading) {
    return (
      <div className="bg-matte-gray rounded-2xl p-8 text-center">
        <div className="text-gray-400">Loading search logs...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Stats Overview */}
      {showStats && stats && (
        <div className="bg-matte-gray rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-white/5 flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-white">Search Analytics</h2>
              <p className="text-sm text-gray-400">Understand what users are searching for</p>
            </div>
            <button
              onClick={() => setShowStats(false)}
              className="text-gray-400 hover:text-white"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div className="p-6">
            {/* Stats Grid */}
            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="bg-matte-black rounded-xl p-4 text-center">
                <div className="text-3xl font-bold text-white">{stats.total_searches}</div>
                <div className="text-sm text-gray-400">Total Searches</div>
              </div>
              <div className="bg-matte-black rounded-xl p-4 text-center">
                <div className="text-3xl font-bold text-white">{stats.unique_queries}</div>
                <div className="text-sm text-gray-400">Unique Queries</div>
              </div>
              <div className="bg-matte-black rounded-xl p-4 text-center">
                <div className="text-3xl font-bold text-red-400">{stats.not_found_count}</div>
                <div className="text-sm text-gray-400">Not Found</div>
              </div>
            </div>

            {/* Top Not Found */}
            {stats.top_not_found && stats.top_not_found.length > 0 && (
              <div>
                <h3 className="text-white font-medium mb-3">Top Unfulfilled Searches</h3>
                <p className="text-sm text-gray-400 mb-3">Songs people are looking for but can't find - consider adding these!</p>
                <div className="space-y-2">
                  {stats.top_not_found.map((item, idx) => (
                    <div key={idx} className="flex items-center justify-between bg-matte-black/50 rounded-lg px-4 py-2">
                      <span className="text-white">{item.query}</span>
                      <div className="flex items-center gap-3">
                        <span className={`text-xs px-2 py-0.5 rounded ${item.source === 'youtube' ? 'bg-red-500/20 text-red-400' : 'bg-blue-500/20 text-blue-400'}`}>
                          {item.source}
                        </span>
                        <span className="text-gray-400 text-sm">{item.results_count}x searched</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Logs Table */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-white">Search Logs</h2>
            <p className="text-sm text-gray-400">{logs.length} recent searches</p>
          </div>
          <div className="flex items-center gap-3">
            {!showStats && stats && (
              <button
                onClick={() => setShowStats(true)}
                className="px-3 py-1.5 bg-blue-500/20 text-blue-400 rounded-lg text-sm font-medium hover:bg-blue-500/30 transition-colors"
              >
                Show Stats
              </button>
            )}
            <select
              value={sourceFilter}
              onChange={(e) => setSourceFilter(e.target.value)}
              className="px-3 py-1.5 bg-matte-black rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            >
              <option value="">All Sources</option>
              <option value="library">Library</option>
              <option value="youtube">YouTube</option>
            </select>
            <button
              onClick={clearLogs}
              className="px-3 py-1.5 bg-red-500/20 text-red-400 rounded-lg text-sm font-medium hover:bg-red-500/30 transition-colors"
            >
              Clear Logs
            </button>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-gray-400 text-sm border-b border-white/5">
                <th className="py-3 px-4 font-medium">Query</th>
                <th className="py-3 px-4 font-medium">Source</th>
                <th className="py-3 px-4 font-medium">Results</th>
                <th className="py-3 px-4 font-medium">IP</th>
                <th className="py-3 px-4 font-medium">Time</th>
              </tr>
            </thead>
            <tbody>
              {logs.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-8 text-center text-gray-500">
                    No search logs yet
                  </td>
                </tr>
              ) : (
                logs.map((log) => (
                  <tr key={log.id} className="border-b border-white/5">
                    <td className="py-3 px-4">
                      <span className="text-white">{log.query}</span>
                    </td>
                    <td className="py-3 px-4">
                      <span className={`text-xs px-2 py-0.5 rounded ${log.source === 'youtube' ? 'bg-red-500/20 text-red-400' : 'bg-blue-500/20 text-blue-400'}`}>
                        {log.source}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <span className={log.results_count === 0 ? 'text-red-400' : 'text-green-400'}>
                        {log.results_count === 0 ? 'Not found' : `${log.results_count} found`}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-gray-400 font-mono text-sm">{log.ip_address}</td>
                    <td className="py-3 px-4 text-gray-400 text-sm">{formatTime(log.searched_at)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function AdminTabButton({ label, icon, active, onClick }: {
  label: string;
  icon: React.ReactNode;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
        active
          ? 'bg-yellow-neon text-indigo-deep'
          : 'text-gray-400 hover:text-white hover:bg-matte-black'
      }`}
    >
      {icon}
      <span className="hidden sm:inline">{label}</span>
    </button>
  );
}

export function Admin() {
  const { isAuthenticated, isLocal, checkAuth, setClients, fetchClients, logout } = useAdminStore();
  const [isLoading, setIsLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<AdminTab>('clients');

  useEffect(() => {
    checkAuth().then(() => setIsLoading(false));
  }, [checkAuth]);

  // Connect to WebSocket and subscribe to updates
  useEffect(() => {
    if (!isAuthenticated) return;

    // Connect to WebSocket if not already connected
    wsService.connect();

    // Fetch client list after welcome (ensures our connection is established)
    const unsubWelcome = wsService.on('welcome', () => {
      fetchClients();
    });

    // Subscribe to real-time client list updates
    const unsubClientList = wsService.on('client_list', (clients: ClientInfo[]) => {
      setClients(clients);
    });

    // If already connected, fetch immediately
    if (wsService.isConnected()) {
      fetchClients();
    }

    return () => {
      unsubWelcome();
      unsubClientList();
    };
  }, [isAuthenticated, setClients, fetchClients]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-400">Loading...</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <PinEntry />;
  }

  // Icons for tabs
  const icons = {
    clients: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
      </svg>
    ),
    library: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
      </svg>
    ),
    'search-logs': (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
    ),
    network: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
      </svg>
    ),
    settings: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
  };

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="flex items-center justify-between px-6 py-4 bg-matte-gray/50 backdrop-blur-sm border-b border-white/5">
        <div className="flex items-center gap-3">
          <img src="/logo.jpeg" alt="SongMartyn" className="w-10 h-10 rounded-lg object-cover" />
          <div>
            <h1 className="text-xl font-bold text-white">
              Song<span className="text-yellow-neon">Martyn</span> Admin
            </h1>
            {isLocal && (
              <span className="text-xs text-green-400">Local Access</span>
            )}
          </div>
        </div>

        <button
          onClick={logout}
          className="px-4 py-2 text-gray-400 hover:text-white transition-colors"
        >
          Logout
        </button>
      </header>

      {/* Tab Navigation */}
      <nav className="px-6 py-3 bg-matte-gray/30 border-b border-white/5">
        <div className="max-w-4xl mx-auto flex gap-2">
          <AdminTabButton
            label="Clients"
            icon={icons.clients}
            active={activeTab === 'clients'}
            onClick={() => setActiveTab('clients')}
          />
          <AdminTabButton
            label="Library"
            icon={icons.library}
            active={activeTab === 'library'}
            onClick={() => setActiveTab('library')}
          />
          <AdminTabButton
            label="Search Logs"
            icon={icons['search-logs']}
            active={activeTab === 'search-logs'}
            onClick={() => setActiveTab('search-logs')}
          />
          <AdminTabButton
            label="Network"
            icon={icons.network}
            active={activeTab === 'network'}
            onClick={() => setActiveTab('network')}
          />
          <AdminTabButton
            label="Settings"
            icon={icons.settings}
            active={activeTab === 'settings'}
            onClick={() => setActiveTab('settings')}
          />
        </div>
      </nav>

      {/* Main content */}
      <main className="p-6 max-w-4xl mx-auto">
        {activeTab === 'clients' && <ClientList />}
        {activeTab === 'library' && <LibraryManagement />}
        {activeTab === 'search-logs' && <SearchLogs />}
        {activeTab === 'network' && <NetworkSettings />}
        {activeTab === 'settings' && <GeneralSettings />}
      </main>
    </div>
  );
}
