import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useAdminStore } from '../stores/adminStore';
import { useLibraryStore } from '../stores/libraryStore';
import { useRoomStore, selectQueue, selectQueuePosition, selectAutoplay, selectCountdown } from '../stores/roomStore';
import { useWebSocket } from '../hooks/useWebSocket';
import { wsService } from '../services/websocket';
import type { ClientInfo, LibraryLocation, AvatarConfig, BGMSourceType, IcecastStream } from '../types';
import { HelpModal, HelpButton, useHelpModal } from '../components/HelpModal';
import { buildAvatarUrl } from '../components/AvatarCreator';

type AdminTab = 'clients' | 'queue' | 'library' | 'search-logs' | 'network' | 'settings';

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

// Block duration options in minutes
const BLOCK_DURATIONS = [
  { value: 5, label: '5 minutes' },
  { value: 15, label: '15 minutes' },
  { value: 60, label: '1 hour' },
  { value: 1440, label: '24 hours' },
  { value: 10080, label: '1 week' },
  { value: 0, label: 'Permanent' },
];

// Avatar component for client list
function ClientAvatar({ config, size = 32 }: { config?: AvatarConfig; size?: number }) {
  if (!config) {
    return (
      <div
        className="rounded-full bg-gray-600 flex items-center justify-center flex-shrink-0"
        style={{ width: size, height: size }}
      >
        <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
    );
  }

  const avatarUrl = buildAvatarUrl(config);
  return (
    <img
      src={avatarUrl}
      alt="User avatar"
      className="rounded-full flex-shrink-0"
      style={{ width: size, height: size }}
    />
  );
}

function KickBlockModal({ client, onClose }: {
  client: ClientInfo;
  onClose: () => void;
}) {
  const { kickClient, blockClient } = useAdminStore();
  const [action, setAction] = useState<'kick' | 'block'>('kick');
  const [blockDuration, setBlockDuration] = useState(60);
  const [reason, setReason] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);

  const handleSubmit = async () => {
    setIsProcessing(true);
    if (action === 'kick') {
      await kickClient(client.martyn_key, reason);
    } else {
      await blockClient(client.martyn_key, blockDuration, reason);
    }
    setIsProcessing(false);
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80" onClick={onClose}>
      <div className="bg-matte-gray rounded-2xl p-6 w-full max-w-md" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-white">Moderate User</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* User Info */}
        <div className="bg-matte-black rounded-xl p-3 mb-4">
          <div className="flex items-center gap-3">
            <ClientAvatar config={client.avatar_config} size={40} />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full flex-shrink-0 ${client.is_online ? 'bg-green-400' : 'bg-gray-500'}`} />
                <span className="text-white font-medium truncate">{client.display_name}</span>
              </div>
              <div className="text-gray-400 text-sm truncate">{client.device_name} - {client.ip_address}</div>
            </div>
          </div>
        </div>

        {/* Action Toggle */}
        <div className="flex gap-2 mb-4">
          <button
            onClick={() => setAction('kick')}
            className={`flex-1 py-2 px-4 rounded-lg text-sm font-medium transition-colors ${
              action === 'kick'
                ? 'bg-orange-500/20 text-orange-400 border border-orange-500/50'
                : 'bg-matte-black text-gray-400 hover:text-white'
            }`}
          >
            Kick
          </button>
          <button
            onClick={() => setAction('block')}
            className={`flex-1 py-2 px-4 rounded-lg text-sm font-medium transition-colors ${
              action === 'block'
                ? 'bg-red-500/20 text-red-400 border border-red-500/50'
                : 'bg-matte-black text-gray-400 hover:text-white'
            }`}
          >
            Block
          </button>
        </div>

        {/* Block Duration (only shown for block action) */}
        {action === 'block' && (
          <div className="mb-4">
            <label className="block text-sm text-gray-400 mb-2">Block Duration</label>
            <div className="grid grid-cols-3 gap-2">
              {BLOCK_DURATIONS.map(({ value, label }) => (
                <button
                  key={value}
                  onClick={() => setBlockDuration(value)}
                  className={`py-2 px-3 rounded-lg text-sm font-medium transition-colors ${
                    blockDuration === value
                      ? 'bg-red-500/30 text-red-300 border border-red-500/50'
                      : 'bg-matte-black text-gray-400 hover:text-white'
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Reason Input */}
        <div className="mb-6">
          <label className="block text-sm text-gray-400 mb-2">Reason (optional)</label>
          <input
            type="text"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder={action === 'kick' ? 'e.g., Disruptive behavior' : 'e.g., Repeated violations'}
            className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
          />
        </div>

        {/* Action Description */}
        <div className="mb-4 text-sm text-gray-400">
          {action === 'kick' ? (
            <p>This will disconnect {client.display_name} from the session. They can reconnect immediately.</p>
          ) : (
            <p>This will disconnect and block {client.display_name} for {BLOCK_DURATIONS.find(d => d.value === blockDuration)?.label.toLowerCase()}. They won't be able to reconnect until the block expires.</p>
          )}
        </div>

        {/* Submit Button */}
        <button
          onClick={handleSubmit}
          disabled={isProcessing}
          className={`w-full py-3 font-semibold rounded-xl transition-colors disabled:opacity-50 ${
            action === 'kick'
              ? 'bg-orange-500 text-white hover:bg-orange-600'
              : 'bg-red-500 text-white hover:bg-red-600'
          }`}
        >
          {isProcessing ? 'Processing...' : action === 'kick' ? 'Kick User' : 'Block User'}
        </button>
      </div>
    </div>
  );
}

function ClientRow({ client, onToggleAdmin, onToggleAFK, onAction, onUnblock }: {
  client: ClientInfo;
  onToggleAdmin: () => void;
  onToggleAFK: () => void;
  onAction: () => void;
  onUnblock: () => void;
}) {
  return (
    <tr className={`border-b border-white/5 ${client.is_blocked ? 'bg-red-500/5' : ''}`}>
      <td className="py-3 px-4">
        <div className="flex items-center gap-3">
          <ClientAvatar config={client.avatar_config} size={32} />
          <div className="flex items-center gap-2 min-w-0">
            <div className={`w-2 h-2 rounded-full flex-shrink-0 ${
              client.is_blocked ? 'bg-red-500' :
              client.is_online ? 'bg-green-400' : 'bg-gray-500'
            }`} />
            <span className="text-white font-medium truncate">{client.display_name}</span>
            {client.is_afk && !client.is_blocked && (
              <span className="px-2 py-0.5 bg-orange-500/20 text-orange-400 text-xs rounded flex-shrink-0">
                AFK
              </span>
            )}
            {client.is_blocked && (
              <span className="px-2 py-0.5 bg-red-500/20 text-red-400 text-xs rounded flex-shrink-0" title={client.block_reason}>
                Blocked
              </span>
            )}
          </div>
        </div>
      </td>
      <td className="py-3 px-4 text-gray-400">{client.device_name || 'Unknown'}</td>
      <td className="py-3 px-4 text-gray-400 font-mono text-sm">{client.ip_address}</td>
      <td className="py-3 px-4">
        {!client.is_blocked && (
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
        )}
      </td>
      <td className="py-3 px-4">
        <div className="flex items-center gap-2">
          {!client.is_blocked && client.is_online && (
            <button
              onClick={onToggleAFK}
              className={`px-3 py-1 rounded-lg text-sm font-medium transition-colors ${
                client.is_afk
                  ? 'bg-orange-500/20 text-orange-400 hover:bg-orange-500/30'
                  : 'bg-matte-black text-gray-400 hover:text-orange-400'
              }`}
              title={client.is_afk ? 'Mark as present' : 'Mark as AFK'}
            >
              {client.is_afk ? 'Back' : 'AFK'}
            </button>
          )}
          {client.is_blocked ? (
            <button
              onClick={onUnblock}
              className="px-3 py-1 bg-green-500/20 text-green-400 rounded-lg text-sm font-medium hover:bg-green-500/30 transition-colors"
            >
              Unblock
            </button>
          ) : (
            <button
              onClick={onAction}
              className="px-3 py-1 bg-red-500/20 text-red-400 rounded-lg text-sm font-medium hover:bg-red-500/30 transition-colors"
            >
              Actions
            </button>
          )}
        </div>
      </td>
    </tr>
  );
}

