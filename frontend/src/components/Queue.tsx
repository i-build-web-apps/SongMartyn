import { useRoomStore, selectQueue, selectQueuePosition, selectSession, selectActiveSessions } from '../stores/roomStore';
import { wsService } from '../services/websocket';
import { buildAvatarUrl } from './AvatarCreator';
import type { Song, Session, AvatarConfig } from '../types';

function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

// Helper to get session info from martyn_key
function getSingerInfo(addedBy: string, sessions: Session[]): { name: string; avatarConfig?: AvatarConfig } {
  const session = sessions.find(s => s.martyn_key === addedBy);
  return {
    name: session?.display_name || 'Unknown Singer',
    avatarConfig: session?.avatar_config,
  };
}

// Small avatar component for queue items
function SingerAvatar({ config, size = 20 }: { config?: AvatarConfig; size?: number }) {
  if (!config) {
    return (
      <div
        className="rounded-full bg-gray-600 flex items-center justify-center"
        style={{ width: size, height: size }}
      >
        <svg className="w-3 h-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
    );
  }

  const avatarUrl = buildAvatarUrl(config);
  return (
    <img
      src={avatarUrl}
      alt="Singer"
      className="rounded-full"
      style={{ width: size, height: size }}
    />
  );
}

interface QueueItemProps {
  song: Song;
  index: number;
  isActive: boolean;
  isPast: boolean;
  singerName: string;
  singerAvatar?: AvatarConfig;
  canRemove: boolean;
}

function QueueItem({ song, index, isActive, isPast, singerName, singerAvatar, canRemove }: QueueItemProps) {
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
        {/* Singer name with avatar */}
        <p className="text-yellow-neon/80 text-xs truncate flex items-center gap-1 mt-0.5">
          <SingerAvatar config={singerAvatar} size={16} />
          {singerName}
        </p>
      </div>

      {/* Duration */}
      <span className="text-gray-500 text-sm">
        {formatDuration(song.duration)}
      </span>

      {/* Remove button - only show if user can remove (their own song and not past) */}
      {!isPast && canRemove && (
        <button
          onClick={handleRemove}
          className="p-2 text-gray-500 hover:text-red-400 transition-colors"
          title="Remove from queue"
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
  const session = useRoomStore(selectSession);
  const sessions = useRoomStore(selectActiveSessions);

  // Only show current song and upcoming songs (no history)
  const upcomingSongs = songs.filter((_, index) => index >= position);

  if (upcomingSongs.length === 0) {
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
          {upcomingSongs.length} song{upcomingSongs.length !== 1 ? 's' : ''}
        </span>
      </div>

      <div className="space-y-2 max-h-80 overflow-y-auto">
        {upcomingSongs.map((song, displayIndex) => {
          const actualIndex = position + displayIndex;
          const singerInfo = getSingerInfo(song.added_by, sessions);
          const isOwnSong = session?.martyn_key === song.added_by;

          return (
            <QueueItem
              key={song.id}
              song={song}
              index={actualIndex}
              isActive={displayIndex === 0}
              isPast={false}
              singerName={singerInfo.name}
              singerAvatar={singerInfo.avatarConfig}
              canRemove={isOwnSong}
            />
          );
        })}
      </div>
    </div>
  );
}
