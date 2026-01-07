import { useEffect, useState, useRef } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useAdminStore } from '../stores/adminStore';
import { useLibraryStore } from '../stores/libraryStore';
import { useRoomStore, selectQueue, selectQueuePosition, selectAutoplay, selectCountdown, selectIdle, selectBgmActive, selectBgmEnabled } from '../stores/roomStore';
import { useWebSocket } from '../hooks/useWebSocket';
import { wsService } from '../services/websocket';
import type { ClientInfo, LibraryLocation, AvatarConfig, BGMSourceType, IcecastStream } from '../types';
import { HelpModal, HelpButton, useHelpModal } from '../components/HelpModal';
import { MPVSetupModal } from '../components/MPVSetupModal';
import { buildAvatarUrl } from '../components/AvatarCreator';
import { Footer } from '../components/Footer';

type AdminTab = 'clients' | 'queue' | 'library' | 'search-logs' | 'network' | 'diagnostics' | 'settings';

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

function EditNameModal({ client, onClose }: {
  client: ClientInfo;
  onClose: () => void;
}) {
  const { setClientName, setClientNameLock } = useAdminStore();
  const [newName, setNewName] = useState(client.display_name);
  const [lockName, setLockName] = useState(client.name_locked);
  const [isProcessing, setIsProcessing] = useState(false);

  const handleSubmit = async () => {
    setIsProcessing(true);

    // Update name if changed
    if (newName !== client.display_name) {
      await setClientName(client.martyn_key, newName);
    }

    // Update lock status if changed
    if (lockName !== client.name_locked) {
      await setClientNameLock(client.martyn_key, lockName);
    }

    setIsProcessing(false);
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80" onClick={onClose}>
      <div className="bg-matte-gray rounded-2xl p-6 w-full max-w-md" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-white">Edit User Name</h3>
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

        {/* Name Input */}
        <div className="mb-4">
          <label className="block text-sm text-gray-400 mb-2">Display Name</label>
          <input
            type="text"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Enter new name"
            className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
          />
        </div>

        {/* Lock Toggle */}
        <div className="mb-6">
          <label className="flex items-center gap-3 cursor-pointer">
            <div
              className={`relative w-12 h-6 rounded-full transition-colors ${lockName ? 'bg-yellow-neon' : 'bg-matte-black'}`}
              onClick={() => setLockName(!lockName)}
            >
              <div className={`absolute w-5 h-5 bg-white rounded-full top-0.5 transition-transform ${lockName ? 'translate-x-6' : 'translate-x-0.5'}`} />
            </div>
            <div>
              <span className="text-white font-medium">Lock Name</span>
              <p className="text-sm text-gray-400">
                {lockName ? 'User cannot change their name' : 'User can change their own name'}
              </p>
            </div>
          </label>
        </div>

        {/* Submit Button */}
        <button
          onClick={handleSubmit}
          disabled={isProcessing || !newName.trim()}
          className="w-full py-3 font-semibold rounded-xl transition-colors disabled:opacity-50 bg-yellow-neon text-indigo-deep hover:bg-yellow-400"
        >
          {isProcessing ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </div>
  );
}

function AdminConfirmModal({ client, onConfirm, onClose }: {
  client: ClientInfo;
  onConfirm: () => void;
  onClose: () => void;
}) {
  const { authenticate } = useAdminStore();
  const [pin, setPin] = useState('');
  const [error, setError] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsProcessing(true);
    setError('');

    // Validate PIN using the existing authenticate endpoint
    const success = await authenticate(pin);
    if (success) {
      onConfirm();
      onClose();
    } else {
      setError('Invalid PIN');
    }
    setIsProcessing(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80" onClick={onClose}>
      <div className="bg-matte-gray rounded-2xl p-6 w-full max-w-md" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-white">Confirm Admin Promotion</h3>
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
              <span className="text-white font-medium truncate block">{client.display_name}</span>
              <div className="text-gray-400 text-sm truncate">{client.device_name}</div>
            </div>
          </div>
        </div>

        <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-xl p-3 mb-4">
          <p className="text-yellow-400 text-sm">
            <strong>Warning:</strong> Making this user an admin will give them full control over the karaoke system, including managing other users and settings.
          </p>
        </div>

        <form onSubmit={handleSubmit}>
          <label className="block text-sm text-gray-400 mb-2">Enter Admin PIN to confirm</label>
          <input
            type="password"
            value={pin}
            onChange={(e) => setPin(e.target.value)}
            placeholder="Enter PIN"
            className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon mb-4"
            autoFocus
          />

          {error && (
            <p className="text-red-400 text-sm mb-4">{error}</p>
          )}

          <div className="flex gap-3">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-3 font-semibold rounded-xl transition-colors bg-matte-black text-gray-400 hover:text-white"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isProcessing || !pin}
              className="flex-1 py-3 font-semibold rounded-xl transition-colors disabled:opacity-50 bg-yellow-neon text-indigo-deep hover:bg-yellow-400"
            >
              {isProcessing ? 'Confirming...' : 'Make Admin'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// Get device icon based on device type and OS
function getDeviceIcon(client: ClientInfo): React.ReactNode {
  const { device_type, device_os } = client;

  // Mobile devices
  if (device_type === 'mobile' || device_type === 'tablet') {
    return (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
      </svg>
    );
  }

  // Desktop by OS
  if (device_os === 'macOS') {
    return (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
      </svg>
    );
  }

  if (device_os === 'Windows') {
    return (
      <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M3 12V6.5l8-1.1v6.6H3zm0 .5h8v6.6l-8-1.1V12.5zm8.5-7.7l8.5-1.3v8.5h-8.5V4.8zm0 14.4v-7.7h8.5v9l-8.5-1.3z"/>
      </svg>
    );
  }

  if (device_os === 'Linux') {
    return (
      <svg className="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12.5 2c-1.5 0-2.5 1-2.5 2.5V6c0 .5.2 1 .5 1.5L9 9.5c-.5.5-.5 1-.5 1.5v2c0 1 .5 2 1.5 2.5V18c0 1 1 2 2 2h1c1 0 2-1 2-2v-2.5c1-.5 1.5-1.5 1.5-2.5v-2c0-.5 0-1-.5-1.5l-1.5-2c.3-.5.5-1 .5-1.5V4.5c0-1.5-1-2.5-2.5-2.5z"/>
      </svg>
    );
  }

  // Default desktop icon
  return (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
    </svg>
  );
}

// Client card for mobile-friendly display
function ClientCard({ client, onClick }: {
  client: ClientInfo;
  onClick: () => void;
}) {
  // Get last 4 chars of martyn_key as short ID
  const shortId = client.martyn_key.slice(-4).toUpperCase();

  return (
    <button
      onClick={onClick}
      className={`w-full p-3 rounded-xl text-left transition-colors ${
        client.is_blocked
          ? 'bg-red-500/10 border border-red-500/30'
          : 'bg-matte-black hover:bg-white/5'
      }`}
    >
      <div className="flex items-center gap-3">
        <div className="relative">
          <ClientAvatar config={client.avatar_config} size={40} />
          {/* Status indicator */}
          <div className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-matte-black ${
            client.is_blocked ? 'bg-red-500' :
            client.is_online ? 'bg-green-400' : 'bg-gray-500'
          }`} />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-white font-medium truncate">{client.display_name}</span>
            {client.name_locked && (
              <svg className="w-3 h-3 text-yellow-neon flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clipRule="evenodd" />
              </svg>
            )}
            {client.is_admin && (
              <span className="px-1.5 py-0.5 bg-yellow-neon/20 text-yellow-neon text-xs rounded flex-shrink-0">
                Admin
              </span>
            )}
            {client.is_afk && !client.is_blocked && (
              <span className="px-1.5 py-0.5 bg-orange-500/20 text-orange-400 text-xs rounded flex-shrink-0">
                AFK
              </span>
            )}
            {client.is_blocked && (
              <span className="px-1.5 py-0.5 bg-red-500/20 text-red-400 text-xs rounded flex-shrink-0">
                Blocked
              </span>
            )}
          </div>
          <div className="flex items-center gap-2 text-xs text-gray-500">
            <span className="font-mono">{shortId}</span>
            <span className="text-gray-600">•</span>
            <span className={`flex items-center gap-1 ${
              client.device_os === 'iOS' || client.device_os === 'Android' ? 'text-blue-400' :
              client.device_os === 'macOS' ? 'text-gray-300' :
              client.device_os === 'Windows' ? 'text-cyan-400' :
              client.device_os === 'Linux' ? 'text-orange-400' : 'text-gray-400'
            }`}>
              {getDeviceIcon(client)}
              {client.device_os || client.device_name || 'Unknown'}
            </span>
          </div>
        </div>

        <svg className="w-5 h-5 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
      </div>
    </button>
  );
}

// Client details modal with all actions
function ClientDetailsModal({ client, onClose }: {
  client: ClientInfo;
  onClose: () => void;
}) {
  const { setAdminStatus, setAFKStatus, unblockClient, setClientNameLock } = useAdminStore();
  const [showKickBlock, setShowKickBlock] = useState(false);
  const [showEditName, setShowEditName] = useState(false);
  const [showAdminConfirm, setShowAdminConfirm] = useState(false);

  const shortId = client.martyn_key.slice(-4).toUpperCase();

  const handleToggleAdmin = async () => {
    if (client.is_admin) {
      await setAdminStatus(client.martyn_key, false);
    } else {
      setShowAdminConfirm(true);
    }
  };

  const handleToggleAFK = async () => {
    await setAFKStatus(client.martyn_key, !client.is_afk);
  };

  const handleUnblock = async () => {
    if (confirm(`Unblock ${client.display_name}?`)) {
      await unblockClient(client.martyn_key);
      onClose();
    }
  };

  const handleToggleNameLock = async () => {
    await setClientNameLock(client.martyn_key, !client.name_locked);
  };

  if (showKickBlock) {
    return <KickBlockModal client={client} onClose={() => { setShowKickBlock(false); onClose(); }} />;
  }

  if (showEditName) {
    return <EditNameModal client={client} onClose={() => { setShowEditName(false); onClose(); }} />;
  }

  if (showAdminConfirm) {
    return (
      <AdminConfirmModal
        client={client}
        onConfirm={async () => {
          await setAdminStatus(client.martyn_key, true);
          setShowAdminConfirm(false);
        }}
        onClose={() => setShowAdminConfirm(false)}
      />
    );
  }

  return (
    <div className="fixed inset-0 bg-black/70 flex items-end sm:items-center justify-center z-50 p-4">
      <div className="bg-matte-gray rounded-2xl w-full max-w-md max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="p-4 border-b border-white/10 flex items-center gap-3">
          <div className="relative">
            <ClientAvatar config={client.avatar_config} size={48} />
            <div className={`absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full border-2 border-matte-gray ${
              client.is_blocked ? 'bg-red-500' :
              client.is_online ? 'bg-green-400' : 'bg-gray-500'
            }`} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="text-white font-semibold truncate">{client.display_name}</h3>
              {client.name_locked && (
                <svg className="w-4 h-4 text-yellow-neon flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clipRule="evenodd" />
                </svg>
              )}
            </div>
            <div className="text-sm text-gray-400">
              {client.is_blocked ? 'Blocked' : client.is_online ? 'Online' : 'Offline'}
              {client.is_afk && !client.is_blocked && ' • AFK'}
              {client.is_admin && ' • Admin'}
            </div>
          </div>
          <button onClick={onClose} className="p-2 text-gray-400 hover:text-white">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Info Section */}
        <div className="p-4 space-y-3 border-b border-white/10">
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div>
              <div className="text-gray-500 text-xs uppercase mb-1">Session ID</div>
              <div className="text-white font-mono">{shortId}</div>
            </div>
            <div>
              <div className="text-gray-500 text-xs uppercase mb-1">IP Address</div>
              <div className="text-white font-mono text-sm">{client.ip_address || 'Unknown'}</div>
            </div>
            <div>
              <div className="text-gray-500 text-xs uppercase mb-1">Device</div>
              <div className="text-white flex items-center gap-1.5">
                {getDeviceIcon(client)}
                <span>{client.device_name || 'Unknown'}</span>
              </div>
            </div>
            <div>
              <div className="text-gray-500 text-xs uppercase mb-1">Platform</div>
              <div className="text-white">
                {client.device_os || 'Unknown'}
                {client.device_browser && ` • ${client.device_browser}`}
              </div>
            </div>
          </div>
          {client.is_blocked && client.block_reason && (
            <div className="p-3 bg-red-500/10 rounded-lg border border-red-500/30">
              <div className="text-gray-500 text-xs uppercase mb-1">Block Reason</div>
              <div className="text-red-400">{client.block_reason}</div>
            </div>
          )}
        </div>

        {/* Actions Section */}
        <div className="p-4 space-y-2">
          {client.is_blocked ? (
            <button
              onClick={handleUnblock}
              className="w-full py-3 px-4 bg-green-500/20 text-green-400 rounded-xl font-medium hover:bg-green-500/30 transition-colors flex items-center justify-center gap-2"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              Unblock User
            </button>
          ) : (
            <>
              {/* Edit Name */}
              <button
                onClick={() => setShowEditName(true)}
                className="w-full py-3 px-4 bg-matte-black text-white rounded-xl font-medium hover:bg-white/10 transition-colors flex items-center justify-between"
              >
                <span className="flex items-center gap-2">
                  <svg className="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                  </svg>
                  Edit Name
                </span>
                <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </button>

              {/* Toggle Name Lock */}
              <button
                onClick={handleToggleNameLock}
                className="w-full py-3 px-4 bg-matte-black text-white rounded-xl font-medium hover:bg-white/10 transition-colors flex items-center justify-between"
              >
                <span className="flex items-center gap-2">
                  <svg className={`w-5 h-5 ${client.name_locked ? 'text-yellow-neon' : 'text-gray-400'}`} fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clipRule="evenodd" />
                  </svg>
                  {client.name_locked ? 'Unlock Name' : 'Lock Name'}
                </span>
                <span className={`text-sm ${client.name_locked ? 'text-yellow-neon' : 'text-gray-500'}`}>
                  {client.name_locked ? 'Locked' : 'Unlocked'}
                </span>
              </button>

              {/* Toggle Role */}
              <button
                onClick={handleToggleAdmin}
                className="w-full py-3 px-4 bg-matte-black text-white rounded-xl font-medium hover:bg-white/10 transition-colors flex items-center justify-between"
              >
                <span className="flex items-center gap-2">
                  <svg className={`w-5 h-5 ${client.is_admin ? 'text-yellow-neon' : 'text-gray-400'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
                  </svg>
                  {client.is_admin ? 'Remove Admin' : 'Make Admin'}
                </span>
                <span className={`text-sm ${client.is_admin ? 'text-yellow-neon' : 'text-gray-500'}`}>
                  {client.is_admin ? 'Admin' : 'User'}
                </span>
              </button>

              {/* Toggle AFK (only if online) */}
              {client.is_online && (
                <button
                  onClick={handleToggleAFK}
                  className="w-full py-3 px-4 bg-matte-black text-white rounded-xl font-medium hover:bg-white/10 transition-colors flex items-center justify-between"
                >
                  <span className="flex items-center gap-2">
                    <svg className={`w-5 h-5 ${client.is_afk ? 'text-orange-400' : 'text-gray-400'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    {client.is_afk ? 'Mark Present' : 'Mark AFK'}
                  </span>
                  <span className={`text-sm ${client.is_afk ? 'text-orange-400' : 'text-gray-500'}`}>
                    {client.is_afk ? 'AFK' : 'Present'}
                  </span>
                </button>
              )}

              {/* Kick/Block */}
              <button
                onClick={() => setShowKickBlock(true)}
                className="w-full py-3 px-4 bg-red-500/20 text-red-400 rounded-xl font-medium hover:bg-red-500/30 transition-colors flex items-center justify-center gap-2"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                </svg>
                Kick / Block
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function ClientList() {
  const { clients } = useAdminStore();
  const [selectedClient, setSelectedClient] = useState<ClientInfo | null>(null);

  const onlineClients = clients.filter((c) => c.is_online && !c.is_blocked);
  const offlineClients = clients.filter((c) => !c.is_online && !c.is_blocked);
  const blockedClients = clients.filter((c) => c.is_blocked);

  return (
    <>
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-4 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Connected Clients</h2>
          <p className="text-sm text-gray-400">
            {onlineClients.length} online, {offlineClients.length} offline
            {blockedClients.length > 0 && `, ${blockedClients.length} blocked`}
          </p>
        </div>

        <div className="p-4 space-y-4">
          {clients.length === 0 ? (
            <div className="py-8 text-center text-gray-500">
              No clients connected
            </div>
          ) : (
            <>
              {/* Online clients */}
              {onlineClients.length > 0 && (
                <div className="space-y-2">
                  <div className="text-xs uppercase text-gray-500 font-medium px-1">
                    Online ({onlineClients.length})
                  </div>
                  {onlineClients.map((client) => (
                    <ClientCard
                      key={client.martyn_key}
                      client={client}
                      onClick={() => setSelectedClient(client)}
                    />
                  ))}
                </div>
              )}

              {/* Offline clients */}
              {offlineClients.length > 0 && (
                <div className="space-y-2">
                  <div className="text-xs uppercase text-gray-500 font-medium px-1">
                    Offline ({offlineClients.length})
                  </div>
                  {offlineClients.map((client) => (
                    <ClientCard
                      key={client.martyn_key}
                      client={client}
                      onClick={() => setSelectedClient(client)}
                    />
                  ))}
                </div>
              )}

              {/* Blocked clients */}
              {blockedClients.length > 0 && (
                <div className="space-y-2">
                  <div className="text-xs uppercase text-red-400 font-medium px-1">
                    Blocked ({blockedClients.length})
                  </div>
                  {blockedClients.map((client) => (
                    <ClientCard
                      key={client.martyn_key}
                      client={client}
                      onClick={() => setSelectedClient(client)}
                    />
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {selectedClient && (
        <ClientDetailsModal
          client={selectedClient}
          onClose={() => setSelectedClient(null)}
        />
      )}
    </>
  );
}

const API_BASE = import.meta.env.DEV ? 'https://localhost:8443' : '';

interface DirEntry {
  name: string;
  path: string;
  is_dir: boolean;
}

interface BrowseDirsResponse {
  current: string;
  parent: string;
  dirs: DirEntry[];
}

function DirectoryPicker({
  isOpen,
  onClose,
  onSelect,
  token
}: {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (path: string) => void;
  token: string | null;
}) {
  const [currentPath, setCurrentPath] = useState('');
  const [parentPath, setParentPath] = useState('');
  const [directories, setDirectories] = useState<DirEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const fetchDirectories = async (path?: string) => {
    setLoading(true);
    setError('');
    try {
      const params = path ? `?path=${encodeURIComponent(path)}` : '';
      const headers: HeadersInit = {};
      if (token) {
        (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
      }
      const res = await fetch(`${API_BASE}/api/admin/browse-dirs${params}`, {
        headers,
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const data: BrowseDirsResponse = await res.json();
      setCurrentPath(data.current);
      setParentPath(data.parent);
      setDirectories(data.dirs);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to browse');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      fetchDirectories();
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
      <div className="bg-matte-gray rounded-2xl w-full max-w-lg max-h-[80vh] flex flex-col">
        <div className="px-6 py-4 border-b border-white/10 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-white">Select Folder</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="px-6 py-3 bg-matte-black/50 border-b border-white/5">
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-500">Current:</span>
            <code className="text-sm text-yellow-neon truncate flex-1">{currentPath}</code>
          </div>
        </div>

        {error && (
          <div className="px-6 py-2 bg-red-500/20 text-red-400 text-sm">
            {error}
          </div>
        )}

        <div className="flex-1 overflow-y-auto">
          {loading ? (
            <div className="px-6 py-8 text-center text-gray-500">Loading...</div>
          ) : (
            <div className="divide-y divide-white/5">
              {currentPath !== parentPath && (
                <button
                  onClick={() => fetchDirectories(parentPath)}
                  className="w-full px-6 py-3 flex items-center gap-3 hover:bg-white/5 transition-colors text-left"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                  </svg>
                  <span className="text-gray-300">..</span>
                  <span className="text-xs text-gray-500">(parent)</span>
                </button>
              )}
              {directories.map((dir) => (
                <button
                  key={dir.path}
                  onClick={() => fetchDirectories(dir.path)}
                  className="w-full px-6 py-3 flex items-center gap-3 hover:bg-white/5 transition-colors text-left"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-yellow-neon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                  </svg>
                  <span className="text-white">{dir.name}</span>
                </button>
              ))}
              {directories.length === 0 && !loading && (
                <div className="px-6 py-8 text-center text-gray-500">
                  No subdirectories found
                </div>
              )}
            </div>
          )}
        </div>

        <div className="px-6 py-4 border-t border-white/10 flex gap-3">
          <button
            onClick={onClose}
            className="flex-1 px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-500 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => {
              onSelect(currentPath);
              onClose();
            }}
            className="flex-1 px-4 py-2 bg-yellow-neon text-indigo-deep font-semibold rounded-lg hover:scale-[1.02] transition-transform"
          >
            Select This Folder
          </button>
        </div>
      </div>
    </div>
  );
}

function LibraryManagement() {
  const token = useAdminStore((state) => state.token);
  const { locations, stats, isLoading, error, fetchLocations, fetchStats, addLocation, removeLocation, scanLocation } = useLibraryStore();
  const [showAddForm, setShowAddForm] = useState(false);
  const [newPath, setNewPath] = useState('');
  const [newName, setNewName] = useState('');
  const [scanningId, setScanningId] = useState<number | null>(null);
  const [showDirPicker, setShowDirPicker] = useState(false);
  const { activeHelp, openHelp, closeHelp } = useHelpModal();

  useEffect(() => {
    fetchLocations();
    fetchStats();
  }, [fetchLocations, fetchStats]);

  const handleAddLocation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPath || !newName) return;

    // Save values before clearing
    const pathToAdd = newPath;
    const nameToAdd = newName;

    const success = await addLocation(pathToAdd, nameToAdd);
    if (success) {
      setNewPath('');
      setNewName('');
      setShowAddForm(false);

      // Auto-scan the newly added location
      // Get the latest locations to find the new one
      await fetchLocations();
      const updatedLocations = useLibraryStore.getState().locations;
      const newLocation = updatedLocations.find(loc => loc.name === nameToAdd);
      if (newLocation) {
        setScanningId(newLocation.id);
        const count = await scanLocation(newLocation.id);
        setScanningId(null);
        if (count !== null) {
          alert(`Found ${count} songs in ${newLocation.name}`);
          fetchStats();
        }
      }
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
            <div className="flex gap-2">
              <input
                type="text"
                value={newPath}
                onChange={(e) => setNewPath(e.target.value)}
                placeholder="/path/to/music/folder"
                className="flex-1 px-4 py-2 bg-matte-black rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
              />
              <button
                type="button"
                onClick={() => setShowDirPicker(true)}
                className="px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg font-medium hover:bg-blue-500/30 transition-colors whitespace-nowrap"
              >
                Browse
              </button>
            </div>
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
    <DirectoryPicker
      isOpen={showDirPicker}
      onClose={() => setShowDirPicker(false)}
      onSelect={(path) => {
        setNewPath(path);
        // Auto-generate a name from the folder name
        const folderName = path.split('/').pop() || path;
        if (!newName) {
          setNewName(folderName);
        }
      }}
      token={token}
    />
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
  const token = useAdminStore((state) => state.token);
  const [networks, setNetworks] = useState<NetworkInterface[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedUrl, setSelectedUrl] = useState<string | null>(null);
  const [savedUrl, setSavedUrl] = useState<string | null>(null);
  const [showQRModal, setShowQRModal] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const { activeHelp, openHelp, closeHelp } = useHelpModal();

  // mDNS settings
  const [mdnsHostname, setMdnsHostname] = useState('');
  const [mdnsEnabled, setMdnsEnabled] = useState(false);
  const [mdnsUrl, setMdnsUrl] = useState('');
  const [mdnsSaving, setMdnsSaving] = useState(false);
  const [mdnsError, setMdnsError] = useState('');

  // Fetch networks, saved URL, and mDNS settings when this tab is active
  useEffect(() => {
    const headers: HeadersInit = {};
    if (token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
    }

    setIsLoading(true);
    Promise.all([
      fetch(`${API_BASE}/api/admin/networks`, { headers }).then(r => r.ok ? r.json() : []),
      fetch(`${API_BASE}/api/connect-url`, { headers }).then(r => r.ok ? r.json() : null),
      fetch(`${API_BASE}/api/admin/mdns`, { headers }).then(r => r.ok ? r.json() : null),
    ]).then(([networksData, connectData, mdnsData]) => {
      setNetworks(Array.isArray(networksData) ? networksData : []);
      const currentUrl = connectData?.url || null;
      setSavedUrl(currentUrl);
      setSelectedUrl(currentUrl);
      if (mdnsData) {
        setMdnsHostname(mdnsData.hostname || '');
        setMdnsEnabled(mdnsData.enabled || false);
        setMdnsUrl(mdnsData.url || '');
      }
    }).catch(err => console.error('Failed to load network data:', err))
      .finally(() => setIsLoading(false));
  }, [token]);

  const saveSelectedUrl = async (url: string) => {
    setIsSaving(true);
    try {
      const headers: HeadersInit = { 'Content-Type': 'application/json' };
      if (token) {
        (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
      }
      const res = await fetch(`${API_BASE}/api/connect-url`, {
        method: 'POST',
        headers,
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

  const saveMdnsHostname = async () => {
    setMdnsSaving(true);
    setMdnsError('');
    try {
      const headers: HeadersInit = { 'Content-Type': 'application/json' };
      if (token) {
        (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
      }
      const res = await fetch(`${API_BASE}/api/admin/mdns`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ hostname: mdnsHostname }),
      });
      const data = await res.json();
      if (res.ok && data.success) {
        setMdnsEnabled(!!mdnsHostname);
        setMdnsUrl(data.url || '');
        // If mDNS URL was just enabled, offer to set it as default
        if (mdnsHostname && data.url) {
          setMdnsUrl(data.url);
        }
      } else {
        setMdnsError(data.error || 'Failed to update mDNS settings');
      }
    } catch (err) {
      setMdnsError('Failed to save mDNS settings');
    } finally {
      setMdnsSaving(false);
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

      {/* Local Hostname (mDNS) */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">Local Hostname (mDNS)</h2>
          <p className="text-sm text-gray-400">Advertise a friendly .local address instead of IP</p>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-2">Hostname</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={mdnsHostname}
                onChange={(e) => setMdnsHostname(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
                placeholder="karaoke"
                maxLength={30}
                className="flex-1 px-4 py-2 bg-matte-black rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon font-mono"
              />
              <span className="px-3 py-2 bg-matte-black rounded-lg text-gray-400 font-mono">.local</span>
            </div>
            <p className="text-xs text-gray-500 mt-1">
              Letters, numbers, and hyphens only. Leave empty to disable.
            </p>
          </div>

          {mdnsError && (
            <p className="text-red-400 text-sm">{mdnsError}</p>
          )}

          <button
            onClick={saveMdnsHostname}
            disabled={mdnsSaving}
            className="w-full py-2 bg-yellow-neon text-matte-black font-semibold rounded-lg hover:bg-yellow-400 transition-colors disabled:opacity-50"
          >
            {mdnsSaving ? 'Saving...' : 'Save Hostname'}
          </button>

          {/* mDNS URL option */}
          {mdnsEnabled && mdnsUrl && (
            <div className="mt-4 p-4 bg-matte-black/50 rounded-xl">
              <div className="flex items-center justify-between">
                <div>
                  <div className="flex items-center gap-2">
                    <svg className="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span className="text-white font-medium">mDNS Active</span>
                  </div>
                  <code className="text-yellow-neon text-sm font-mono mt-1 block">{mdnsUrl}</code>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      setSelectedUrl(mdnsUrl);
                      setShowQRModal(true);
                    }}
                    className="px-3 py-1.5 bg-gray-700 text-white rounded-lg text-sm hover:bg-gray-600 transition-colors"
                  >
                    Show QR
                  </button>
                  {savedUrl !== mdnsUrl && (
                    <button
                      onClick={() => saveSelectedUrl(mdnsUrl)}
                      disabled={isSaving}
                      className="px-3 py-1.5 bg-blue-500/20 text-blue-400 rounded-lg text-sm hover:bg-blue-500/30 transition-colors disabled:opacity-50"
                    >
                      Set as Default
                    </button>
                  )}
                  {savedUrl === mdnsUrl && (
                    <span className="px-3 py-1.5 bg-yellow-neon/20 text-yellow-neon rounded-lg text-sm">
                      Default
                    </span>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
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

interface PortCheck {
  port: number;
  protocol: string;
  description: string;
  status: 'open' | 'closed' | 'error';
  error?: string;
}

interface DisplayInfo {
  name: string;
  resolution: string;
  type: string;
  connection: string;
  main: boolean;
}

interface DiagnosticsInfo {
  port_checks: PortCheck[];
  displays: DisplayInfo[];
  firewall_enabled: boolean;
  firewall_status: string;
}

interface ServerSettings {
  https_port: string;
  http_port: string;
  admin_pin: string;
  youtube_api_key: string;
  video_player: string;
  data_dir: string;
  // Display settings
  target_display: string;
  auto_fullscreen: boolean;
  // Feature toggles
  pitch_control_enabled: boolean;
  tempo_control_enabled: boolean;
  fair_rotation_enabled: boolean;
  scrolling_ticker_enabled: boolean;
  singer_name_overlay: boolean;
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

// Diagnostics Tab - System info, port checks, displays, firewall status
function DiagnosticsTab() {
  const token = useAdminStore((state) => state.token);
  const [diagnostics, setDiagnostics] = useState<DiagnosticsInfo | null>(null);
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [isDiagnosticsLoading, setIsDiagnosticsLoading] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const API_BASE = window.location.origin;

  // Helper to get auth headers
  const getHeaders = (): HeadersInit => {
    const headers: HeadersInit = {};
    if (token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  };

  // Refresh diagnostics (POST to force re-check)
  const refreshDiagnostics = async () => {
    setIsDiagnosticsLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/admin/diagnostics`, {
        method: 'POST',
        headers: getHeaders(),
      });
      const data = await res.json();
      setDiagnostics(data);
    } catch (err) {
      console.error('Failed to refresh diagnostics:', err);
    } finally {
      setIsDiagnosticsLoading(false);
    }
  };

  // Fetch diagnostics and system info when tab is active
  useEffect(() => {
    const headers = getHeaders();
    Promise.all([
      fetch(`${API_BASE}/api/admin/diagnostics`, { headers }).then(r => r.ok ? r.json() : null),
      fetch(`${API_BASE}/api/admin/system-info`, { headers }).then(r => r.ok ? r.json() : null),
    ]).then(([diagnosticsData, sysInfo]) => {
      if (diagnosticsData) setDiagnostics(diagnosticsData);
      if (sysInfo) setSystemInfo(sysInfo);
      setIsLoading(false);
    }).catch(err => {
      console.error('Failed to fetch diagnostics:', err);
      setIsLoading(false);
    });
  }, [token]);

  if (isLoading) {
    return (
      <div className="p-8 text-center text-gray-500">
        <svg className="w-8 h-8 mx-auto mb-3 animate-spin opacity-50" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        <p>Loading diagnostics...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* System Information */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5">
          <h2 className="text-lg font-semibold text-white">System Information</h2>
          <p className="text-sm text-gray-400">Server hardware and runtime details</p>
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
              <span className="text-white font-mono text-sm">{API_BASE}</span>
            </div>
          </div>
        )}
      </div>

      {/* Hardware Diagnostics */}
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/5 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-white">Hardware Diagnostics</h2>
            <p className="text-sm text-gray-400">Port status, firewall, and connected displays</p>
          </div>
          <button
            onClick={refreshDiagnostics}
            disabled={isDiagnosticsLoading}
            className="flex items-center gap-2 px-3 py-1.5 text-sm bg-matte-black hover:bg-gray-700 text-gray-300 rounded-lg transition-colors disabled:opacity-50"
          >
            {isDiagnosticsLoading ? (
              <>
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                Checking...
              </>
            ) : (
              <>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                Re-check
              </>
            )}
          </button>
        </div>

        {diagnostics ? (
          <div className="p-6 space-y-6">
            {/* Port Status */}
            <div>
              <h3 className="text-white font-medium mb-3 flex items-center gap-2">
                <svg className="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                </svg>
                Port Status
              </h3>
              <div className="grid gap-2">
                {diagnostics.port_checks.map((port, i) => (
                  <div key={i} className="flex items-center justify-between bg-matte-black p-3 rounded-lg">
                    <div className="flex items-center gap-3">
                      <div className={`w-2.5 h-2.5 rounded-full ${
                        port.status === 'open' ? 'bg-green-500' :
                        port.status === 'closed' ? 'bg-red-500' : 'bg-yellow-500'
                      }`} />
                      <div>
                        <span className="text-white font-mono">{port.port}</span>
                        <span className="text-gray-400 text-sm ml-2">({port.protocol.toUpperCase()})</span>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className={`text-sm font-medium ${
                        port.status === 'open' ? 'text-green-400' :
                        port.status === 'closed' ? 'text-red-400' : 'text-yellow-400'
                      }`}>
                        {port.status === 'open' ? 'Open' : port.status === 'closed' ? 'Closed' : 'Error'}
                      </div>
                      <div className="text-gray-500 text-xs">{port.description}</div>
                      {port.error && <div className="text-red-400 text-xs">{port.error}</div>}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Firewall Status */}
            <div>
              <h3 className="text-white font-medium mb-3 flex items-center gap-2">
                <svg className="w-5 h-5 text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
                Firewall
              </h3>
              <div className="flex items-center gap-3 bg-matte-black p-3 rounded-lg">
                <div className={`w-2.5 h-2.5 rounded-full ${diagnostics.firewall_enabled ? 'bg-yellow-500' : 'bg-green-500'}`} />
                <span className="text-white">{diagnostics.firewall_status}</span>
                {diagnostics.firewall_enabled && (
                  <span className="text-yellow-400 text-sm ml-auto">May block connections</span>
                )}
              </div>
            </div>

            {/* Connected Displays */}
            <div>
              <h3 className="text-white font-medium mb-3 flex items-center gap-2">
                <svg className="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                </svg>
                Connected Displays ({diagnostics.displays.length})
              </h3>
              {diagnostics.displays.length > 0 ? (
                <div className="grid gap-2">
                  {diagnostics.displays.map((display, i) => (
                    <div key={i} className="flex items-center justify-between bg-matte-black p-3 rounded-lg">
                      <div className="flex items-center gap-3">
                        <div className={`w-2.5 h-2.5 rounded-full ${display.main ? 'bg-blue-500' : 'bg-gray-500'}`} />
                        <div>
                          <span className="text-white">{display.name}</span>
                          {display.main && (
                            <span className="ml-2 text-xs bg-blue-500/20 text-blue-400 px-2 py-0.5 rounded">Main</span>
                          )}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-gray-300 font-mono text-sm">{display.resolution}</div>
                        <div className="text-gray-500 text-xs">{display.type} • {display.connection}</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-gray-500 text-center py-4">No displays detected</div>
              )}
            </div>
          </div>
        ) : (
          <div className="p-8 text-center text-gray-500">
            <svg className="w-8 h-8 mx-auto mb-3 animate-spin opacity-50" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            <p>Loading diagnostics...</p>
          </div>
        )}
      </div>
    </div>
  );
}

function GeneralSettings() {
  const token = useAdminStore((state) => state.token);
  const [settings, setSettings] = useState<ServerSettings>({
    https_port: '8443',
    http_port: '8080',
    admin_pin: '',
    youtube_api_key: '',
    video_player: 'mpv',
    data_dir: './data',
    // Display settings
    target_display: '',
    auto_fullscreen: true,
    // Feature toggles
    pitch_control_enabled: true,
    tempo_control_enabled: true,
    fair_rotation_enabled: false,
    scrolling_ticker_enabled: true,
    singer_name_overlay: true,
  });
  const [availableDisplays, setAvailableDisplays] = useState<DisplayInfo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const { activeHelp, openHelp, closeHelp } = useHelpModal();

  // Helper to get auth headers
  const getAuthHeaders = (): HeadersInit => {
    const headers: HeadersInit = { 'Content-Type': 'application/json' };
    if (token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  };
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


  // MPV Setup Modal
  const [showMpvSetup, setShowMpvSetup] = useState(false);

  const fetchBgmSettings = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/admin/bgm`, { headers: getAuthHeaders() });
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
        headers: getAuthHeaders(),
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
      const res = await fetch(`${API_BASE}/api/admin/icecast-streams`, { headers: getAuthHeaders() });
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
      const res = await fetch(`${API_BASE}/api/admin/database`, { headers: getAuthHeaders() });
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
        headers: getAuthHeaders(),
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
      const res = await fetch(`${API_BASE}/api/admin/player`, { headers: getAuthHeaders() });
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
        headers: getAuthHeaders(),
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
    const headers: HeadersInit = {};
    if (token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
    }

    Promise.all([
      fetch(`${API_BASE}/api/admin/settings`, { headers }).then(r => {
        if (!r.ok) throw new Error(`Settings API returned ${r.status}`);
        return r.json();
      }),
      fetch(`${API_BASE}/api/admin/diagnostics`, { headers }).then(r => {
        if (!r.ok) return null;
        return r.json();
      }),
    ]).then(([settingsData, diagnosticsData]) => {
      if (settingsData && typeof settingsData === 'object') {
        setSettings(prev => ({ ...prev, ...settingsData }));
      }
      if (diagnosticsData?.displays) {
        setAvailableDisplays(diagnosticsData.displays);
      }
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
  }, [token]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setMessage(null);

    try {
      const res = await fetch(`${API_BASE}/api/admin/settings`, {
        method: 'POST',
        headers: getAuthHeaders(),
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

          {/* YouTube Settings */}
          {bgmSettings.source_type === 'youtube' && (
            <div className="space-y-3">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <label className="block text-sm text-gray-400">YouTube API Key</label>
                  <HelpButton onClick={() => openHelp('youtubeApi')} />
                </div>
                <input
                  type="text"
                  value={settings.youtube_api_key}
                  onChange={(e) => setSettings({ ...settings, youtube_api_key: e.target.value })}
                  placeholder="Enter YouTube Data API v3 key"
                  className="w-full px-4 py-2 bg-matte-black rounded-lg border border-white/10 text-white placeholder-gray-500 focus:outline-none focus:border-yellow-neon font-mono text-sm"
                />
                <p className="text-xs text-gray-500 mt-1">Required for YouTube search and BGM playback</p>
              </div>
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
              type="password"
              value={settings.admin_pin}
              onChange={(e) => setSettings({ ...settings, admin_pin: e.target.value })}
              placeholder="Enter PIN for remote admin access"
              className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
            />
          </div>

          <div>
            <div className="flex items-center gap-2 mb-1">
              <label className="block text-sm text-gray-400">Video Player Path</label>
              <HelpButton onClick={() => openHelp('videoPlayer')} />
            </div>
            <div className="flex gap-2">
              <input
                type="text"
                value={settings.video_player}
                onChange={(e) => setSettings({ ...settings, video_player: e.target.value })}
                placeholder="mpv"
                className="flex-1 px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon font-mono"
              />
              <button
                type="button"
                onClick={() => setShowMpvSetup(true)}
                className="px-4 py-3 bg-blue-500/20 text-blue-400 rounded-xl hover:bg-blue-500/30 transition-colors flex items-center gap-2 whitespace-nowrap"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                Install mpv
              </button>
            </div>
          </div>

          {/* Display Settings */}
          <div className="border-t border-white/5 pt-4 mt-4">
            <h3 className="text-md font-semibold text-white mb-4">Display Settings</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Target Display</label>
                <select
                  value={settings.target_display}
                  onChange={(e) => setSettings({ ...settings, target_display: e.target.value })}
                  className="w-full px-4 py-3 bg-matte-black rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-yellow-neon"
                >
                  <option value="">Auto (Primary Display)</option>
                  {availableDisplays.map((display, i) => (
                    <option key={i} value={display.name}>
                      {display.name} {display.resolution && `(${display.resolution})`} {display.main && '★'}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-gray-500 mt-1">
                  {availableDisplays.length === 0
                    ? 'No displays detected. Check Network tab for diagnostics.'
                    : `${availableDisplays.length} display(s) detected`}
                </p>
              </div>

              <label className="flex items-center justify-between cursor-pointer group">
                <div>
                  <span className="text-white group-hover:text-yellow-neon transition-colors">Auto Fullscreen</span>
                  <p className="text-xs text-gray-500">Automatically fullscreen the player on startup</p>
                </div>
                <div className={`relative w-12 h-7 rounded-full transition-colors ${settings.auto_fullscreen ? 'bg-yellow-neon' : 'bg-gray-600'}`}>
                  <input
                    type="checkbox"
                    checked={settings.auto_fullscreen}
                    onChange={(e) => setSettings({ ...settings, auto_fullscreen: e.target.checked })}
                    className="sr-only"
                  />
                  <div className={`absolute top-1 w-5 h-5 rounded-full bg-white shadow transition-transform ${settings.auto_fullscreen ? 'translate-x-6' : 'translate-x-1'}`} />
                </div>
              </label>
            </div>
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

          {/* Feature Toggles */}
          <div className="border-t border-white/5 pt-4 mt-4">
            <h3 className="text-md font-semibold text-white mb-4">Feature Toggles</h3>
            <div className="space-y-3">
              <label className="flex items-center justify-between cursor-pointer group">
                <div>
                  <span className="text-white group-hover:text-yellow-neon transition-colors">Pitch Control</span>
                  <p className="text-xs text-gray-500">Allow key/pitch changes for songs</p>
                </div>
                <div className={`relative w-12 h-7 rounded-full transition-colors ${settings.pitch_control_enabled ? 'bg-yellow-neon' : 'bg-gray-600'}`}>
                  <input
                    type="checkbox"
                    checked={settings.pitch_control_enabled}
                    onChange={(e) => setSettings({ ...settings, pitch_control_enabled: e.target.checked })}
                    className="sr-only"
                  />
                  <div className={`absolute top-1 w-5 h-5 rounded-full bg-white shadow transition-transform ${settings.pitch_control_enabled ? 'translate-x-6' : 'translate-x-1'}`} />
                </div>
              </label>

              <label className="flex items-center justify-between cursor-pointer group">
                <div>
                  <span className="text-white group-hover:text-yellow-neon transition-colors">Tempo Control</span>
                  <p className="text-xs text-gray-500">Allow speed/tempo changes for songs</p>
                </div>
                <div className={`relative w-12 h-7 rounded-full transition-colors ${settings.tempo_control_enabled ? 'bg-yellow-neon' : 'bg-gray-600'}`}>
                  <input
                    type="checkbox"
                    checked={settings.tempo_control_enabled}
                    onChange={(e) => setSettings({ ...settings, tempo_control_enabled: e.target.checked })}
                    className="sr-only"
                  />
                  <div className={`absolute top-1 w-5 h-5 rounded-full bg-white shadow transition-transform ${settings.tempo_control_enabled ? 'translate-x-6' : 'translate-x-1'}`} />
                </div>
              </label>

              <label className="flex items-center justify-between cursor-pointer group">
                <div>
                  <span className="text-white group-hover:text-yellow-neon transition-colors">Singer Name Overlay</span>
                  <p className="text-xs text-gray-500">Show singer name at start of each song</p>
                </div>
                <div className={`relative w-12 h-7 rounded-full transition-colors ${settings.singer_name_overlay ? 'bg-yellow-neon' : 'bg-gray-600'}`}>
                  <input
                    type="checkbox"
                    checked={settings.singer_name_overlay}
                    onChange={(e) => setSettings({ ...settings, singer_name_overlay: e.target.checked })}
                    className="sr-only"
                  />
                  <div className={`absolute top-1 w-5 h-5 rounded-full bg-white shadow transition-transform ${settings.singer_name_overlay ? 'translate-x-6' : 'translate-x-1'}`} />
                </div>
              </label>

              <label className="flex items-center justify-between cursor-pointer group">
                <div>
                  <span className="text-white group-hover:text-yellow-neon transition-colors">Scrolling Ticker</span>
                  <p className="text-xs text-gray-500">Show upcoming singers on display</p>
                </div>
                <div className={`relative w-12 h-7 rounded-full transition-colors ${settings.scrolling_ticker_enabled ? 'bg-yellow-neon' : 'bg-gray-600'}`}>
                  <input
                    type="checkbox"
                    checked={settings.scrolling_ticker_enabled}
                    onChange={(e) => setSettings({ ...settings, scrolling_ticker_enabled: e.target.checked })}
                    className="sr-only"
                  />
                  <div className={`absolute top-1 w-5 h-5 rounded-full bg-white shadow transition-transform ${settings.scrolling_ticker_enabled ? 'translate-x-6' : 'translate-x-1'}`} />
                </div>
              </label>

              <label className="flex items-center justify-between cursor-pointer group">
                <div>
                  <span className="text-white group-hover:text-yellow-neon transition-colors">Fair Rotation</span>
                  <p className="text-xs text-gray-500">Queue songs by singer rotation instead of FIFO</p>
                </div>
                <div className={`relative w-12 h-7 rounded-full transition-colors ${settings.fair_rotation_enabled ? 'bg-yellow-neon' : 'bg-gray-600'}`}>
                  <input
                    type="checkbox"
                    checked={settings.fair_rotation_enabled}
                    onChange={(e) => setSettings({ ...settings, fair_rotation_enabled: e.target.checked })}
                    className="sr-only"
                  />
                  <div className={`absolute top-1 w-5 h-5 rounded-full bg-white shadow transition-transform ${settings.fair_rotation_enabled ? 'translate-x-6' : 'translate-x-1'}`} />
                </div>
              </label>
            </div>
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
    <MPVSetupModal isOpen={showMpvSetup} onClose={() => setShowMpvSetup(false)} />
    </>
  );
}

function SearchLogs() {
  const token = useAdminStore((state) => state.token);
  const [logs, setLogs] = useState<SearchLogEntry[]>([]);
  const [stats, setStats] = useState<SearchStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [sourceFilter, setSourceFilter] = useState<string>('');
  const [showStats, setShowStats] = useState(true);

  // Helper to get auth headers
  const getAuthHeaders = (): HeadersInit => {
    const headers: HeadersInit = {};
    if (token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
    }
    return headers;
  };

  const fetchLogs = async () => {
    try {
      const url = sourceFilter
        ? `${API_BASE}/api/admin/search-logs?source=${sourceFilter}`
        : `${API_BASE}/api/admin/search-logs`;
      const res = await fetch(url, { headers: getAuthHeaders() });
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
      const res = await fetch(`${API_BASE}/api/admin/search-stats`, { headers: getAuthHeaders() });
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
        headers: getAuthHeaders(),
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
  const idle = useRoomStore(selectIdle);
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
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 border-b border-white/5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-white">Song Queue</h2>
              <p className="text-sm text-gray-400">
                {queueSongs.length} upcoming • {historySongs.length} in history
              </p>
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

          {/* Pitch & Tempo Controls (only when playing) */}
          {!idle && (
            <div className="mt-4 pt-4 border-t border-white/5">
              <div className="flex items-center gap-6">
                {/* Key Change (Pitch) */}
                <div className="flex-1">
                  <div className="flex items-center justify-between mb-1">
                    <label className="text-sm text-gray-400">Key</label>
                    <span className="text-sm font-mono text-white">
                      {(() => {
                        const currentSong = queue[currentPosition];
                        const semitones = currentSong?.key_change || 0;
                        if (semitones === 0) return '0';
                        return semitones > 0 ? `+${semitones}` : `${semitones}`;
                      })()}
                    </span>
                  </div>
                  <input
                    type="range"
                    min="-12"
                    max="12"
                    step="1"
                    value={queue[currentPosition]?.key_change || 0}
                    onChange={(e) => wsService.setKeyChange(parseInt(e.target.value, 10))}
                    className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-yellow-neon"
                  />
                  <div className="flex justify-between text-xs text-gray-500 mt-0.5">
                    <span>-12</span>
                    <span>0</span>
                    <span>+12</span>
                  </div>
                </div>

                {/* Tempo */}
                <div className="flex-1">
                  <div className="flex items-center justify-between mb-1">
                    <label className="text-sm text-gray-400">Tempo</label>
                    <span className="text-sm font-mono text-white">
                      {((queue[currentPosition]?.tempo_change || 1) * 100).toFixed(0)}%
                    </span>
                  </div>
                  <input
                    type="range"
                    min="0.5"
                    max="2.0"
                    step="0.05"
                    value={queue[currentPosition]?.tempo_change || 1}
                    onChange={(e) => wsService.setTempoChange(parseFloat(e.target.value))}
                    className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-cyan-400"
                  />
                  <div className="flex justify-between text-xs text-gray-500 mt-0.5">
                    <span>50%</span>
                    <span>100%</span>
                    <span>200%</span>
                  </div>
                </div>

                {/* Reset Button */}
                <button
                  onClick={() => {
                    wsService.setKeyChange(0);
                    wsService.setTempoChange(1.0);
                  }}
                  className="px-3 py-2 bg-gray-700 text-gray-300 text-sm rounded-lg hover:bg-gray-600 transition-colors"
                >
                  Reset
                </button>
              </div>
            </div>
          )}
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

// Preset messages for quick selection
const PRESET_MESSAGES = [
  'Welcome to Karaoke Night!',
  'Sing your heart out!',
  'Who\'s next?',
  'Let\'s party!',
  'The stage is yours!',
  'Ready to sing?',
];

function HoldingMessageModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [message, setMessage] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const token = useAdminStore(state => state.token);

  // Fetch current message when modal opens
  useEffect(() => {
    if (isOpen && token) {
      setIsLoading(true);
      fetch('/api/admin/holding-message', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
        .then(res => res.json())
        .then(data => {
          setMessage(data.message || '');
          setIsLoading(false);
          setTimeout(() => inputRef.current?.focus(), 100);
        })
        .catch(() => {
          setIsLoading(false);
          setTimeout(() => inputRef.current?.focus(), 100);
        });
    }
  }, [isOpen, token]);

  const handleSave = () => {
    setIsSaving(true);
    wsService.adminSetMessage(message);
    setTimeout(() => {
      setIsSaving(false);
      onClose();
    }, 300);
  };

  const handleClear = () => {
    setIsSaving(true);
    wsService.adminSetMessage('');
    setMessage('');
    setTimeout(() => {
      setIsSaving(false);
      onClose();
    }, 300);
  };

  const selectPreset = (preset: string) => {
    setMessage(preset);
    inputRef.current?.focus();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div
        className="bg-matte-gray rounded-2xl w-full max-w-md overflow-hidden"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-5 py-4 border-b border-white/10 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
              <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5.882V19.24a1.76 1.76 0 01-3.417.592l-2.147-6.15M18 13a3 3 0 100-6M5.436 13.683A4.001 4.001 0 017 6h1.832c4.1 0 7.625-1.234 9.168-3v14c-1.543-1.766-5.067-3-9.168-3H7a3.988 3.988 0 01-1.564-.317z" />
              </svg>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-white">Screen Message</h3>
              <p className="text-xs text-gray-400">Show on the holding screen</p>
            </div>
          </div>
          <button onClick={onClose} className="p-2 text-gray-400 hover:text-white transition-colors">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-5 space-y-4">
          {/* Message Input */}
          <div>
            <input
              ref={inputRef}
              type="text"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder={isLoading ? "Loading..." : "Type your message..."}
              disabled={isLoading}
              maxLength={100}
              className={`w-full px-4 py-3 bg-matte-black rounded-xl text-white text-lg placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-purple-500 ${isLoading ? 'opacity-50' : ''}`}
            />
            <div className="flex justify-between mt-2 text-xs text-gray-500">
              <span>Leave empty to clear</span>
              <span>{message.length}/100</span>
            </div>
          </div>

          {/* Preset Messages */}
          <div>
            <div className="text-xs text-gray-400 uppercase tracking-wide mb-2">Quick Messages</div>
            <div className="grid grid-cols-2 gap-2">
              {PRESET_MESSAGES.map((preset) => (
                <button
                  key={preset}
                  onClick={() => selectPreset(preset)}
                  className="px-3 py-2 text-sm text-left bg-matte-black hover:bg-white/10 rounded-lg text-gray-300 hover:text-white transition-colors truncate"
                >
                  {preset}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="px-5 py-4 border-t border-white/10 flex gap-3">
          <button
            onClick={handleClear}
            disabled={isSaving}
            className="flex-1 py-3 bg-matte-black text-gray-400 font-medium rounded-xl hover:bg-white/10 hover:text-white transition-colors disabled:opacity-50"
          >
            Clear Screen
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving || !message.trim()}
            className="flex-1 py-3 bg-gradient-to-r from-purple-500 to-pink-500 text-white font-bold rounded-xl hover:from-purple-400 hover:to-pink-400 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSaving ? 'Saving...' : 'Set Message'}
          </button>
        </div>
      </div>
    </div>
  );
}

export function Admin() {
  const { isAuthenticated, isLocal, checkAuth, setClients, fetchClients, logout, clients } = useAdminStore();
  const [isLoading, setIsLoading] = useState(true);
  const [searchParams, setSearchParams] = useSearchParams();
  const [showMessageModal, setShowMessageModal] = useState(false);
  const [activeTab, setActiveTabState] = useState<AdminTab>('clients');

  // Sync tab state with URL on mount and when searchParams change
  useEffect(() => {
    const tabFromUrl = searchParams.get('tab') as AdminTab | null;
    const validTabs: AdminTab[] = ['clients', 'queue', 'library', 'search-logs', 'network', 'diagnostics', 'settings'];
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

  // Playback state for header controls
  const queue = useRoomStore(selectQueue);
  const currentPosition = useRoomStore(selectQueuePosition);
  const countdown = useRoomStore(selectCountdown);
  const idle = useRoomStore(selectIdle);
  const bgmActive = useRoomStore(selectBgmActive);
  const bgmEnabled = useRoomStore(selectBgmEnabled);

  // Derived playback state
  const isPlaying = !idle || countdown.active;
  const canToggleBGM = idle && !countdown.active;
  const queueSongs = queue.filter((_, index) => index >= currentPosition);
  const currentSong = !idle && currentPosition < queue.length ? queue[currentPosition] : null;

  // Playback handlers
  const handlePlay = () => {
    wsService.adminPlayNext();
  };

  const handleStartNow = () => {
    wsService.adminStartNow();
  };

  const handleStop = () => {
    if (!idle || countdown.active) {
      wsService.adminStop();
    }
  };

  const handleToggleBGM = () => {
    wsService.adminToggleBGM();
  };

  // Helper to get singer name from MartynKey
  const getSingerName = (martynKey: string): string => {
    const client = clients.find((c) => c.martyn_key === martynKey);
    return client?.display_name || 'Unknown';
  };

  // Get next song info
  const nextSong = queueSongs[0];

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
    diagnostics: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
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
      {/* Header - Clean and simple */}
      <header className="px-4 py-3 bg-matte-gray/50 backdrop-blur-sm border-b border-white/5">
        <div className="flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
            <img src="/logo.jpeg" alt="SongMartyn" className="w-9 h-9 rounded-lg object-cover" />
            <div>
              <h1 className="text-lg font-bold text-white leading-tight">
                Song<span className="text-yellow-neon">Martyn</span>
              </h1>
              <span className="text-xs text-gray-400">Admin</span>
            </div>
          </Link>

          <div className="flex items-center gap-1">
            {isLocal && (
              <span className="hidden sm:inline-block text-xs text-green-400 bg-green-500/10 px-2 py-1 rounded-full mr-1">
                Local
              </span>
            )}
            {/* Screen Message Button */}
            <button
              onClick={() => setShowMessageModal(true)}
              className="p-2 text-gray-400 hover:text-purple-400 transition-colors"
              title="Set Screen Message"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5.882V19.24a1.76 1.76 0 01-3.417.592l-2.147-6.15M18 13a3 3 0 100-6M5.436 13.683A4.001 4.001 0 017 6h1.832c4.1 0 7.625-1.234 9.168-3v14c-1.543-1.766-5.067-3-9.168-3H7a3.988 3.988 0 01-1.564-.317z" />
              </svg>
            </button>
            {!isLocal && (
              <button
                onClick={logout}
                className="p-2 text-gray-400 hover:text-white transition-colors"
                title="Logout"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                </svg>
              </button>
            )}
          </div>
        </div>
      </header>

      {/* Player Controls Bar */}
      <div className={`relative px-4 py-3 border-b border-white/10 overflow-hidden ${
        countdown.active
          ? 'bg-matte-gray'
          : 'bg-gradient-to-r from-indigo-deep via-matte-gray to-indigo-deep'
      }`}>
        {/* Progress bar background for countdown */}
        {countdown.active && (
          <div
            className={`absolute inset-0 transition-all duration-1000 ease-linear ${
              countdown.requires_approval
                ? 'bg-gradient-to-r from-orange-500/30 to-amber-500/30'
                : 'bg-gradient-to-r from-cyan-500/30 to-cyan-400/30'
            }`}
            style={{
              width: `${(countdown.seconds_remaining / 10) * 100}%`,
              transition: 'width 1s linear'
            }}
          />
        )}
        <div className="relative flex items-center justify-between gap-3">
          {/* Song Info / Status */}
          <div className="flex-1 min-w-0">
            {countdown.active ? (
              // Countdown active - show countdown with song info
              <div className="flex items-center gap-3">
                <div className={`w-12 h-12 rounded-full flex items-center justify-center text-xl font-bold flex-shrink-0 shadow-lg ${
                  countdown.requires_approval
                    ? 'bg-gradient-to-br from-orange-500 to-amber-500 text-white animate-pulse'
                    : 'bg-gradient-to-br from-cyan-500 to-cyan-400 text-white'
                }`}>
                  {countdown.seconds_remaining}
                </div>
                <div className="min-w-0">
                  <div className="text-sm font-semibold text-white truncate">
                    {countdown.requires_approval ? 'Waiting for approval...' : 'Starting soon...'}
                  </div>
                  <div className="text-base font-medium text-white truncate">
                    {nextSong?.title || 'Next Song'}
                  </div>
                  <div className="text-xs text-gray-300 truncate">
                    {nextSong ? getSingerName(nextSong.added_by) : 'Loading...'}
                  </div>
                </div>
              </div>
            ) : isPlaying && currentSong ? (
              // Playing - show current song
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-yellow-neon to-amber-500 flex items-center justify-center flex-shrink-0">
                  <svg className="w-5 h-5 text-gray-900" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z" />
                  </svg>
                </div>
                <div className="min-w-0">
                  <div className="text-sm font-medium text-white truncate">
                    {currentSong.title}
                  </div>
                  <div className="text-xs text-yellow-neon truncate">
                    Now Playing
                  </div>
                </div>
              </div>
            ) : nextSong && idle ? (
              // Idle with next song queued
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-white/10 flex items-center justify-center flex-shrink-0">
                  <svg className="w-5 h-5 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </div>
                <div className="min-w-0">
                  <div className="text-sm font-medium text-white truncate">
                    {nextSong.title}
                  </div>
                  <div className="text-xs text-gray-400 truncate">
                    Up Next • {getSingerName(nextSong.added_by)}
                  </div>
                </div>
              </div>
            ) : (
              // Idle - no songs
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-white/5 flex items-center justify-center flex-shrink-0">
                  <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
                  </svg>
                </div>
                <div className="min-w-0">
                  <div className="text-sm text-gray-400">
                    {bgmActive ? 'Background Music' : 'Waiting for songs...'}
                  </div>
                  <div className="text-xs text-gray-500">
                    {queueSongs.length} in queue
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-2 flex-shrink-0">
            {/* BGM Button - only when idle and enabled */}
            {canToggleBGM && bgmEnabled && (
              <button
                onClick={handleToggleBGM}
                className={`p-2.5 rounded-xl transition-all ${
                  bgmActive
                    ? 'bg-gradient-to-br from-purple-500 to-purple-600 text-white shadow-lg shadow-purple-500/30'
                    : 'bg-white/10 text-gray-400 hover:bg-white/20 hover:text-white'
                }`}
                title={bgmActive ? 'Stop Background Music' : 'Play Background Music'}
              >
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z" />
                </svg>
              </button>
            )}

            {/* Start Now button during countdown */}
            {countdown.active && (
              <button
                onClick={handleStartNow}
                className={`px-3 py-2 text-sm font-bold rounded-xl transition-all ${
                  countdown.requires_approval
                    ? 'bg-gradient-to-r from-orange-500 to-amber-500 text-white'
                    : 'bg-gradient-to-r from-cyan-500 to-cyan-400 text-gray-900'
                }`}
              >
                Start
              </button>
            )}

            {/* Play/Stop Button */}
            {isPlaying ? (
              <button
                onClick={handleStop}
                className="flex items-center gap-1.5 px-4 py-2.5 bg-gradient-to-r from-yellow-neon to-amber-400 text-gray-900 font-bold rounded-xl transition-all shadow-lg shadow-yellow-500/30 hover:shadow-yellow-400/40 hover:scale-105 active:scale-95"
              >
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M6 6h12v12H6z" />
                </svg>
                <span className="hidden sm:inline">Stop</span>
              </button>
            ) : (
              <button
                onClick={handlePlay}
                disabled={queueSongs.length === 0}
                className="flex items-center gap-1.5 px-4 py-2.5 bg-gradient-to-r from-cyan-500 to-cyan-400 text-gray-900 font-bold rounded-xl transition-all disabled:opacity-40 disabled:cursor-not-allowed shadow-lg shadow-cyan-500/30 hover:shadow-cyan-400/40 hover:scale-105 active:scale-95"
              >
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z" />
                </svg>
                <span className="hidden sm:inline">Play</span>
              </button>
            )}
          </div>
        </div>
      </div>

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
            label="Diagnostics"
            icon={icons.diagnostics}
            active={activeTab === 'diagnostics'}
            onClick={() => setActiveTab('diagnostics')}
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
      <main className="p-6 max-w-4xl mx-auto pb-10">
        {activeTab === 'clients' && <ClientList />}
        {activeTab === 'queue' && <QueueManagement />}
        {activeTab === 'library' && <LibraryManagement />}
        {activeTab === 'search-logs' && <SearchLogs />}
        {activeTab === 'network' && <NetworkSettings />}
        {activeTab === 'diagnostics' && <DiagnosticsTab />}
        {activeTab === 'settings' && <GeneralSettings />}
      </main>

      <Footer />

      {/* Screen Message Modal */}
      <HoldingMessageModal
        isOpen={showMessageModal}
        onClose={() => setShowMessageModal(false)}
      />
    </div>
  );
}