function ClientList() {
  const { clients, setAdminStatus, setAFKStatus, unblockClient } = useAdminStore();
  const [modalClient, setModalClient] = useState<ClientInfo | null>(null);

  const handleToggleAdmin = async (client: ClientInfo) => {
    await setAdminStatus(client.martyn_key, !client.is_admin);
  };

  const handleToggleAFK = async (client: ClientInfo) => {
    await setAFKStatus(client.martyn_key, !client.is_afk);
  };

  const handleUnblock = async (client: ClientInfo) => {
    if (confirm(`Unblock ${client.display_name}?`)) {
      await unblockClient(client.martyn_key);
    }
  };

  const onlineClients = clients.filter((c) => c.is_online && !c.is_blocked);
  const offlineClients = clients.filter((c) => !c.is_online && !c.is_blocked);
  const blockedClients = clients.filter((c) => c.is_blocked);

  return (
    <>
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Connected Clients</h2>
          <p className="text-sm text-gray-400">
            {onlineClients.length} online, {offlineClients.length} offline
            {blockedClients.length > 0 && `, ${blockedClients.length} blocked`}
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
                    onToggleAFK={() => handleToggleAFK(client)}
                    onAction={() => setModalClient(client)}
                    onUnblock={() => handleUnblock(client)}
                  />
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {modalClient && (
        <KickBlockModal
          client={modalClient}
          onClose={() => setModalClient(null)}
        />
      )}
    </>
  );
}

const API_BASE = import.meta.env.DEV ? 'https://localhost:8443' : '';

function LibraryManagement() {
  const { locations, stats, isLoading, error, fetchLocations, fetchStats, addLocation, removeLocation, scanLocation } = useLibraryStore();
  const [showAddForm, setShowAddForm] = useState(false);
  const [newPath, setNewPath] = useState('');
  const [newName, setNewName] = useState('');
  const [scanningId, setScanningId] = useState<number | null>(null);
  const { activeHelp, openHelp, closeHelp } = useHelpModal();

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
    <>
    <div className="bg-matte-gray rounded-2xl overflow-hidden">
      <div className="px-6 py-4 border-b border-white/5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div>
              <h2 className="text-lg font-semibold text-white">Song Library</h2>
              <p className="text-sm text-gray-400">
                {stats.total_songs} songs, {stats.total_plays} total plays
              </p>
            </div>
            <HelpButton onClick={() => openHelp('library')} />
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
    {activeHelp && (
      <HelpModal topic={activeHelp} isOpen={true} onClose={closeHelp} />
    )}
    </>
  );
}

interface NetworkInterface {
  name: string;
  display_name: string;
  type: string;
  mac_address: string;
  ipv4: string[];
  ipv6: string[];
  is_up: boolean;
  is_loopback: boolean;
  is_wireless: boolean;
  connect_urls: string[];
}

function NetworkSettings() {
  const [networks, setNetworks] = useState<NetworkInterface[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedUrl, setSelectedUrl] = useState<string | null>(null);
  const [savedUrl, setSavedUrl] = useState<string | null>(null);
  const [showQRModal, setShowQRModal] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const { activeHelp, openHelp, closeHelp } = useHelpModal();

  // Fetch networks and saved URL when this tab is active
  useEffect(() => {
    setIsLoading(true);
    Promise.all([
      fetch(`${API_BASE}/api/admin/networks`, { credentials: 'include' }).then(r => r.json()),
      fetch(`${API_BASE}/api/connect-url`, { credentials: 'include' }).then(r => r.json()),
    ]).then(([networksData, connectData]) => {
      setNetworks(networksData || []);
      const currentUrl = connectData?.url || null;
      setSavedUrl(currentUrl);
      setSelectedUrl(currentUrl);
    }).catch(err => console.error('Failed to load network data:', err))
      .finally(() => setIsLoading(false));
  }, []);

  const saveSelectedUrl = async (url: string) => {
    setIsSaving(true);
    try {
      const res = await fetch(`${API_BASE}/api/connect-url`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ url }),
      });
      if (res.ok) {
        setSavedUrl(url);
        setSelectedUrl(url);
      }
    } catch (err) {
      console.error('Failed to save URL:', err);
    } finally {
      setIsSaving(false);
    }
  };

  const getTypeIcon = (type: string, isWireless: boolean) => {
    if (isWireless || type === 'wireless' || type === 'wifi_or_ethernet') {
      return (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
        </svg>
      );
    }
    if (type === 'ethernet') {
      return (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
        </svg>
      );
    }
    if (type === 'vpn') {
      return (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        </svg>
      );
    }
    return (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
      </svg>
    );
  };

  // Simple QR code generator using a data URL approach
  const generateQRCodeUrl = (url: string) => {
    // Use a public QR code API for simplicity
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(url)}`;
  };

  return (
    <>
    <div className="space-y-6">
      {/* Network Interfaces */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold text-white">Network Interfaces</h2>
            <HelpButton onClick={() => openHelp('network')} />
          </div>
          <p className="text-sm text-gray-400">Available network connections for client access</p>
        </div>

        {isLoading ? (
          <div className="p-8 text-center">
            <div className="inline-flex items-center gap-3 text-gray-400">
              <svg className="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              <span>Enumerating networks...</span>
            </div>
          </div>
        ) : networks.filter(n => !n.is_loopback).length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            No network interfaces found
          </div>
        ) : (
          <div className="divide-y divide-white/5">
            {networks.filter(n => !n.is_loopback).map((network) => (
              <div key={network.name} className="p-4">
                <div className="flex items-start gap-4">
                  <div className={`p-2 rounded-lg ${network.is_wireless ? 'bg-blue-500/20 text-blue-400' : 'bg-gray-500/20 text-gray-400'}`}>
                    {getTypeIcon(network.type, network.is_wireless)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h3 className="text-white font-medium">{network.display_name}</h3>
                      <span className="text-xs text-gray-500 font-mono">({network.name})</span>
                      {network.is_up && (
                        <span className="px-2 py-0.5 bg-green-500/20 text-green-400 text-xs rounded">Active</span>
                      )}
                    </div>

                    {/* IPv4 Addresses */}
                    {network.ipv4.length > 0 && (
                      <div className="mt-2">
                        <div className="text-xs text-gray-500 mb-1">IPv4</div>
                        <div className="flex flex-wrap gap-2">
                          {network.ipv4.map((ip, i) => (
                            <span key={i} className="text-white font-mono text-sm bg-matte-black px-2 py-1 rounded">
                              {ip}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Connection URLs */}
                    {network.connect_urls.length > 0 && (
                      <div className="mt-3">
                        <div className="text-xs text-gray-500 mb-1">Connection URLs</div>
                        <div className="space-y-2">
                          {network.connect_urls.map((url, i) => (
                            <div key={i} className="flex items-center gap-2">
                              <button
                                onClick={() => {
                                  setSelectedUrl(url);
                                  setShowQRModal(true);
                                }}
                                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                                  savedUrl === url
                                    ? 'bg-yellow-neon/20 text-yellow-neon'
                                    : 'bg-matte-black text-gray-300 hover:text-white'
                                }`}
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
                                </svg>
                                {url}
                                {savedUrl === url && (
                                  <span className="ml-1 text-xs bg-yellow-neon/30 px-1.5 py-0.5 rounded">Default</span>
                                )}
                              </button>
                              {savedUrl !== url && (
                                <button
                                  onClick={() => saveSelectedUrl(url)}
                                  disabled={isSaving}
                                  className="px-2 py-1 text-xs bg-blue-500/20 text-blue-400 rounded hover:bg-blue-500/30 transition-colors disabled:opacity-50"
                                >
                                  {isSaving ? '...' : 'Set as Default'}
                                </button>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* MAC Address */}
                    {network.mac_address && (
                      <div className="mt-2 text-xs text-gray-500 font-mono">
                        MAC: {network.mac_address}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* QR Code Modal */}
      {showQRModal && selectedUrl && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80" onClick={() => setShowQRModal(false)}>
          <div className="bg-matte-gray rounded-2xl p-6 max-w-sm w-full" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Scan to Connect</h3>
              <button onClick={() => setShowQRModal(false)} className="text-gray-400 hover:text-white">
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="bg-white p-4 rounded-xl mb-4 flex items-center justify-center">
              <img
                src={generateQRCodeUrl(selectedUrl)}
                alt="QR Code"
                className="w-48 h-48"
              />
            </div>

            <div className="text-center">
              <p className="text-gray-400 text-sm mb-2">Scan with your phone camera</p>
              <code className="text-yellow-neon text-sm font-mono bg-matte-black px-3 py-2 rounded-lg block overflow-x-auto">
                {selectedUrl}
              </code>
            </div>
          </div>
        </div>
      )}

      {/* Quick Access QR */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Guest Access QR Code</h2>
          <p className="text-sm text-gray-400">This QR code will be shown in the app header</p>
        </div>

        <div className="p-6">
          {savedUrl ? (
            <div className="flex items-center gap-4">
              <div className="bg-white p-3 rounded-xl">
                <img
                  src={generateQRCodeUrl(savedUrl)}
                  alt="QR Code"
                  className="w-24 h-24"
                />
              </div>
              <div className="flex-1">
                <p className="text-white font-medium mb-1">Default Connection URL</p>
                <code className="text-yellow-neon text-sm font-mono">{savedUrl}</code>
                <p className="text-gray-400 text-sm mt-2">
                  Click "Set as Default" next to any URL above to change
                </p>
                <button
                  onClick={() => {
                    setSelectedUrl(savedUrl);
                    setShowQRModal(true);
                  }}
                  className="mt-3 px-4 py-2 bg-yellow-neon text-indigo-deep font-semibold rounded-lg hover:scale-[1.02] transition-transform"
                >
                  Show Full QR Code
                </button>
              </div>
            </div>
          ) : (
            <div className="text-center text-gray-500 py-4">
              No connection URL set. Click "Set as Default" on a URL above.
            </div>
          )}
        </div>
      </div>
    </div>
    {activeHelp && (
      <HelpModal topic={activeHelp} isOpen={true} onClose={closeHelp} />
    )}
    </>
  );
}

