import { useRoomStore, selectQueue, selectQueuePosition, selectSession, selectActiveSessions, selectIdle } from '../stores/roomStore';
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

// YouTube badge for YouTube-sourced songs
function YTBadge() {
  return (
    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold bg-red-500/20 text-red-400 leading-none">
      YT
    </span>
  );
}

// Download progress indicator for YouTube songs
function DownloadIndicator({ status, progress }: { status?: string; progress?: number }) {
  if (!status || status === 'ready') return null;

  if (status === 'failed') {
    return (
      <span className="text-red-400 text-xs">Download failed - will skip</span>
    );
  }

  if (status === 'pending') {
    return (
      <span className="text-yellow-neon/60 text-xs">Queued for download...</span>
    );
  }

  // downloading
  const pct = Math.round(progress || 0);
  return (
    <div className="flex items-center gap-2 mt-1">
      <div className="flex-1 h-1 bg-matte-black rounded-full overflow-hidden">
        <div
          className="h-full bg-yellow-neon rounded-full transition-all duration-300"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-yellow-neon/80 text-xs tabular-nums">{pct}%</span>
    </div>
  );
}

// Now Playing card — shown separately above the queue
function NowPlayingCard({ song, singerName, singerAvatar }: {
  song: Song;
  singerName: string;
  singerAvatar?: AvatarConfig;
}) {
  const isYouTube = song.id.startsWith('youtube:');

  return (
    <div className="bg-matte-gray rounded-2xl p-4 mb-3">
      <h3 className="text-xs text-yellow-neon uppercase tracking-wide mb-3 flex items-center gap-2">
        <div className="w-2.5 h-2.5 bg-yellow-neon rounded-full animate-pulse" />
        Now Playing
      </h3>
      <div className="flex items-center gap-3">
        {/* Thumbnail */}
        <div className="w-14 h-14 rounded-lg bg-matte-black overflow-hidden flex-shrink-0">
          {song.thumbnail_url && (
            <img src={song.thumbnail_url} alt={song.title} className="w-full h-full object-cover" />
          )}
        </div>

        {/* Song info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5">
            <h4 className="text-yellow-neon font-semibold truncate">{song.title}</h4>
            {isYouTube && <YTBadge />}
          </div>
          <p className="text-gray-400 text-sm truncate">{song.artist}</p>
          <p className="text-white/60 text-xs truncate flex items-center gap-1 mt-0.5">
            <SingerAvatar config={singerAvatar} size={16} />
            {singerName}
          </p>
        </div>

        {/* Duration */}
        <span className="text-gray-500 text-sm">
          {formatDuration(song.duration)}
        </span>
      </div>
    </div>
  );
}

interface QueueItemProps {
  song: Song;
  position: number;
  singerName: string;
  singerAvatar?: AvatarConfig;
  canRemove: boolean;
}

function QueueItem({ song, position, singerName, singerAvatar, canRemove }: QueueItemProps) {
  const handleRemove = () => {
    wsService.queueRemove(song.id);
  };

  const isYouTube = song.id.startsWith('youtube:');
  const isDownloading = isYouTube && song.download_status && song.download_status !== 'ready';

  return (
    <div
      className={`
        flex items-center gap-3 p-3 rounded-xl transition-all
        bg-matte-light/50 hover:bg-matte-light
        ${song.download_status === 'failed' ? 'opacity-60' : ''}
      `}
    >
      {/* Position number */}
      <div className="w-8 text-center">
        <span className="text-gray-500 text-sm">{position}</span>
      </div>

      {/* Thumbnail */}
      <div className="w-12 h-12 rounded-lg bg-matte-black overflow-hidden flex-shrink-0 relative">
        {song.thumbnail_url && (
          <img
            src={song.thumbnail_url}
            alt={song.title}
            className={`w-full h-full object-cover ${isDownloading ? 'opacity-60' : ''}`}
          />
        )}
        {/* Circular progress overlay on thumbnail while downloading */}
        {isDownloading && song.download_status === 'downloading' && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/40">
            <svg className="w-6 h-6" viewBox="0 0 24 24">
              <circle
                cx="12" cy="12" r="10"
                fill="none" stroke="currentColor" strokeWidth="2"
                className="text-gray-600"
              />
              <circle
                cx="12" cy="12" r="10"
                fill="none" stroke="currentColor" strokeWidth="2"
                className="text-yellow-neon"
                strokeDasharray={`${(song.download_progress || 0) * 0.628} 62.8`}
                strokeLinecap="round"
                transform="rotate(-90 12 12)"
              />
            </svg>
          </div>
        )}
      </div>

      {/* Song info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <h4 className="text-white font-medium truncate">{song.title}</h4>
          {isYouTube && <YTBadge />}
        </div>
        <p className="text-gray-500 text-sm truncate">{song.artist}</p>
        {/* Singer name with avatar */}
        <p className="text-yellow-neon/80 text-xs truncate flex items-center gap-1 mt-0.5">
          <SingerAvatar config={singerAvatar} size={16} />
          {singerName}
        </p>
        {/* Download progress bar */}
        {isYouTube && <DownloadIndicator status={song.download_status} progress={song.download_progress} />}
      </div>

      {/* Duration */}
      <span className="text-gray-500 text-sm">
        {formatDuration(song.duration)}
      </span>

      {/* Remove button */}
      {canRemove && (
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
  const idle = useRoomStore(selectIdle);
  const session = useRoomStore(selectSession);
  const sessions = useRoomStore(selectActiveSessions);

  // Current song (at position) — only show as "now playing" if not idle
  const currentSong = !idle && position >= 0 && position < songs.length ? songs[position] : null;

  // Upcoming songs — everything after the current position (or from position if idle)
  const upcomingStart = currentSong ? position + 1 : position;
  const upcomingSongs = songs.filter((_, index) => index >= upcomingStart);

  const noContent = !currentSong && upcomingSongs.length === 0;

  if (noContent) {
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
    <div>
      {/* Now Playing — separate from queue */}
      {currentSong && (() => {
        const singerInfo = getSingerInfo(currentSong.added_by, sessions);
        return (
          <NowPlayingCard
            song={currentSong}
            singerName={singerInfo.name}
            singerAvatar={singerInfo.avatarConfig}
          />
        );
      })()}

      {/* Up Next queue */}
      <div className="bg-matte-gray rounded-2xl p-4">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm text-gray-400 uppercase tracking-wide">
            Up Next
          </h3>
          <span className="text-xs text-gray-500">
            {upcomingSongs.length} song{upcomingSongs.length !== 1 ? 's' : ''}
          </span>
        </div>

        {upcomingSongs.length === 0 ? (
          <div className="text-center py-6">
            <p className="text-gray-600 text-sm">No more songs queued</p>
          </div>
        ) : (
          <div className="space-y-2 max-h-80 overflow-y-auto">
            {upcomingSongs.map((song, displayIndex) => {
              const singerInfo = getSingerInfo(song.added_by, sessions);
              const isOwnSong = session?.martyn_key === song.added_by;

              return (
                <QueueItem
                  key={song.id}
                  song={song}
                  position={displayIndex + 1}
                  singerName={singerInfo.name}
                  singerAvatar={singerInfo.avatarConfig}
                  canRemove={isOwnSong}
                />
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
