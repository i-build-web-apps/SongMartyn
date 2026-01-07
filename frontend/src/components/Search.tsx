import { useEffect, useRef, useState } from 'react';
import { useSearchStore, getFeatures } from '../stores/searchStore';
import { useRoomStore, selectVocalAssist, selectFavorites } from '../stores/roomStore';
import type { LibrarySong, VocalAssistLevel } from '../types';
import { VOCAL_LABELS } from '../types';

function formatDuration(seconds: number): string {
  if (!seconds) return '';
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

function SongCard({ song, onSelect, isFavorited, onToggleFavorite }: {
  song: LibrarySong;
  onSelect: () => void;
  isFavorited?: boolean;
  onToggleFavorite?: () => void;
}) {
  return (
    <div className="w-full flex items-center gap-3 p-3 bg-matte-black/50 rounded-xl hover:bg-matte-black transition-colors">
      {/* Main clickable area */}
      <button
        onClick={onSelect}
        className="flex items-center gap-3 flex-1 min-w-0 text-left"
      >
        {/* Thumbnail or placeholder */}
        <div className="w-12 h-12 bg-matte-gray rounded-lg flex-shrink-0 flex items-center justify-center overflow-hidden">
          {song.thumbnail_url ? (
            <img src={song.thumbnail_url} alt="" className="w-full h-full object-cover" />
          ) : (
            <svg className="w-6 h-6 text-gray-500" fill="currentColor" viewBox="0 0 20 20">
              <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.37 4.37 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
            </svg>
          )}
        </div>

        {/* Song info */}
        <div className="flex-1 min-w-0">
          <h4 className="text-white font-medium truncate">{song.title}</h4>
          <p className="text-gray-400 text-sm truncate">{song.artist || 'Unknown Artist'}</p>
        </div>

        {/* Duration */}
        {song.duration > 0 && (
          <span className="text-gray-500 text-sm flex-shrink-0">{formatDuration(song.duration)}</span>
        )}
      </button>

      {/* Favorite button */}
      {onToggleFavorite && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onToggleFavorite();
          }}
          className={`p-2 rounded-lg transition-colors flex-shrink-0 ${
            isFavorited
              ? 'text-red-500 hover:text-red-400'
              : 'text-gray-500 hover:text-red-400'
          }`}
          title={isFavorited ? 'Remove from favorites' : 'Add to favorites'}
        >
          <svg className="w-5 h-5" fill={isFavorited ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
          </svg>
        </button>
      )}

      {/* Add to queue icon */}
      <button
        onClick={onSelect}
        className="p-2 text-yellow-neon hover:text-yellow-300 transition-colors flex-shrink-0"
        title="Add to queue"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
      </button>
    </div>
  );
}