interface ServerSettings {
  https_port: string;
  http_port: string;
  admin_pin: string;
  youtube_api_key: string;
  video_player: string;
  data_dir: string;
}

interface SystemInfo {
  os: string;
  arch: string;
  hostname: string;
  cpu_count: number;
  memory_total: number;
  memory_free: number;
  memory_used: number;
  disk_total: number;
  disk_free: number;
  disk_used: number;
  go_version: string;
  server_uptime: string;
  network_addrs: string[];
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function GeneralSettings() {
  const [settings, setSettings] = useState<ServerSettings>({
    https_port: '8443',
    http_port: '8080',
    admin_pin: '',
    youtube_api_key: '',
    video_player: 'mpv',
    data_dir: './data',
  });
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const { activeHelp, openHelp, closeHelp } = useHelpModal();
  const [isSaving, setIsSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [playerRunning, setPlayerRunning] = useState(false);
  const [playerLoading, setPlayerLoading] = useState(false);
  const [dbStats, setDbStats] = useState<{
    sessions: { count: number };
    blocked_users: { count: number };
    search_logs: { count: number };
    song_history: { count: number };
    queue: { count: number };
  } | null>(null);
  const [flushingAction, setFlushingAction] = useState<string | null>(null);

  // BGM (Background Music) settings
  const [bgmSettings, setBgmSettings] = useState<{
    enabled: boolean;
    source_type: BGMSourceType;
    url: string;
    volume: number;
  }>({
    enabled: false,
    source_type: 'youtube',
    url: '',
    volume: 50,
  });
  const [bgmSaving, setBgmSaving] = useState(false);
  const [icecastStreams, setIcecastStreams] = useState<IcecastStream[]>([]);
  const [streamsLoading, setStreamsLoading] = useState(false);
  const [showStreamPicker, setShowStreamPicker] = useState(false);

  const fetchBgmSettings = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/bgm`, { credentials: 'include' });
      if (res.ok) {
        setBgmSettings(await res.json());
      }
    } catch (err) {
      console.error('Failed to fetch BGM settings:', err);
    }
  };

  const saveBgmSettings = async () => {
    setBgmSaving(true);
    try {
      const res = await fetch(`${API_BASE}/api/admin/bgm`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(bgmSettings),
      });
      if (res.ok) {
        setMessage({ type: 'success', text: 'BGM settings saved' });
      } else {
        setMessage({ type: 'error', text: 'Failed to save BGM settings' });
      }
    } catch (err) {
      setMessage({ type: 'error', text: 'Failed to save BGM settings' });
    } finally {
      setBgmSaving(false);
    }
  };

  const fetchIcecastStreams = async () => {
    setStreamsLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/admin/icecast-streams`, { credentials: 'include' });
      if (res.ok) {
        const streams = await res.json();
        setIcecastStreams(streams);
        setShowStreamPicker(true);
      } else {
        setMessage({ type: 'error', text: 'Failed to fetch Icecast streams' });
      }
    } catch (err) {
      setMessage({ type: 'error', text: 'Failed to fetch Icecast streams' });
    } finally {
      setStreamsLoading(false);
    }
  };

  const selectIcecastStream = (stream: IcecastStream) => {
    setBgmSettings({
      ...bgmSettings,
      source_type: 'icecast',
      url: stream.url,
    });
    setShowStreamPicker(false);
  };

  const fetchDbStats = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/database`, { credentials: 'include' });
      if (res.ok) {
        setDbStats(await res.json());
      }
    } catch (err) {
      console.error('Failed to fetch database stats:', err);
    }
  };

  const flushData = async (action: string, confirmMessage: string) => {
    if (!confirm(confirmMessage)) return;
    setFlushingAction(action);
    try {
      const res = await fetch(`${API_BASE}/api/admin/database`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ action }),
      });
      const data = await res.json();
      if (res.ok) {
        setMessage({ type: 'success', text: data.message });
        fetchDbStats();
      } else {
        setMessage({ type: 'error', text: data.error || 'Operation failed' });
      }
    } catch (err) {
      setMessage({ type: 'error', text: 'Failed to perform operation' });
    } finally {
      setFlushingAction(null);
    }
  };

  const checkPlayerStatus = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/player`, { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setPlayerRunning(data.is_running);
      }
    } catch (err) {
      console.error('Failed to check player status:', err);
    }
  };

  const launchPlayer = async (action: 'launch' | 'restart') => {
    setPlayerLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/admin/player`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ action }),
      });
      const data = await res.json();
      if (res.ok) {
        setPlayerRunning(data.is_running);
      }
    } catch (err) {
      console.error('Failed to launch player:', err);
    } finally {
      setPlayerLoading(false);
    }
  };

  useEffect(() => {
    Promise.all([
      fetch(`${API_BASE}/api/admin/settings`, { credentials: 'include' }).then(r => r.json()),
      fetch(`${API_BASE}/api/admin/system-info`, { credentials: 'include' }).then(r => r.json()),
    ]).then(([settingsData, sysInfo]) => {
      setSettings(settingsData);
      setSystemInfo(sysInfo);
    }).catch(err => {
      console.error('Failed to load settings:', err);
    }).finally(() => {
      setIsLoading(false);
    });
    checkPlayerStatus();
    fetchDbStats();
    fetchBgmSettings();
    // Poll player status every 5 seconds
    const interval = setInterval(checkPlayerStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setMessage(null);

    try {
      const res = await fetch(`${API_BASE}/api/admin/settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(settings),
      });
      const data = await res.json();
      if (res.ok) {
        setMessage({ type: 'success', text: data.message || 'Settings saved successfully.' });
      } else {
        setMessage({ type: 'error', text: data.error || 'Failed to save settings.' });
      }
    } catch {
      setMessage({ type: 'error', text: 'Failed to save settings.' });
    } finally {
      setIsSaving(false);
    }
  };

  if (isLoading) {
    return (
      <div className="bg-matte-gray rounded-2xl p-8 text-center">
        <div className="text-gray-400">Loading settings...</div>
      </div>
    );
  }

  return (
    <>
    <div className="space-y-6">
      {/* Server Configuration */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold text-white">Server Configuration</h2>
            <HelpButton onClick={() => openHelp('certificates')} />
          </div>
          <p className="text-sm text-gray-400">Changes require server restart to take effect</p>
        </div>

        <form onSubmit={handleSave} className="p-6 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-gray-400 mb-1">HTTPS Port</label>
              <input
                type="text"
                value={settings.https_port}
                onChange={(e) => setSettings({ ...settings, https_port: e.target.value })}
                placeholder="8443"
                className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">HTTP Port</label>
              <input
                type="text"
                value={settings.http_port}
                onChange={(e) => setSettings({ ...settings, http_port: e.target.value })}
                placeholder="8080"
                className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
              />
            </div>
          </div>

          <div>
            <div className="flex items-center gap-2 mb-1">
              <label className="block text-sm text-gray-400">Admin PIN</label>
              <HelpButton onClick={() => openHelp('adminPin')} />
            </div>
            <p className="text-xs text-gray-500 mb-2">Leave empty for localhost-only access</p>
            <input
              type="text"
              value={settings.admin_pin}
              onChange={(e) => setSettings({ ...settings, admin_pin: e.target.value })}
              placeholder="Enter PIN for remote admin access"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            />
          </div>

          <div>
            <div className="flex items-center gap-2 mb-1">
              <label className="block text-sm text-gray-400">YouTube API Key</label>
              <HelpButton onClick={() => openHelp('youtubeApi')} />
            </div>
            <p className="text-xs text-gray-500 mb-2">Leave empty to disable YouTube search</p>
            <input
              type="text"
              value={settings.youtube_api_key}
              onChange={(e) => setSettings({ ...settings, youtube_api_key: e.target.value })}
              placeholder="Enter YouTube Data API v3 key"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon font-mono text-sm"
            />
          </div>

          <div>
            <div className="flex items-center gap-2 mb-1">
              <label className="block text-sm text-gray-400">Video Player Path</label>
              <HelpButton onClick={() => openHelp('videoPlayer')} />
            </div>
            <input
              type="text"
              value={settings.video_player}
              onChange={(e) => setSettings({ ...settings, video_player: e.target.value })}
              placeholder="mpv"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon font-mono"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Data Directory</label>
            <input
              type="text"
              value={settings.data_dir}
              onChange={(e) => setSettings({ ...settings, data_dir: e.target.value })}
              placeholder="./data"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon font-mono"
            />
          </div>

          {message && (
            <div className={`p-3 rounded-lg text-sm ${message.type === 'success' ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
              {message.text}
            </div>
          )}

          <button
            type="submit"
            disabled={isSaving}
            className="w-full py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform disabled:opacity-50"
          >
            {isSaving ? 'Saving...' : 'Save Settings'}
          </button>
        </form>
      </div>

      {/* Media Player Control */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Media Player</h2>
          <p className="text-sm text-gray-400">Launch and position the video player window before karaoke begins</p>
        </div>

        <div className="p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              {/* Status Indicator */}
              <div className={`flex items-center gap-2 px-3 py-2 rounded-lg ${
                playerRunning ? 'bg-green-500/20' : 'bg-gray-500/20'
              }`}>
                <div className={`w-3 h-3 rounded-full ${
                  playerRunning ? 'bg-green-400 animate-pulse' : 'bg-gray-500'
                }`} />
                <span className={`text-sm font-medium ${
                  playerRunning ? 'text-green-400' : 'text-gray-400'
                }`}>
                  {playerRunning ? 'Running' : 'Stopped'}
                </span>
              </div>

              <div className="text-sm text-gray-400">
                {playerRunning
                  ? 'Player is active. Position the window and set to fullscreen.'
                  : 'Launch the player to position it before starting.'}
              </div>
            </div>

            <div className="flex gap-2">
              {playerRunning ? (
                <button
                  onClick={() => launchPlayer('restart')}
                  disabled={playerLoading}
                  className="px-4 py-2 bg-orange-500/20 text-orange-400 font-medium rounded-lg hover:bg-orange-500/30 transition-colors disabled:opacity-50"
                >
                  {playerLoading ? 'Restarting...' : 'Restart Player'}
                </button>
              ) : (
                <button
                  onClick={() => launchPlayer('launch')}
                  disabled={playerLoading}
                  className="px-4 py-2 bg-yellow-neon text-indigo-deep font-semibold rounded-lg hover:scale-[1.02] transition-transform disabled:opacity-50"
                >
                  {playerLoading ? 'Launching...' : 'Launch Player'}
                </button>
              )}
            </div>
          </div>

          <div className="mt-4 p-4 bg-matte-black/50 rounded-xl">
            <h4 className="text-white text-sm font-medium mb-2">Setup Tips</h4>
            <ul className="text-xs text-gray-400 space-y-1">
              <li>1. Click "Launch Player" to open the MPV video player</li>
              <li>2. Drag the window to your TV/projector display</li>
              <li>3. Press <kbd className="px-1.5 py-0.5 bg-matte-black rounded text-gray-300">F</kbd> to toggle fullscreen</li>
              <li>4. The player will stay open and play songs from the queue</li>
            </ul>
          </div>
        </div>
      </div>

      {/* System Information */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">System Information</h2>
        </div>

        {systemInfo && (
          <div className="p-6 space-y-4">
            {/* Basic Info */}
            <div className="grid grid-cols-2 gap-4">
              <div className="bg-matte-black/50 rounded-xl p-4">
                <div className="text-gray-400 text-sm mb-1">Operating System</div>
                <div className="text-white font-medium">{systemInfo.os} ({systemInfo.arch})</div>
              </div>
              <div className="bg-matte-black/50 rounded-xl p-4">
                <div className="text-gray-400 text-sm mb-1">Hostname</div>
                <div className="text-white font-medium">{systemInfo.hostname}</div>
              </div>
              <div className="bg-matte-black/50 rounded-xl p-4">
                <div className="text-gray-400 text-sm mb-1">CPU Cores</div>
                <div className="text-white font-medium">{systemInfo.cpu_count}</div>
              </div>
              <div className="bg-matte-black/50 rounded-xl p-4">
                <div className="text-gray-400 text-sm mb-1">Server Uptime</div>
                <div className="text-white font-medium">{systemInfo.server_uptime}</div>
              </div>
            </div>

            {/* Disk Usage */}
            {systemInfo.disk_total > 0 && (
              <div className="bg-matte-black/50 rounded-xl p-4">
                <div className="flex justify-between mb-2">
                  <span className="text-gray-400 text-sm">Disk Usage</span>
                  <span className="text-white text-sm">{formatBytes(systemInfo.disk_used)} / {formatBytes(systemInfo.disk_total)}</span>
                </div>
                <div className="w-full h-2 bg-matte-black rounded-full overflow-hidden">
                  <div
                    className="h-full bg-yellow-neon"
                    style={{ width: `${(systemInfo.disk_used / systemInfo.disk_total) * 100}%` }}
                  />
                </div>
                <div className="text-gray-500 text-xs mt-1">{formatBytes(systemInfo.disk_free)} free</div>
              </div>
            )}

            {/* Memory Usage */}
            {systemInfo.memory_used > 0 && (
              <div className="bg-matte-black/50 rounded-xl p-4">
                <div className="text-gray-400 text-sm mb-1">Go Runtime Memory</div>
                <div className="text-white font-medium">{formatBytes(systemInfo.memory_used)}</div>
              </div>
            )}

            {/* Network Addresses */}
            {systemInfo.network_addrs && systemInfo.network_addrs.length > 0 && (
              <div className="bg-matte-black/50 rounded-xl p-4">
                <div className="text-gray-400 text-sm mb-2">Network Addresses</div>
                <div className="space-y-1">
                  {systemInfo.network_addrs.map((addr, i) => (
                    <div key={i} className="text-white font-mono text-sm">{addr}</div>
                  ))}
                </div>
              </div>
            )}

            {/* Version Info */}
            <div className="flex justify-between text-sm">
              <span className="text-gray-400">Server Version</span>
              <span className="text-white font-mono">0.1.0</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-400">Go Version</span>
              <span className="text-white font-mono">{systemInfo.go_version}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-400">Server URL</span>
              <span className="text-white font-mono text-sm">{window.location.origin}</span>
            </div>
          </div>
        )}
      </div>

      {/* Background Music Settings */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Background Music</h2>
          <p className="text-sm text-gray-400">Play music when the queue is empty</p>
        </div>

        <div className="p-6 space-y-4">
          {/* Enable Toggle */}
          <div className="flex items-center justify-between">
            <div>
              <label className="text-white font-medium">Enable Background Music</label>
              <p className="text-sm text-gray-400">Automatically play music when queue ends</p>
            </div>
            <button
              onClick={() => setBgmSettings({ ...bgmSettings, enabled: !bgmSettings.enabled })}
              className={`relative w-12 h-6 rounded-full transition-colors ${
                bgmSettings.enabled ? 'bg-yellow-neon' : 'bg-gray-600'
              }`}
            >
              <span
                className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${
                  bgmSettings.enabled ? 'translate-x-6' : ''
                }`}
              />
            </button>
          </div>

          {/* Source Type Selection */}
          <div>
            <label className="block text-sm text-gray-400 mb-2">Source Type</label>
            <div className="flex gap-2">
              <button
                onClick={() => setBgmSettings({ ...bgmSettings, source_type: 'youtube', url: bgmSettings.source_type === 'youtube' ? bgmSettings.url : '' })}
                className={`flex-1 py-2 px-4 rounded-lg font-medium transition-colors ${
                  bgmSettings.source_type === 'youtube'
                    ? 'bg-red-600 text-white'
                    : 'bg-matte-black text-gray-400 hover:bg-gray-700'
                }`}
              >
                YouTube
              </button>
              <button
                onClick={() => setBgmSettings({ ...bgmSettings, source_type: 'icecast', url: bgmSettings.source_type === 'icecast' ? bgmSettings.url : '' })}
                className={`flex-1 py-2 px-4 rounded-lg font-medium transition-colors ${
                  bgmSettings.source_type === 'icecast'
                    ? 'bg-blue-600 text-white'
                    : 'bg-matte-black text-gray-400 hover:bg-gray-700'
                }`}
              >
                Icecast Stream
              </button>
            </div>
          </div>

          {/* YouTube URL Input */}
          {bgmSettings.source_type === 'youtube' && (
            <div>
              <label className="block text-sm text-gray-400 mb-1">YouTube URL or Playlist</label>
              <input
                type="text"
                value={bgmSettings.url}
                onChange={(e) => setBgmSettings({ ...bgmSettings, url: e.target.value })}
                placeholder="https://www.youtube.com/watch?v=..."
                className="w-full px-4 py-2 bg-matte-black rounded-lg border border-white/10 text-white placeholder-gray-500 focus:outline-none focus:border-yellow-neon"
              />
              <p className="text-xs text-gray-500 mt-1">Supports single videos, playlists, and live streams</p>
            </div>
          )}

          {/* Icecast Stream Input */}
          {bgmSettings.source_type === 'icecast' && (
            <div className="space-y-3">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Stream URL</label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={bgmSettings.url}
                    onChange={(e) => setBgmSettings({ ...bgmSettings, url: e.target.value })}
                    placeholder="https://stream.example.com/radio.mp3"
                    className="flex-1 px-4 py-2 bg-matte-black rounded-lg border border-white/10 text-white placeholder-gray-500 focus:outline-none focus:border-yellow-neon"
                  />
                  <button
                    onClick={fetchIcecastStreams}
                    disabled={streamsLoading}
                    className="px-4 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-500 transition-colors disabled:opacity-50 whitespace-nowrap"
                  >
                    {streamsLoading ? 'Loading...' : 'Find Streams'}
                  </button>
                </div>
                <p className="text-xs text-gray-500 mt-1">Enter a stream URL or browse popular music streams</p>
              </div>

              {/* Stream Picker Modal */}
              {showStreamPicker && icecastStreams.length > 0 && (
                <div className="bg-matte-black rounded-xl border border-white/10 max-h-80 overflow-y-auto">
                  <div className="sticky top-0 bg-matte-black px-4 py-2 border-b border-white/10 flex justify-between items-center">
                    <span className="text-sm font-medium text-white">Select a Stream</span>
                    <button
                      onClick={() => setShowStreamPicker(false)}
                      className="text-gray-400 hover:text-white"
                    >
                      Close
                    </button>
                  </div>
                  <div className="divide-y divide-white/5">
                    {icecastStreams.map((stream, index) => (
                      <button
                        key={index}
                        onClick={() => selectIcecastStream(stream)}
                        className="w-full px-4 py-3 text-left hover:bg-gray-800 transition-colors"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex-1 min-w-0">
                            <div className="text-white font-medium truncate">{stream.name}</div>
                            <div className="text-sm text-gray-400 truncate">{stream.description}</div>
                            <div className="flex gap-2 mt-1">
                              <span className="text-xs px-2 py-0.5 bg-gray-700 rounded text-gray-300">
                                {stream.genre}
                              </span>
                              <span className="text-xs px-2 py-0.5 bg-gray-700 rounded text-gray-300">
                                {stream.bitrate}kbps {stream.format}
                              </span>
                            </div>
                          </div>
                          <svg className="w-5 h-5 text-gray-400 flex-shrink-0 ml-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Volume Slider */}
          <div>
            <label className="block text-sm text-gray-400 mb-2">BGM Volume: {bgmSettings.volume}%</label>
            <input
              type="range"
              min="0"
              max="100"
              value={bgmSettings.volume}
              onChange={(e) => setBgmSettings({ ...bgmSettings, volume: parseInt(e.target.value) })}
              className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-yellow-neon"
            />
          </div>

          {/* Save Button */}
          <button
            onClick={saveBgmSettings}
            disabled={bgmSaving}
            className="w-full py-2 bg-yellow-neon text-matte-black font-semibold rounded-lg hover:bg-yellow-400 transition-colors disabled:opacity-50"
          >
            {bgmSaving ? 'Saving...' : 'Save BGM Settings'}
          </button>
        </div>
      </div>

      {/* Database Management */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Database Management</h2>
          <p className="text-sm text-gray-400">Clear data from the database (use with caution)</p>
        </div>

        <div className="p-6 space-y-4">
          {/* Stats Grid */}
          {dbStats && (
            <div className="grid grid-cols-5 gap-3 mb-6">
              <div className="bg-matte-black/50 rounded-xl p-3 text-center">
                <div className="text-xl font-bold text-white">{dbStats.sessions.count}</div>
                <div className="text-xs text-gray-400">Users</div>
              </div>
              <div className="bg-matte-black/50 rounded-xl p-3 text-center">
                <div className="text-xl font-bold text-red-400">{dbStats.blocked_users.count}</div>
                <div className="text-xs text-gray-400">Blocked</div>
              </div>
              <div className="bg-matte-black/50 rounded-xl p-3 text-center">
                <div className="text-xl font-bold text-white">{dbStats.queue.count}</div>
                <div className="text-xs text-gray-400">Queue</div>
              </div>
              <div className="bg-matte-black/50 rounded-xl p-3 text-center">
                <div className="text-xl font-bold text-white">{dbStats.search_logs.count}</div>
                <div className="text-xs text-gray-400">Searches</div>
              </div>
              <div className="bg-matte-black/50 rounded-xl p-3 text-center">
                <div className="text-xl font-bold text-white">{dbStats.song_history.count}</div>
                <div className="text-xs text-gray-400">History</div>
              </div>
            </div>
          )}

          {/* Flush Buttons */}
          <div className="space-y-3">
            <div className="flex items-center justify-between p-3 bg-matte-black/30 rounded-xl">
              <div>
                <div className="text-white font-medium">Flush Users</div>
                <div className="text-xs text-gray-400">Clear all user sessions (will disconnect everyone)</div>
              </div>
              <button
                onClick={() => flushData('flush_sessions', 'Clear all user sessions? This will disconnect all users.')}
                disabled={flushingAction !== null}
                className="px-4 py-2 bg-orange-500/20 text-orange-400 font-medium rounded-lg hover:bg-orange-500/30 transition-colors disabled:opacity-50"
              >
                {flushingAction === 'flush_sessions' ? 'Clearing...' : 'Flush'}
              </button>
            </div>

            <div className="flex items-center justify-between p-3 bg-matte-black/30 rounded-xl">
              <div>
                <div className="text-white font-medium">Flush Blocked Users</div>
                <div className="text-xs text-gray-400">Remove all blocks, allowing everyone to reconnect</div>
              </div>
              <button
                onClick={() => flushData('flush_blocked', 'Remove all user blocks?')}
                disabled={flushingAction !== null}
                className="px-4 py-2 bg-orange-500/20 text-orange-400 font-medium rounded-lg hover:bg-orange-500/30 transition-colors disabled:opacity-50"
              >
                {flushingAction === 'flush_blocked' ? 'Clearing...' : 'Flush'}
              </button>
            </div>

            <div className="flex items-center justify-between p-3 bg-matte-black/30 rounded-xl">
              <div>
                <div className="text-white font-medium">Flush Queue</div>
                <div className="text-xs text-gray-400">Clear all songs from the queue</div>
              </div>
              <button
                onClick={() => flushData('flush_queue', 'Clear the entire song queue?')}
                disabled={flushingAction !== null}
                className="px-4 py-2 bg-orange-500/20 text-orange-400 font-medium rounded-lg hover:bg-orange-500/30 transition-colors disabled:opacity-50"
              >
                {flushingAction === 'flush_queue' ? 'Clearing...' : 'Flush'}
              </button>
            </div>

            <div className="flex items-center justify-between p-3 bg-matte-black/30 rounded-xl">
              <div>
                <div className="text-white font-medium">Flush Search Logs</div>
                <div className="text-xs text-gray-400">Clear all search history and analytics</div>
              </div>
              <button
                onClick={() => flushData('flush_search_logs', 'Clear all search logs and analytics?')}
                disabled={flushingAction !== null}
                className="px-4 py-2 bg-orange-500/20 text-orange-400 font-medium rounded-lg hover:bg-orange-500/30 transition-colors disabled:opacity-50"
              >
                {flushingAction === 'flush_search_logs' ? 'Clearing...' : 'Flush'}
              </button>
            </div>

            <div className="flex items-center justify-between p-3 bg-matte-black/30 rounded-xl">
              <div>
                <div className="text-white font-medium">Flush Song History</div>
                <div className="text-xs text-gray-400">Clear all play counts and history</div>
              </div>
              <button
                onClick={() => flushData('flush_song_history', 'Clear all song play history and counts?')}
                disabled={flushingAction !== null}
                className="px-4 py-2 bg-orange-500/20 text-orange-400 font-medium rounded-lg hover:bg-orange-500/30 transition-colors disabled:opacity-50"
              >
                {flushingAction === 'flush_song_history' ? 'Clearing...' : 'Flush'}
              </button>
            </div>

            <div className="pt-4 border-t border-white/10">
              <div className="flex items-center justify-between p-3 bg-red-500/10 rounded-xl border border-red-500/30">
                <div>
                  <div className="text-red-400 font-medium">Flush All Data</div>
                  <div className="text-xs text-gray-400">Clear everything except library songs</div>
                </div>
                <button
                  onClick={() => flushData('flush_all', 'This will clear ALL data except your song library. Are you sure?')}
                  disabled={flushingAction !== null}
                  className="px-4 py-2 bg-red-500/20 text-red-400 font-medium rounded-lg hover:bg-red-500/30 transition-colors disabled:opacity-50"
                >
                  {flushingAction === 'flush_all' ? 'Clearing...' : 'Flush All'}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    {activeHelp && (
      <HelpModal topic={activeHelp} isOpen={true} onClose={closeHelp} />
    )}
    </>
  );
}

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

function QueueManagement() {
  // Use the shared roomStore state - same as the Queue component
  const queue = useRoomStore(selectQueue);
  const currentPosition = useRoomStore(selectQueuePosition);
  const autoplay = useRoomStore(selectAutoplay);
  const countdown = useRoomStore(selectCountdown);
  const clients = useAdminStore((state) => state.clients);
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  // URL param for tab persistence
  const [searchParams, setSearchParams] = useSearchParams();
  const [activeTab, setActiveTabState] = useState<'queue' | 'history'>('queue');

  // Sync tab state with URL on mount and when searchParams change
  useEffect(() => {
    const tabFromUrl = searchParams.get('qtab') as 'queue' | 'history' | null;
    if (tabFromUrl === 'history' || tabFromUrl === 'queue') {
      setActiveTabState(tabFromUrl);
    }
  }, [searchParams]);

  // Requeue modal state
  const [requeueModal, setRequeueModal] = useState<{ song: typeof queue[0] } | null>(null);
  const [requeSelectedUser, setRequeueSelectedUser] = useState<string>('');

  const setActiveTab = (tab: 'queue' | 'history') => {
    setActiveTabState(tab);
    setSearchParams((prev) => {
      const newParams = new URLSearchParams(prev);
      newParams.set('qtab', tab);
      return newParams;
    });
  };

  // Filter songs based on active tab
  const historySongs = queue.filter((_, index) => index < currentPosition);
  const queueSongs = queue.filter((_, index) => index >= currentPosition);

  const handleToggleAutoplay = () => {
    wsService.setAutoplay(!autoplay);
  };

  const handlePlay = () => {
    // Admin play always triggers adminPlayNext which starts a countdown
    wsService.adminPlayNext();
  };

  const handleStop = () => {
    // Only stop if something is playing (current position is within queue)
    if (currentPosition < queue.length) {
      wsService.adminStop();
    }
  };

  // Check if a song is currently playing
  const isPlaying = currentPosition < queue.length;

  const handleShuffle = () => {
    wsService.queueShuffle();
  };

  // Helper to get display name from MartynKey
  const getSingerName = (martynKey: string): string => {
    const client = clients.find((c) => c.martyn_key === martynKey);
    return client?.display_name || 'Unknown';
  };

  const handleRemove = (songId: string) => {
    wsService.queueRemove(songId);
  };

  const handleClear = () => {
    if (confirm('Clear the entire queue? This cannot be undone.')) {
      wsService.queueClear();
    }
  };

  const handleDragStart = (e: React.DragEvent, index: number) => {
    setDraggedIndex(index);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', index.toString());
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setDragOverIndex(index);
  };

  const handleDragLeave = () => {
    setDragOverIndex(null);
  };

  const handleDrop = (e: React.DragEvent, toIndex: number) => {
    e.preventDefault();
    const fromIndex = parseInt(e.dataTransfer.getData('text/plain'), 10);
    if (fromIndex !== toIndex) {
      wsService.queueMove(fromIndex, toIndex);
    }
    setDraggedIndex(null);
    setDragOverIndex(null);
  };

  const handleDragEnd = () => {
    setDraggedIndex(null);
    setDragOverIndex(null);
  };

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  return (
    <div className="space-y-6">
      {/* Queue Controls */}
      {/* Countdown Alert Banner */}
      {countdown.active && (
        <div className={`rounded-2xl overflow-hidden mb-6 ${
          countdown.requires_approval
            ? 'bg-orange-500/20 border border-orange-500/30'
            : 'bg-cyan-500/20 border border-cyan-500/30'
        }`}>
          <div className="px-6 py-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className={`w-14 h-14 rounded-full flex items-center justify-center text-2xl font-bold ${
                countdown.requires_approval ? 'bg-orange-500/30 text-orange-400' : 'bg-cyan-500/30 text-cyan-400'
              }`}>
                {countdown.seconds_remaining}
              </div>
              <div>
                <h3 className={`font-semibold ${countdown.requires_approval ? 'text-orange-400' : 'text-cyan-400'}`}>
                  {countdown.requires_approval ? 'Waiting for Admin Approval' : 'Next Song Starting Soon'}
                </h3>
                <p className="text-sm text-gray-400">
                  {countdown.requires_approval
                    ? 'Different singer - tap "Start Now" to begin'
                    : `Same singer - auto-starting in ${countdown.seconds_remaining}s`
                  }
                </p>
              </div>
            </div>
            <button
              onClick={handlePlay}
              className={`px-6 py-3 font-bold rounded-xl transition-colors ${
                countdown.requires_approval
                  ? 'bg-orange-500 text-white hover:bg-orange-600'
                  : 'bg-cyan-500 text-white hover:bg-cyan-600'
              }`}
            >
              Start Now
            </button>
          </div>
        </div>
      )}

      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        {/* Header with prominent Play/Stop buttons */}
        <div className="px-6 py-4 border-b border-white/5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-white">Song Queue</h2>
              <p className="text-sm text-gray-400">
                {queueSongs.length} upcoming • {historySongs.length} in history
              </p>
            </div>

            {/* Prominent Play/Stop Toggle Button */}
            <div className="flex items-center gap-3">
              {isPlaying ? (
                <button
                  onClick={handleStop}
                  className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-yellow-500 to-amber-400 text-gray-900 font-bold rounded-xl hover:from-yellow-400 hover:to-amber-300 transition-all shadow-lg shadow-yellow-500/30 hover:shadow-yellow-400/40 hover:scale-105 active:scale-95"
                >
                  <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M6 6h12v12H6z" />
                  </svg>
                  Stop
                </button>
              ) : (
                <button
                  onClick={handlePlay}
                  disabled={queueSongs.length === 0}
                  className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-cyan-500 to-cyan-400 text-gray-900 font-bold rounded-xl hover:from-cyan-400 hover:to-cyan-300 transition-all disabled:opacity-40 disabled:cursor-not-allowed shadow-lg shadow-cyan-500/30 hover:shadow-cyan-400/40 hover:scale-105 active:scale-95"
                >
                  <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M8 5v14l11-7z" />
                  </svg>
                  Play
                </button>
              )}
            </div>
          </div>

          {/* Proper Tab Bar */}
          <div className="flex border-b border-white/10 mb-4">
            <button
              onClick={() => setActiveTab('queue')}
              className={`px-6 py-3 font-medium transition-colors relative ${
                activeTab === 'queue'
                  ? 'text-yellow-neon'
                  : 'text-gray-400 hover:text-white'
              }`}
            >
              Up Next ({queueSongs.length})
              {activeTab === 'queue' && (
                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-yellow-neon" />
              )}
            </button>
            <button
              onClick={() => setActiveTab('history')}
              className={`px-6 py-3 font-medium transition-colors relative ${
                activeTab === 'history'
                  ? 'text-purple-400'
                  : 'text-gray-400 hover:text-white'
              }`}
            >
              History ({historySongs.length})
              {activeTab === 'history' && (
                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-purple-400" />
              )}
            </button>
          </div>

          {/* Secondary Controls */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              {/* Autoplay Toggle Switch */}
              <div className="flex items-center gap-3">
                <span className="text-sm text-gray-400">Autoplay</span>
                <button
                  onClick={handleToggleAutoplay}
                  className={`relative w-12 h-6 rounded-full transition-colors ${
                    autoplay ? 'bg-yellow-neon' : 'bg-gray-600'
                  }`}
                >
                  <div
                    className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform shadow ${
                      autoplay ? 'translate-x-7' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>

              {/* Shuffle Button */}
              <button
                onClick={handleShuffle}
                disabled={queue.length - currentPosition <= 2}
                className="flex items-center gap-2 px-4 py-2 bg-purple-500/20 text-purple-400 font-medium rounded-lg hover:bg-purple-500/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.5 12c0-1.232-.046-2.453-.138-3.662a4.006 4.006 0 00-3.7-3.7 48.678 48.678 0 00-7.324 0 4.006 4.006 0 00-3.7 3.7c-.017.22-.032.441-.046.662M19.5 12l3-3m-3 3l-3-3m-12 3c0 1.232.046 2.453.138 3.662a4.006 4.006 0 003.7 3.7 48.656 48.656 0 007.324 0 4.006 4.006 0 003.7-3.7c.017-.22.032-.441.046-.662M4.5 12l3 3m-3-3l-3 3" />
                </svg>
                Shuffle
              </button>
            </div>

            {/* Clear Queue */}
            <button
              onClick={handleClear}
              disabled={queue.length === 0}
              className="px-4 py-2 bg-red-500/20 text-red-400 font-medium rounded-lg hover:bg-red-500/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Clear Queue
            </button>
          </div>
        </div>

        {/* Queue List */}
        <div className="divide-y divide-white/5">
          {activeTab === 'queue' && queueSongs.length === 0 ? (
            <div className="px-6 py-12 text-center">
              <svg className="w-12 h-12 mx-auto text-gray-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
              </svg>
              <p className="text-gray-500">No songs in the queue</p>
              <p className="text-gray-600 text-sm mt-1">Add songs from the search page</p>
            </div>
          ) : activeTab === 'history' && historySongs.length === 0 ? (
            <div className="px-6 py-12 text-center">
              <svg className="w-12 h-12 mx-auto text-gray-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="text-gray-500">No song history yet</p>
              <p className="text-gray-600 text-sm mt-1">Previously played songs will appear here</p>
            </div>
          ) : (
            (activeTab === 'queue' ? queueSongs : historySongs.slice().reverse()).map((song) => {
              const index = queue.findIndex(s => s.id === song.id);
              return (
              <div
                key={song.id}
                draggable
                onDragStart={(e) => handleDragStart(e, index)}
                onDragOver={(e) => handleDragOver(e, index)}
                onDragLeave={handleDragLeave}
                onDrop={(e) => handleDrop(e, index)}
                onDragEnd={handleDragEnd}
                className={`flex items-center gap-4 px-6 py-4 transition-all cursor-grab active:cursor-grabbing ${
                  index === currentPosition ? 'bg-yellow-neon/10' : ''
                } ${
                  draggedIndex === index ? 'opacity-50' : ''
                } ${
                  dragOverIndex === index ? 'border-t-2 border-yellow-neon' : ''
                }`}
              >
                {/* Position & Drag Handle */}
                <div className="flex items-center gap-2 w-12 shrink-0">
                  <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
                  </svg>
                  <span className={`text-sm font-medium ${
                    index === currentPosition ? 'text-yellow-neon' : 'text-gray-500'
                  }`}>
                    {index + 1}
                  </span>
                </div>

                {/* Thumbnail */}
                <div className="w-12 h-12 rounded-lg overflow-hidden bg-matte-black shrink-0">
                  {song.thumbnail_url ? (
                    <img src={song.thumbnail_url} alt="" className="w-full h-full object-cover" />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-gray-600">
                      <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
                      </svg>
                    </div>
                  )}
                </div>

                {/* Song Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className={`font-medium truncate ${
                      index === currentPosition ? 'text-yellow-neon' : 'text-white'
                    }`}>
                      {song.title}
                    </h3>
                    {index === currentPosition && (
                      <span className="px-2 py-0.5 bg-yellow-neon text-indigo-deep text-xs font-bold rounded shrink-0">
                        NOW PLAYING
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-gray-400 truncate">{song.artist}</p>
                  <p className="text-xs text-gray-500 truncate">
                    <span className="text-purple-400">{getSingerName(song.added_by)}</span>
                  </p>
                </div>

                {/* Duration */}
                <div className="text-gray-400 text-sm shrink-0">
                  {formatDuration(song.duration)}
                </div>

                {/* Vocal Assist Level */}
                <div className={`px-2 py-1 rounded text-xs font-medium shrink-0 ${
                  song.vocal_assist === 'OFF' ? 'bg-gray-700 text-gray-400' :
                  song.vocal_assist === 'LOW' ? 'bg-blue-500/20 text-blue-400' :
                  song.vocal_assist === 'MED' ? 'bg-purple-500/20 text-purple-400' :
                  'bg-green-500/20 text-green-400'
                }`}>
                  {song.vocal_assist}
                </div>

                {/* Action Buttons */}
                {activeTab === 'history' ? (
                  <button
                    onClick={() => {
                      setRequeueModal({ song });
                      setRequeueSelectedUser(song.added_by || '');
                    }}
                    className="p-2 text-gray-500 hover:text-purple-400 hover:bg-purple-500/10 rounded-lg transition-colors shrink-0"
                    title="Re-add to queue"
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                  </button>
                ) : (
                  <button
                    onClick={() => handleRemove(song.id)}
                    className="p-2 text-gray-500 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors shrink-0"
                    title="Remove from queue"
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                )}
              </div>
              );
            })
          )}
        </div>
      </div>

      {/* Requeue Modal */}
      {requeueModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
          <div className="bg-matte-gray rounded-2xl w-full max-w-md overflow-hidden">
            {/* Header */}
            <div className="px-6 py-4 border-b border-white/10">
              <h2 className="text-lg font-bold text-white">Re-add to Queue</h2>
              <p className="text-sm text-gray-400 mt-1">Add this song back to the queue</p>
            </div>

            {/* Song Preview */}
            <div className="px-6 py-4 border-b border-white/10">
              <div className="flex items-center gap-4">
                {requeueModal.song.thumbnail_url ? (
                  <img
                    src={requeueModal.song.thumbnail_url}
                    alt=""
                    className="w-16 h-16 rounded-lg object-cover"
                  />
                ) : (
                  <div className="w-16 h-16 rounded-lg bg-matte-black flex items-center justify-center text-gray-600">
                    <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
                    </svg>
                  </div>
                )}
                <div className="flex-1 min-w-0">
                  <h3 className="font-medium text-white truncate">{requeueModal.song.title}</h3>
                  <p className="text-sm text-gray-400 truncate">{requeueModal.song.artist}</p>
                </div>
              </div>
            </div>

            {/* Singer Selection */}
            <div className="px-6 py-4">
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Assign to singer
              </label>
              <select
                value={requeSelectedUser}
                onChange={(e) => setRequeueSelectedUser(e.target.value)}
                className="w-full px-4 py-3 bg-matte-black rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-yellow-neon"
              >
                <option value="">-- Select a singer --</option>
                {clients.map((client) => (
                  <option key={client.martyn_key} value={client.martyn_key}>
                    {client.display_name || client.martyn_key.slice(0, 8)}
                    {client.martyn_key === requeueModal.song.added_by && ' (original)'}
                  </option>
                ))}
              </select>
              <p className="text-xs text-gray-500 mt-2">
                The selected singer will be credited for this song
              </p>
            </div>

            {/* Actions */}
            <div className="px-6 py-4 border-t border-white/10 flex items-center justify-end gap-3">
              <button
                onClick={() => setRequeueModal(null)}
                className="px-4 py-2 text-gray-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  if (requeSelectedUser) {
                    wsService.queueRequeue(requeueModal.song.id, requeSelectedUser);
                    setRequeueModal(null);
                  }
                }}
                disabled={!requeSelectedUser}
                className="px-6 py-2 bg-gradient-to-r from-purple-500 to-pink-500 text-white font-bold rounded-xl hover:from-purple-400 hover:to-pink-400 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-purple-500/30"
              >
                Add to Queue
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Queue Help */}
      <div className="bg-matte-gray/50 rounded-xl p-4">
        <h3 className="text-white font-medium mb-2">Queue Management Tips</h3>
        <ul className="text-sm text-gray-400 space-y-1">
          <li>• Drag and drop songs to reorder them</li>
          <li>• The currently playing song is highlighted in yellow</li>
          <li>• Vocal assist level shows each singer's preference</li>
          <li>• Changes sync in real-time to all connected clients</li>
        </ul>
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
  const [searchParams, setSearchParams] = useSearchParams();
  const [activeTab, setActiveTabState] = useState<AdminTab>('clients');

  // Sync tab state with URL on mount and when searchParams change
  useEffect(() => {
    const tabFromUrl = searchParams.get('tab') as AdminTab | null;
    const validTabs: AdminTab[] = ['clients', 'queue', 'library', 'search-logs', 'network', 'settings'];
    if (tabFromUrl && validTabs.includes(tabFromUrl)) {
      setActiveTabState(tabFromUrl);
    }
  }, [searchParams]);

  const setActiveTab = (tab: AdminTab) => {
    setActiveTabState(tab);
    setSearchParams((prev) => {
      const newParams = new URLSearchParams(prev);
      newParams.set('tab', tab);
      return newParams;
    });
  };

  // Initialize WebSocket connection and roomStore updates
  useWebSocket();

  useEffect(() => {
    checkAuth().then(() => setIsLoading(false));
  }, [checkAuth]);

  // Get connection state from roomStore
  const isConnected = useRoomStore((state) => state.isConnected);

  // Fetch client list when connected
  useEffect(() => {
    if (!isAuthenticated || !isConnected) return;
    fetchClients();
  }, [isAuthenticated, isConnected, fetchClients]);

  // Subscribe to real-time client list updates
  useEffect(() => {
    if (!isAuthenticated) return;

    const unsubClientList = wsService.on('client_list', (clients: ClientInfo[]) => {
      setClients(clients);
    });

    return () => {
      unsubClientList();
    };
  }, [isAuthenticated, setClients]);

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
    queue: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
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
        <Link to="/" className="flex items-center gap-3 hover:opacity-80 transition-opacity">
          <img src="/logo.jpeg" alt="SongMartyn" className="w-10 h-10 rounded-lg object-cover" />
          <div>
            <h1 className="text-xl font-bold text-white">
              Song<span className="text-yellow-neon">Martyn</span> Admin
            </h1>
            {isLocal && (
              <span className="text-xs text-green-400">Local Access</span>
            )}
          </div>
        </Link>

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
            label="Queue"
            icon={icons.queue}
            active={activeTab === 'queue'}
            onClick={() => setActiveTab('queue')}
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
        {activeTab === 'queue' && <QueueManagement />}
        {activeTab === 'library' && <LibraryManagement />}
        {activeTab === 'search-logs' && <SearchLogs />}
        {activeTab === 'network' && <NetworkSettings />}
        {activeTab === 'settings' && <GeneralSettings />}
      </main>
    </div>
  );
}
