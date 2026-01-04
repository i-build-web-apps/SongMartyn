import { useWebSocket } from '../hooks/useWebSocket';
import { Header } from '../components/Header';
import { NowPlaying } from '../components/NowPlaying';
import { VocalAssist } from '../components/VocalAssist';
import { Queue } from '../components/Queue';
import { Search } from '../components/Search';
import { useSearchStore } from '../stores/searchStore';

export function Home() {
  // Initialize WebSocket connection
  useWebSocket();

  const openSearch = useSearchStore((state) => state.openSearch);

  return (
    <div className="min-h-screen flex flex-col">
      <Header />

      <main className="flex-1 p-4 space-y-4 max-w-lg mx-auto w-full">
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

      {/* Footer - Search button */}
      <footer className="p-4 bg-matte-gray/50 backdrop-blur-sm border-t border-white/5">
        <button
          onClick={openSearch}
          className="w-full py-4 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform flex items-center justify-center gap-2"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          Search for Songs
        </button>
      </footer>

      {/* Search modal */}
      <Search />
    </div>
  );
}