function ConfirmSongModal({ song, onConfirm, onCancel }: {
  song: LibrarySong;
  onConfirm: (vocalAssist: VocalAssistLevel) => void;
  onCancel: () => void;
}) {
  const roomVocalAssist = useRoomStore(selectVocalAssist);
  const [selectedLevel, setSelectedLevel] = useState<VocalAssistLevel>(roomVocalAssist);

  const levels: VocalAssistLevel[] = ['OFF', 'LOW', 'MED', 'HIGH'];

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60" onClick={onCancel} />
      <div className="relative bg-matte-gray rounded-2xl p-6 w-full max-w-sm animate-slide-up">
        {/* Song preview */}
        <div className="flex items-center gap-4 mb-6">
          <div className="w-16 h-16 bg-matte-black rounded-xl flex-shrink-0 flex items-center justify-center overflow-hidden">
            {song.thumbnail_url ? (
              <img src={song.thumbnail_url} alt="" className="w-full h-full object-cover" />
            ) : (
              <svg className="w-8 h-8 text-gray-500" fill="currentColor" viewBox="0 0 20 20">
                <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.37 4.37 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
              </svg>
            )}
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="text-white font-semibold text-lg truncate">{song.title}</h3>
            <p className="text-gray-400 truncate">{song.artist || 'Unknown Artist'}</p>
            {song.duration > 0 && (
              <p className="text-gray-500 text-sm">{formatDuration(song.duration)}</p>
            )}
          </div>
        </div>

        {/* Vocal Assist Selector */}
        <div className="mb-6">
          <label className="block text-gray-400 text-sm mb-3">Vocal Assist Level</label>
          <div className="grid grid-cols-4 gap-2">
            {levels.map((level) => (
              <button
                key={level}
                onClick={() => setSelectedLevel(level)}
                className={`py-2 px-2 rounded-lg text-sm font-medium transition-all ${
                  selectedLevel === level
                    ? 'bg-yellow-neon text-indigo-deep'
                    : 'bg-matte-black text-gray-400 hover:text-white'
                }`}
              >
                {VOCAL_LABELS[level]}
              </button>
            ))}
          </div>
          <p className="text-gray-500 text-xs mt-2 text-center">
            {selectedLevel === 'OFF' && 'Pure instrumental - no vocal guidance'}
            {selectedLevel === 'LOW' && 'Subtle pitch reference in background'}
            {selectedLevel === 'MED' && 'Light melody guide to follow along'}
            {selectedLevel === 'HIGH' && 'Full backing vocals for support'}
          </p>
        </div>

        {/* Buttons */}
        <div className="flex gap-3">
          <button
            onClick={onCancel}
            className="flex-1 py-3 bg-matte-black text-gray-400 font-semibold rounded-xl hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => onConfirm(selectedLevel)}
            className="flex-1 py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform"
          >
            Add to Queue
          </button>
        </div>
      </div>
    </div>
  );
}

function TabButton({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`py-2 px-4 text-sm font-medium rounded-lg transition-colors ${
        active
          ? 'bg-yellow-neon text-indigo-deep'
          : 'bg-matte-black text-gray-400 hover:text-white'
      }`}
    >
      {label}
    </button>
  );
}

