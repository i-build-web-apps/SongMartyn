import { useState } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import { Header } from '../components/Header';
import { NowPlaying } from '../components/NowPlaying';
import { VocalAssist } from '../components/VocalAssist';
import { Queue } from '../components/Queue';
import { Search } from '../components/Search';
import { UserSettings } from '../components/UserSettings';

export function Home() {
  // Initialize WebSocket connection
  useWebSocket();

  const [showSettings, setShowSettings] = useState(false);

  return (
    <div className="min-h-screen flex flex-col">
      <Header onOpenSettings={() => setShowSettings(true)} />

      <main className="flex-1 p-4 pb-24 space-y-4 max-w-lg mx-auto w-full">
        {/* Now Playing */}
        <section className="aerial-enter">
          <NowPlaying />
        </section>

        {/* Vocal Assist Control */}
        <section className="aerial-enter" style={{ animationDelay: '0.1s' }}>
          <VocalAssist />
        </section>

        {/* Queue */}
        <section className="aerial-enter" style={{ animationDelay: '0.2s' }}>
          <Queue />
        </section>
      </main>

      {/* Search - shows button when closed, fullscreen when open */}
      <Search />

      {/* User Settings Modal */}
      <UserSettings isOpen={showSettings} onClose={() => setShowSettings(false)} />
    </div>
  );
}
