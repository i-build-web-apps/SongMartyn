import { useState, useEffect } from 'react';
import { useAdminStore } from '../stores/adminStore';

interface MPVStatus {
  installed: boolean;
  version?: string;
  path?: string;
  error?: string;
  platform: string;
  install_command: string;
  install_method: string;
  download_url: string;
  alternative_note?: string;
}

interface MPVSetupModalProps {
  isOpen: boolean;
  onClose: () => void;
}

// Platform display names
const PLATFORM_NAMES: Record<string, string> = {
  darwin: 'macOS',
  linux: 'Linux',
  windows: 'Windows',
};

export function MPVSetupModal({ isOpen, onClose }: MPVSetupModalProps) {
  const token = useAdminStore((state) => state.token);
  const [status, setStatus] = useState<MPVStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const fetchStatus = async () => {
    setLoading(true);
    setError(null);
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }

      const response = await fetch('/api/admin/mpv-check', { headers });
      if (!response.ok) {
        throw new Error('Failed to check MPV status');
      }
      const data = await response.json();
      setStatus(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      fetchStatus();
    }
  }, [isOpen]);

  const copyCommand = async () => {
    if (status?.install_command) {
      await navigator.clipboard.writeText(status.install_command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/70" onClick={onClose} />
      <div className="relative bg-matte-gray rounded-2xl w-full max-w-lg overflow-hidden flex flex-col animate-slide-up">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
          <div className="flex items-center gap-3">
            <div className={`p-2 rounded-lg ${status?.installed ? 'bg-green-500/20' : 'bg-orange-500/20'}`}>
              {status?.installed ? (
                <svg className="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              ) : (
                <svg className="w-5 h-5 text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              )}
            </div>
            <h2 className="text-xl font-bold text-white">MPV Video Player Setup</h2>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-gray-400 hover:text-white transition-colors rounded-lg hover:bg-white/5"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-2 border-yellow-neon border-t-transparent"></div>
            </div>
          ) : error ? (
            <div className="bg-red-500/20 rounded-xl p-4 text-red-400">
              {error}
            </div>
          ) : status ? (
            <>
              {/* Status Banner */}
              <div className={`rounded-xl p-4 ${status.installed ? 'bg-green-500/10 border border-green-500/30' : 'bg-orange-500/10 border border-orange-500/30'}`}>
                <div className="flex items-start gap-3">
                  {status.installed ? (
                    <>
                      <svg className="w-6 h-6 text-green-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      <div>
                        <h3 className="text-green-400 font-semibold">MPV is installed</h3>
                        <p className="text-gray-300 text-sm mt-1">{status.version}</p>
                        <p className="text-gray-500 text-xs mt-1 font-mono">{status.path}</p>
                      </div>
                    </>
                  ) : (
                    <>
                      <svg className="w-6 h-6 text-orange-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                      </svg>
                      <div>
                        <h3 className="text-orange-400 font-semibold">MPV is not installed</h3>
                        <p className="text-gray-300 text-sm mt-1">
                          SongMartyn requires mpv to play karaoke videos. Follow the instructions below to install it.
                        </p>
                      </div>
                    </>
                  )}
                </div>
              </div>

              {/* Platform Info */}
              <div>
                <h3 className="text-lg font-semibold text-yellow-neon mb-3">
                  Installation for {PLATFORM_NAMES[status.platform] || status.platform}
                </h3>

                {/* Install Command */}
                <div className="bg-matte-black rounded-xl p-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-gray-400 text-sm">
                      Install via {status.install_method}
                    </span>
                    <button
                      onClick={copyCommand}
                      className="text-xs px-2 py-1 rounded bg-white/5 text-gray-400 hover:text-white hover:bg-white/10 transition-colors flex items-center gap-1"
                    >
                      {copied ? (
                        <>
                          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                          </svg>
                          Copied!
                        </>
                      ) : (
                        <>
                          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                          </svg>
                          Copy
                        </>
                      )}
                    </button>
                  </div>
                  <code className="text-yellow-neon font-mono text-sm block">
                    {status.install_command}
                  </code>
                </div>

                {/* Alternative Note */}
                {status.alternative_note && (
                  <p className="text-gray-400 text-sm mt-3">
                    {status.alternative_note}
                  </p>
                )}
              </div>

              {/* Download Link */}
              <div>
                <a
                  href={status.download_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg text-sm font-medium hover:bg-blue-500/30 transition-colors"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                  Official mpv Downloads
                </a>
              </div>

              {/* What is mpv? */}
              <div className="border-t border-white/10 pt-4">
                <h3 className="text-lg font-semibold text-yellow-neon mb-3">What is mpv?</h3>
                <p className="text-gray-300 text-sm leading-relaxed">
                  mpv is a free, open-source media player that SongMartyn uses to display karaoke videos.
                  It's fast, lightweight, and supports virtually all video formats. mpv is developed by
                  volunteers and is completely free to use.
                </p>
              </div>
            </>
          ) : null}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-white/10 bg-matte-black/30 flex gap-3">
          <button
            onClick={fetchStatus}
            disabled={loading}
            className="flex-1 py-3 bg-matte-black text-white font-semibold rounded-xl hover:bg-white/10 transition-colors flex items-center justify-center gap-2 disabled:opacity-50"
          >
            <svg className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Check Again
          </button>
          <button
            onClick={onClose}
            className="flex-1 py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform"
          >
            {status?.installed ? 'Done' : 'Close'}
          </button>
        </div>
      </div>
    </div>
  );
}
