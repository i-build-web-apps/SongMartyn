import { useRoomStore, selectCurrentSong, selectIsPlaying, selectActiveSessions } from '../stores/roomStore';
import { buildAvatarUrl } from './AvatarCreator';
import type { Session, AvatarConfig } from '../types';

// Helper to get session info from martyn_key
function getSingerInfo(addedBy: string, sessions: Session[]): { name: string; avatarConfig?: AvatarConfig } {
  const session = sessions.find(s => s.martyn_key === addedBy);
  return {
    name: session?.display_name || 'Unknown Singer',
    avatarConfig: session?.avatar_config,
  };
}

// Avatar component for the singer
function SingerAvatar({ config, size = 48 }: { config?: AvatarConfig; size?: number }) {
  if (!config) {
    return (
      <div
        className="rounded-full bg-yellow-neon/20 flex items-center justify-center flex-shrink-0"
        style={{ width: size, height: size }}
      >
        <svg className="w-1/2 h-1/2 text-yellow-neon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
      className="rounded-full flex-shrink-0"
      style={{ width: size, height: size }}
    />
  );
}

// Animated audio bars
function AudioBars({ isPlaying, size = 'md' }: { isPlaying: boolean; size?: 'sm' | 'md' | 'lg' }) {
  const heights = size === 'lg' ? 'h-8' : size === 'md' ? 'h-5' : 'h-4';
  const widths = size === 'lg' ? 'w-1.5' : size === 'md' ? 'w-1' : 'w-0.5';
  const gaps = size === 'lg' ? 'gap-1' : 'gap-0.5';

  return (
    <div className={`flex items-end ${gaps} ${heights}`}>
      {[1, 2, 3, 4, 5].map((i) => (
        <div
          key={i}
          className={`${widths} bg-gradient-to-t from-yellow-neon to-green-400 rounded-full`}
          style={{
            height: isPlaying ? '20%' : '20%',
            animation: isPlaying ? `audioBar${i} 0.${3 + i}s ease-in-out infinite alternate` : 'none',
          }}
        />
      ))}
      <style>{`
        @keyframes audioBar1 { 0% { height: 20%; } 100% { height: 100%; } }
        @keyframes audioBar2 { 0% { height: 40%; } 100% { height: 70%; } }
        @keyframes audioBar3 { 0% { height: 60%; } 100% { height: 90%; } }
        @keyframes audioBar4 { 0% { height: 30%; } 100% { height: 80%; } }
        @keyframes audioBar5 { 0% { height: 50%; } 100% { height: 100%; } }
      `}</style>
    </div>
  );
}

export function NowPlaying() {
  const currentSong = useRoomStore(selectCurrentSong);
  const isPlaying = useRoomStore(selectIsPlaying);
  const sessions = useRoomStore(selectActiveSessions);
  const player = useRoomStore((state) => state.player);

  if (!currentSong) {
    return (
      <div className="bg-matte-gray rounded-2xl overflow-hidden">
        {/* Placeholder with logo background */}
        <div className="relative aspect-[16/9]">
          <img
            src="/logo.jpeg"
            alt="SongMartyn"
            className="w-full h-full object-cover"
          />
          {/* Dark gradient overlay for text readability */}
          <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/40 to-transparent" />

          {/* Centered content */}
          <div className="absolute inset-0 flex flex-col items-center justify-center text-center p-6">
            <h2 className="text-2xl font-bold text-white drop-shadow-lg">Ready to Sing</h2>
            <p className="text-gray-300 mt-2 drop-shadow-lg">Add a song to get started</p>
          </div>
        </div>
      </div>
    );
  }

  const singerInfo = getSingerInfo(currentSong.added_by, sessions);

  return (
    <div className="bg-matte-gray rounded-2xl overflow-hidden relative">
      {/* Animated gradient background */}
      <div className="absolute inset-0 bg-gradient-to-br from-purple-900/40 via-matte-black to-yellow-neon/10" />

      {/* Animated gradient orbs */}
      <div className={`absolute top-0 right-0 w-64 h-64 bg-yellow-neon/20 rounded-full blur-3xl ${isPlaying ? 'animate-pulse' : ''}`} />
      <div className={`absolute bottom-0 left-0 w-48 h-48 bg-purple-500/20 rounded-full blur-3xl ${isPlaying ? 'animate-pulse' : ''}`} style={{ animationDelay: '0.5s' }} />

      {/* Main content */}
      <div className="relative z-10 p-6">
        {/* Top row: Status badges */}
        <div className="flex items-center justify-between mb-4">
          {/* BGM indicator or spacer */}
          {player.bgm_active ? (
            <div className="bg-purple-500/30 backdrop-blur-sm text-purple-300 px-3 py-1.5 rounded-full text-sm font-medium border border-purple-500/30">
              Background Music
            </div>
          ) : (
            <div />
          )}

          {/* Playing/Paused indicator */}
          {isPlaying ? (
            <div className="flex items-center gap-2 bg-green-500/30 backdrop-blur-sm text-green-300 px-4 py-2 rounded-full text-sm font-semibold border border-green-500/30 shadow-lg shadow-green-500/20">
              <AudioBars isPlaying={true} size="sm" />
              Now Playing
            </div>
          ) : (
            <div className="flex items-center gap-2 bg-yellow-neon/20 backdrop-blur-sm text-yellow-neon px-4 py-2 rounded-full text-sm font-semibold border border-yellow-neon/30">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
              </svg>
              Paused
            </div>
          )}
        </div>

        {/* Center: Large singer avatar + Song info */}
        <div className="flex items-center gap-6">
          {/* Singer avatar - large and prominent */}
          <div className="relative flex-shrink-0">
            {/* Multi-layer glow effect behind avatar */}
            <div className={`absolute inset-0 rounded-full blur-2xl ${isPlaying ? 'animate-pulse' : ''}`}
              style={{
                background: 'linear-gradient(135deg, #DFFF00, #00ff88)',
                transform: 'scale(1.3)',
                opacity: 0.4
              }}
            />
            <div className="absolute inset-0 rounded-full blur-xl"
              style={{
                background: 'linear-gradient(135deg, #a855f7, #DFFF00)',
                transform: 'scale(1.2)',
                opacity: 0.3
              }}
            />
            {/* Avatar container with ring */}
            <div className="relative">
              <div className="absolute -inset-1 bg-gradient-to-r from-yellow-neon via-green-400 to-purple-500 rounded-full" />
              <div className="relative rounded-full overflow-hidden ring-4 ring-matte-black">
                <SingerAvatar config={singerInfo.avatarConfig} size={96} />
              </div>
            </div>
          </div>

          {/* Song and singer info */}
          <div className="flex-1 min-w-0">
            {/* Singer name - prominent */}
            <div className="flex items-center gap-2 mb-2">
              <span className="text-gray-400 text-sm uppercase tracking-wider">On Stage</span>
              {isPlaying && <AudioBars isPlaying={true} size="sm" />}
            </div>
            <p className="text-2xl font-bold text-white truncate mb-3">{singerInfo.name}</p>

            {/* Song info - below singer */}
            <div className="bg-white/5 backdrop-blur-sm rounded-lg px-4 py-3 border border-white/10">
              <p className="text-white font-semibold truncate text-lg">
                {currentSong.title}
              </p>
              <p className="text-gray-400 truncate">{currentSong.artist}</p>
            </div>
          </div>
        </div>

        {/* Bottom: Vocal assist badge */}
        <div className="mt-4 flex justify-end">
          <div className={`px-4 py-2 rounded-full text-sm font-medium backdrop-blur-sm ${
            currentSong.vocal_assist === 'OFF' ? 'bg-gray-700/50 text-gray-400 border border-gray-600/50' :
            currentSong.vocal_assist === 'LOW' ? 'bg-blue-500/20 text-blue-300 border border-blue-500/30' :
            currentSong.vocal_assist === 'MED' ? 'bg-purple-500/20 text-purple-300 border border-purple-500/30' :
            'bg-green-500/20 text-green-300 border border-green-500/30'
          }`}>
            {currentSong.vocal_assist === 'OFF' ? 'No vocal assist' :
             currentSong.vocal_assist === 'LOW' ? 'Pitch guide' :
             currentSong.vocal_assist === 'MED' ? 'Melody guide' :
             'Full vocals'}
          </div>
        </div>
      </div>
    </div>
  );
}
