import { useState } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import { useRoomStore } from '../stores/roomStore';
import { wsService } from '../services/websocket';
import { Header } from '../components/Header';
import { NowPlaying } from '../components/NowPlaying';
import { Search } from '../components/Search';
import { UserSettings } from '../components/UserSettings';
import { StatusBar } from '../components/StatusBar';

function BlockedPage({ reason }: { reason: string }) {
  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="bg-matte-gray rounded-2xl p-8 max-w-md w-full text-center">
        <div className="w-20 h-20 mx-auto mb-6 rounded-full bg-red-500/20 flex items-center justify-center">
          <svg className="w-10 h-10 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
          </svg>
        </div>

        <h1 className="text-2xl font-bold text-white mb-2">Access Blocked</h1>
        <p className="text-gray-400 mb-6">
          {reason || 'You have been blocked from this karaoke session.'}
        </p>

        <div className="bg-matte-black rounded-xl p-4 mb-6">
          <p className="text-sm text-gray-500">
            If you believe this is a mistake, please contact the host to have your access restored.
          </p>
        </div>

        <button
          onClick={() => window.location.reload()}
          className="w-full py-3 bg-matte-black text-white font-semibold rounded-xl hover:bg-opacity-80 transition-colors"
        >
          Try Again
        </button>
      </div>
    </div>
  );
}

function AFKModal({ onReturn }: { onReturn: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Dimmed backdrop */}
      <div className="absolute inset-0 bg-black/80 backdrop-blur-sm" />

      {/* Modal content */}
      <div className="relative bg-matte-gray rounded-2xl p-8 max-w-sm w-full mx-4 text-center animate-fade-in">
        {/* Clock icon */}
        <div className="w-20 h-20 mx-auto mb-6 rounded-full bg-orange-500/20 flex items-center justify-center">
          <svg className="w-10 h-10 text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>

        <h1 className="text-2xl font-bold text-white mb-2">You're Away</h1>
        <p className="text-gray-400 mb-6">
          Your songs have been moved to the end of the queue.
        </p>

        <button
          onClick={onReturn}
          className="w-full py-4 bg-yellow-neon text-matte-black font-bold text-lg rounded-xl hover:bg-yellow-400 transition-colors active:scale-95"
        >
          I'm Back!
        </button>

        <p className="text-gray-500 text-sm mt-4">
          Tap the button above to rejoin the queue
        </p>
      </div>
    </div>
  );
}

export function Home() {
  // Initialize WebSocket connection
  useWebSocket();

  const isBlocked = useRoomStore((state) => state.isBlocked);
  const blockReason = useRoomStore((state) => state.blockReason);
  const session = useRoomStore((state) => state.session);
  const [showSettings, setShowSettings] = useState(false);

  const handleReturnFromAFK = () => {
    wsService.setAFK(false);
  };

  // Show blocked page if user is blocked
  if (isBlocked) {
    return <BlockedPage reason={blockReason} />;
  }

  return (
    <div className="min-h-screen flex flex-col">
      <Header onOpenSettings={() => setShowSettings(true)} />
      <StatusBar />

      <main className="flex-1 p-4 pb-24 space-y-4 max-w-lg mx-auto w-full">
        {/* Now Playing */}
        <section className="aerial-enter">
          <NowPlaying />
        </section>
      </main>

      {/* Search - shows button when closed, fullscreen when open */}
      <Search />

      {/* User Settings Modal */}
      <UserSettings isOpen={showSettings} onClose={() => setShowSettings(false)} />

      {/* AFK Modal - shown when user is away */}
      {session?.is_afk && <AFKModal onReturn={handleReturnFromAFK} />}
    </div>
  );
}