export function Search() {
  const {
    query,
    results,
    popularSongs,
    historySongs,
    favoriteSongs,
    youtubeResults,
    activeTab,
    isLoading,
    isOpen,
    setQuery,
    setActiveTab,
    search,
    searchYouTube,
    closeSearch,
    addToQueue,
    openSearch,
    toggleFavorite,
  } = useSearchStore();

  const favorites = useRoomStore(selectFavorites);

  const inputRef = useRef<HTMLInputElement>(null);
  const searchTimeoutRef = useRef<number | null>(null);
  const [selectedSong, setSelectedSong] = useState<LibrarySong | null>(null);
  const [youtubeEnabled, setYoutubeEnabled] = useState(true);

  // Fetch feature flags on mount
  useEffect(() => {
    getFeatures().then(features => {
      setYoutubeEnabled(features.youtube_enabled);
    });
  }, []);

  // Focus input when opening
  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isOpen]);

  // Debounced search
  useEffect(() => {
    if (!isOpen) return;
    if (activeTab !== 'search' && activeTab !== 'youtube') return;

    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }

    if (query.trim()) {
      searchTimeoutRef.current = window.setTimeout(() => {
        if (activeTab === 'youtube') {
          searchYouTube();
        } else {
          search();
        }
      }, 300);
    }

    return () => {
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
    };
  }, [query, activeTab, search, searchYouTube, isOpen]);

  // Handle escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (selectedSong) {
          setSelectedSong(null);
        } else if (isOpen) {
          closeSearch();
        }
      }
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [isOpen, selectedSong, closeSearch]);

  // Get current song list based on active tab
  const currentSongs =
    activeTab === 'search' ? results :
    activeTab === 'popular' ? popularSongs :
    activeTab === 'history' ? historySongs :
    activeTab === 'favorites' ? favoriteSongs :
    activeTab === 'youtube' ? youtubeResults : [];

  const handleSelectSong = (song: LibrarySong) => {
    setSelectedSong(song);
  };

  const handleConfirmAdd = (vocalAssist: VocalAssistLevel) => {
    if (selectedSong) {
      addToQueue(selectedSong, vocalAssist);
      setSelectedSong(null);
      closeSearch();
    }
  };

  const getEmptyMessage = () => {
    if (activeTab === 'search' && !query) return 'Start typing to search your library...';
    if (activeTab === 'search' && query) return 'No songs found in library';
    if (activeTab === 'popular') return 'No popular songs yet';
    if (activeTab === 'history') return 'No song history yet';
    if (activeTab === 'favorites') return 'No favorites yet - tap the heart on any song to save it!';
    if (activeTab === 'youtube' && !query) return 'Search YouTube for karaoke tracks...';
    if (activeTab === 'youtube' && query) return 'No YouTube results found';
    return '';
  };

  // If not open, show the search button trigger
  if (!isOpen) {
    return (
      <div className="fixed bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-matte-black via-matte-black to-transparent pt-12">
        <button
          onClick={openSearch}
          className="w-full max-w-lg mx-auto py-4 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform flex items-center justify-center gap-2 shadow-lg shadow-yellow-neon/20"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          Search for Songs
        </button>
      </div>
    );
  }

  // Fullscreen search view
  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-matte-black">
      {/* Header with search input */}
      <div className="flex-shrink-0 bg-matte-gray border-b border-white/5">
        {/* Search input row */}
        <div className="flex items-center gap-3 p-4">
          <button
            onClick={closeSearch}
            className="p-2 text-gray-400 hover:text-white transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>

          <div className="flex-1 relative">
            <svg className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              ref={inputRef}
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={activeTab === 'youtube' ? 'Search YouTube for karaoke...' : 'Search by title or artist...'}
              className="w-full pl-12 pr-10 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon text-lg"
              autoFocus
            />
            {query && (
              <button
                onClick={() => setQuery('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-white"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 px-4 pb-4 overflow-x-auto">
          <TabButton label="Library" active={activeTab === 'search'} onClick={() => setActiveTab('search')} />
          {youtubeEnabled && (
            <TabButton label="YouTube" active={activeTab === 'youtube'} onClick={() => setActiveTab('youtube')} />
          )}
          <TabButton label="Popular" active={activeTab === 'popular'} onClick={() => setActiveTab('popular')} />
          <TabButton label="Favorites" active={activeTab === 'favorites'} onClick={() => setActiveTab('favorites')} />
          <TabButton label="My Songs" active={activeTab === 'history'} onClick={() => setActiveTab('history')} />
        </div>
      </div>

      {/* Results - takes remaining space */}
      <div className="flex-1 overflow-y-auto">
        <div className="p-4 space-y-2 max-w-2xl mx-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-20">
              <div className="w-10 h-10 border-3 border-yellow-neon border-t-transparent rounded-full animate-spin" />
            </div>
          ) : currentSongs.length === 0 ? (
            <div className="text-center py-20">
              <div className="text-6xl mb-4">🎤</div>
              <p className="text-gray-400 text-lg">{getEmptyMessage()}</p>
              {activeTab === 'youtube' && (
                <p className="text-gray-600 text-sm mt-2">
                  Tip: Add "karaoke" or "instrumental" to your search
                </p>
              )}
            </div>
          ) : (
            <>
              <p className="text-gray-500 text-sm mb-3">
                {currentSongs.length} {currentSongs.length === 1 ? 'result' : 'results'}
              </p>
              {currentSongs.map((song) => {
                // Show favorite button for library songs (not YouTube)
                const isLibrarySong = !String(song.id).startsWith('youtube:');
                const songId = String(song.id);
                return (
                  <SongCard
                    key={song.id}
                    song={song}
                    onSelect={() => handleSelectSong(song)}
                    isFavorited={isLibrarySong ? favorites.includes(songId) : undefined}
                    onToggleFavorite={isLibrarySong ? () => toggleFavorite(songId) : undefined}
                  />
                );
              })}
            </>
          )}
        </div>
      </div>

      {/* Confirmation modal */}
      {selectedSong && (
        <ConfirmSongModal
          song={selectedSong}
          onConfirm={handleConfirmAdd}
          onCancel={() => setSelectedSong(null)}
        />
      )}
    </div>
  );
}
