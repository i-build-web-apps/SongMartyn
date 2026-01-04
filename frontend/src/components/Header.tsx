import { useState, useEffect, useCallback } from 'react';
import { useRoomStore } from '../stores/roomStore';

const API_BASE = import.meta.env.DEV ? 'https://localhost:8443' : '';

// QR code generator using public API
const generateQRCodeUrl = (url: string) => {
  return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(url)}`;
};

export function Header({ onOpenSettings }: { onOpenSettings?: () => void }) {
  const [showQR, setShowQR] = useState(false);
  const [connectUrl, setConnectUrl] = useState(window.location.origin);
  const [isLocal, setIsLocal] = useState(false);
  const session = useRoomStore((state) => state.session);
  const isConnected = useRoomStore((state) => state.isConnected);
  const isConnecting = useRoomStore((state) => state.isConnecting);

  // Fetch the connect URL from API
  const fetchConnectUrl = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/connect-url`);
      const data = await res.json();
      if (data?.url) {
        setConnectUrl(data.url);
      }
    } catch {
      // Fallback to current origin
    }
  }, []);

  // Check if local user
  useEffect(() => {
    fetch(`${API_BASE}/api/admin/check`)
      .then(r => r.json())
      .then(data => {
        setIsLocal(data.is_local === true);
      })
      .catch(() => {});

    // Initial fetch of connect URL
    fetchConnectUrl();
  }, [fetchConnectUrl]);

  // Re-fetch connect URL when QR modal opens
  const handleOpenQR = () => {
    fetchConnectUrl();
    setShowQR(true);
  };

  // Connection status display
  const connectionDisplay = () => {
    if (isConnecting) {
      return (
        <div className="flex items-center gap-2 text-yellow-neon">
          <div className="w-2 h-2 bg-yellow-neon rounded-full animate-pulse" />
          <span className="text-sm">Connecting...</span>
        </div>
      );
    }
    if (!isConnected) {
      return (
        <div className="flex items-center gap-2 text-red-400">
          <div className="w-2 h-2 bg-red-400 rounded-full" />
          <span className="text-sm">Disconnected</span>
        </div>
      );
    }
    // Connected - show clickable username with avatar
    return (
      <button
        onClick={onOpenSettings}
        className="flex items-center gap-2 text-green-400 hover:text-yellow-neon transition-colors group"
        title="User Settings"
      >
        {/* Pixel avatar */}
        {session?.avatar_id ? (
          <img
            src={`/avatars/${session.avatar_id}.png`}
            alt=""
            className="w-6 h-6 rounded"
          />
        ) : (
          <div className="w-2 h-2 bg-green-400 rounded-full group-hover:bg-yellow-neon" />
        )}
        <span className="text-sm group-hover:underline">
          {session?.display_name || 'Connected'}
        </span>
        <svg className="w-4 h-4 opacity-0 group-hover:opacity-100 transition-opacity" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      </button>
    );
  };

  return (
    <>
      <header className="flex items-center justify-between px-4 py-3 bg-matte-gray/50 backdrop-blur-sm border-b border-white/5">
        {/* Logo */}
        <div className="flex items-center gap-2">
          <img src="/logo.jpeg" alt="SongMartyn" className="w-10 h-10 rounded-lg object-cover" />
          <span className="text-xl font-bold text-white">
            Song<span className="text-yellow-neon">Martyn</span>
          </span>
        </div>

        {/* Right side: Admin link (local only) + QR code button + Connection status */}
        <div className="flex items-center gap-2">
          {/* Admin link - only for local users */}
          {isLocal && (
            <a
              href="/admin"
              className="p-2 text-gray-400 hover:text-yellow-neon transition-colors"
              title="Admin Panel"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
            </a>
          )}

          {/* QR Code button */}
          <button
            onClick={handleOpenQR}
            className="p-2 text-gray-400 hover:text-yellow-neon transition-colors"
            title="Show QR Code"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
            </svg>
          </button>

          {/* Connection status */}
          {connectionDisplay()}
        </div>
      </header>

      {/* QR Code Modal */}
      {showQR && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80"
          onClick={() => setShowQR(false)}
        >
          <div
            className="bg-matte-gray rounded-2xl p-6 max-w-sm w-full"
            onClick={e => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Invite Friends</h3>
              <button
                onClick={() => setShowQR(false)}
                className="text-gray-400 hover:text-white"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="bg-white p-4 rounded-xl mb-4 flex items-center justify-center">
              <img
                src={generateQRCodeUrl(connectUrl)}
                alt="QR Code"
                className="w-48 h-48"
              />
            </div>

            <div className="text-center space-y-3">
              <p className="text-gray-400 text-sm">
                Scan with phone camera to join the karaoke session
              </p>
              <code className="text-yellow-neon text-sm font-mono bg-matte-black px-3 py-2 rounded-lg block overflow-x-auto">
                {connectUrl}
              </code>
              <p className="text-gray-500 text-xs">
                Make sure guests are on the same WiFi network
              </p>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
