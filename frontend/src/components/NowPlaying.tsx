import { useRoomStore, selectCurrentSong, selectIsPlaying } from '../stores/roomStore';
import { wsService } from '../services/websocket';

function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

export function NowPlaying() {
  const currentSong = useRoomStore(selectCurrentSong);
  const isPlaying = useRoomStore(selectIsPlaying);
  const player = useRoomStore((state) => state.player);

  if (!currentSong) {
    return (
      <div className="bg-matte-gray rounded-2xl p-6 text-center">
        <div className="text-6xl mb-4">🎤</div>
        <h2 className="text-xl font-semibold text-gray-300">Ready to Sing</h2>
        <p className="text-gray-500 mt-2">Add a song to get started</p>
      </div>
    );
  }

  const progress = player.duration > 0
    ? (player.position / player.duration) * 100
    : 0;

  const handleSeek = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const percentage = x / rect.width;
    const newPosition = percentage * player.duration;
    wsService.seek(newPosition);
  };

  return (
    <div className="bg-matte-gray rounded-2xl overflow-hidden">
      {/* Album art / Video thumbnail */}
      <div className="relative aspect-video bg-matte-black">
        {currentSong.thumbnail_url && (
          <img
            src={currentSong.thumbnail_url}
            alt={currentSong.title}
            className="w-full h-full object-cover"
          />
        )}

        {/* Overlay gradient */}
        <div className="absolute inset-0 bg-gradient-to-t from-matte-gray via-transparent to-transparent" />

        {/* BGM indicator */}
        {player.bgm_active && (
          <div className="absolute top-4 left-4 bg-yellow-neon/20 text-yellow-neon px-3 py-1 rounded-full text-xs font-medium">
            Background Music
          </div>
        )}
      </div>

      {/* Song info */}
      <div className="p-4 -mt-16 relative z-10">
        <h2 className="text-xl font-bold text-white truncate">
          {currentSong.title}
        </h2>
        <p className="text-gray-400 truncate">{currentSong.artist}</p>

        {/* Progress bar */}
        <div
          className="mt-4 h-1 bg-matte-black rounded-full cursor-pointer"
          onClick={handleSeek}
        >
          <div
            className="h-full bg-yellow-neon rounded-full transition-all duration-100"
            style={{ width: `${progress}%` }}
          />
        </div>

        {/* Time */}
        <div className="flex justify-between text-xs text-gray-500 mt-1">
          <span>{formatTime(player.position)}</span>
          <span>{formatTime(player.duration)}</span>
        </div>

        {/* Controls */}
        <div className="flex items-center justify-center gap-6 mt-4">
          <button
            onClick={() => wsService.seek(player.position - 10)}
            className="text-gray-400 hover:text-white transition-colors"
          >
            <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
              <path d="M11 18V6l-8.5 6 8.5 6zm.5-6l8.5 6V6l-8.5 6z" />
            </svg>
          </button>

          <button
            onClick={() => (isPlaying ? wsService.pause() : wsService.play())}
            className="w-16 h-16 rounded-full bg-yellow-neon text-indigo-deep flex items-center justify-center hover:scale-105 transition-transform"
          >
            {isPlaying ? (
              <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
                <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
              </svg>
            ) : (
              <svg className="w-8 h-8 ml-1" fill="currentColor" viewBox="0 0 24 24">
                <path d="M8 5v14l11-7z" />
              </svg>
            )}
          </button>

          <button
            onClick={() => wsService.skip()}
            className="text-gray-400 hover:text-white transition-colors"
          >
            <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
              <path d="M4 18l8.5-6L4 6v12zm9-12v12l8.5-6L13 6z" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
