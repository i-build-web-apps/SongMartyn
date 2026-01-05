import { useEffect, useState } from 'react';

const API_BASE = import.meta.env.DEV ? 'https://localhost:8443' : '';

interface ServiceStatus {
  status: 'connected' | 'unavailable';
}

interface StatusData {
  database: ServiceStatus;
  websocket: ServiceStatus;
  library: ServiceStatus;
  media_player: ServiceStatus;
  internet: ServiceStatus;
}

function StatusBadge({ status }: { status: 'connected' | 'unavailable' }) {
  const isConnected = status === 'connected';
  return (
    <span className={`px-3 py-1 rounded-full text-sm font-medium ${
      isConnected
        ? 'bg-green-500/20 text-green-400'
        : 'bg-red-500/20 text-red-400'
    }`}>
      {isConnected ? 'Connected' : 'Unavailable'}
    </span>
  );
}

function StatusRow({ label, status }: { label: string; status: ServiceStatus }) {
  return (
    <div className="flex items-center justify-between py-4 border-b border-white/5 last:border-0">
      <span className="text-gray-300">{label}</span>
      <StatusBadge status={status.status} />
    </div>
  );
}

export function Status() {
  const [status, setStatus] = useState<StatusData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchStatus = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/status`);
      if (!res.ok) throw new Error('Failed to fetch status');
      const data = await res.json();
      setStatus(data);
      setLastUpdated(new Date());
      setError(null);
    } catch (err) {
      setError('Unable to connect to server');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    // Refresh every 30 seconds
    const interval = setInterval(fetchStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="min-h-screen bg-matte-black p-4">
      <div className="max-w-md mx-auto">
        {/* Header */}
        <div className="text-center mb-8 pt-8">
          <h1 className="text-2xl font-bold text-white mb-2">System Status</h1>
          <p className="text-gray-500 text-sm">SongMartyn Service Health</p>
        </div>

        {/* Status Card */}
        <div className="bg-matte-gray rounded-2xl p-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <div className="w-8 h-8 border-2 border-yellow-neon border-t-transparent rounded-full animate-spin" />
            </div>
          ) : error ? (
            <div className="text-center py-12">
              <div className="text-4xl mb-4">⚠️</div>
              <p className="text-red-400">{error}</p>
              <button
                onClick={fetchStatus}
                className="mt-4 px-4 py-2 bg-matte-black text-gray-400 rounded-lg hover:text-white transition-colors"
              >
                Retry
              </button>
            </div>
          ) : status ? (
            <>
              <StatusRow label="Database" status={status.database} />
              <StatusRow label="WebSocket Server" status={status.websocket} />
              <StatusRow label="Music Library" status={status.library} />
              <StatusRow label="Media Player" status={status.media_player} />
              <StatusRow label="Internet Connection" status={status.internet} />
            </>
          ) : null}
        </div>

        {/* Last Updated */}
        {lastUpdated && (
          <p className="text-center text-gray-600 text-xs mt-4">
            Last updated: {lastUpdated.toLocaleTimeString()}
          </p>
        )}

        {/* Refresh Button */}
        <div className="text-center mt-6">
          <button
            onClick={fetchStatus}
            disabled={loading}
            className="px-6 py-2 bg-matte-gray text-gray-400 rounded-lg hover:text-white transition-colors disabled:opacity-50"
          >
            {loading ? 'Checking...' : 'Refresh'}
          </button>
        </div>

        {/* Back Link */}
        <div className="text-center mt-8">
          <a href="/" className="text-yellow-neon hover:underline text-sm">
            ← Back to SongMartyn
          </a>
        </div>
      </div>
    </div>
  );
}
