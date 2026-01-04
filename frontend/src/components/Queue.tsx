import { useRoomStore, selectQueue, selectQueuePosition } from '../stores/roomStore';
import { wsService } from '../services/websocket';
import type { Song } from '../types';

function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

interface QueueItemProps {
  song: Song;
  index: number;
  isActive: boolean;
  isPast: boolean;
}

function QueueItem({ song, index, isActive, isPast }: QueueItemProps) {
  const handleRemove = () => {
    wsService.queueRemove(song.id);
  };

  return (
    <div
      className={`
        flex items-center gap-3 p-3 rounded-xl transition-all
        ${isActive ? 'bg-yellow-neon/10 border border-yellow-neon/30' : ''}
        ${isPast ? 'opacity-50' : ''}
        ${!isActive && !isPast ? 'bg-matte-light/50 hover:bg-matte-light' : ''}
      `}
    >
      {/* Index / Now Playing indicator */}
      <div className="w-8 text-center">
        {isActive ? (
          <div className="w-3 h-3 mx-auto bg-yellow-neon rounded-full animate-pulse" />
        ) : (
          <span className="text-gray-500 text-sm">{index + 1}</span>
        )}
      </div>

      {/* Thumbnail */}
      <div className="w-12 h-12 rounded-lg bg-matte-black overflow-hidden flex-shrink-0">
        {song.thumbnail_url && (
          <img
            src={song.thumbnail_url}
            alt={song.title}
            className="w-full h-full object-cover"
          />
        )}
      </div>

      {/* Song info */}
      <div className="flex-1 min-w-0">
        <h4 className="text-white font-medium truncate">{song.title}</h4>
        <p className="text-gray-500 text-sm truncate">{song.artist}</p>
      </div>

      {/* Duration */}
      <span className="text-gray-500 text-sm">
        {formatDuration(song.duration)}
      </span>

      {/* Remove button */}
      {!isPast && (
        <button
          onClick={handleRemove}
          className="p-2 text-gray-500 hover:text-red-400 transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  );
}

export function Queue() {
  const songs = useRoomStore(selectQueue);
  const position = useRoomStore(selectQueuePosition);

  if (songs.length === 0) {
    return (
      <div className="bg-matte-gray rounded-2xl p-6">
        <h3 className="text-sm text-gray-400 mb-4 uppercase tracking-wide">
          Up Next
        </h3>
        <div className="text-center py-8">
          <div className="text-4xl mb-2">📋</div>
          <p className="text-gray-500">Queue is empty</p>
          <p className="text-gray-600 text-sm mt-1">Search for songs to add them</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-matte-gray rounded-2xl p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm text-gray-400 uppercase tracking-wide">
          Up Next
        </h3>
        <span className="text-xs text-gray-500">
          {songs.length} song{songs.length !== 1 ? 's' : ''}
        </span>
      </div>

      <div className="space-y-2 max-h-80 overflow-y-auto">
        {songs.map((song, index) => (
          <QueueItem
            key={song.id}
            song={song}
            index={index}
            isActive={index === position}
            isPast={index < position}
          />
        ))}
      </div>
    </div>
  );
}
