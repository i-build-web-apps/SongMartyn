import { useState, useEffect } from 'react';
import { useRoomStore } from '../stores/roomStore';
import { wsService } from '../services/websocket';

// Pixel avatar definitions - each is an 8x8 grid represented as a hex color array
const PIXEL_AVATARS = [
  { id: 'singer-red', name: 'Red Rocker', colors: ['#ef4444', '#dc2626', '#fbbf24', '#f59e0b'] },
  { id: 'singer-blue', name: 'Blues Brother', colors: ['#3b82f6', '#2563eb', '#60a5fa', '#93c5fd'] },
  { id: 'singer-purple', name: 'Purple Prince', colors: ['#8b5cf6', '#7c3aed', '#a78bfa', '#c4b5fd'] },
  { id: 'singer-green', name: 'Green Machine', colors: ['#22c55e', '#16a34a', '#4ade80', '#86efac'] },
  { id: 'singer-orange', name: 'Orange Star', colors: ['#f97316', '#ea580c', '#fb923c', '#fdba74'] },
  { id: 'singer-pink', name: 'Pink Diva', colors: ['#ec4899', '#db2777', '#f472b6', '#f9a8d4'] },
  { id: 'singer-cyan', name: 'Cyan Crooner', colors: ['#06b6d4', '#0891b2', '#22d3ee', '#67e8f9'] },
  { id: 'singer-yellow', name: 'Golden Voice', colors: ['#eab308', '#ca8a04', '#facc15', '#fde047'] },
];

// Generate a pixel avatar SVG
function PixelAvatar({ colors, size = 48 }: { colors: string[]; size?: number }) {
  // 8x8 pixel grid pattern for a singing character
  const pattern = [
    [0, 0, 1, 1, 1, 1, 0, 0],
    [0, 1, 2, 2, 2, 2, 1, 0],
    [1, 2, 3, 2, 2, 3, 2, 1],
    [1, 2, 2, 2, 2, 2, 2, 1],
    [1, 2, 2, 0, 0, 2, 2, 1],
    [0, 1, 2, 2, 2, 2, 1, 0],
    [0, 0, 1, 0, 0, 1, 0, 0],
    [0, 1, 1, 0, 0, 1, 1, 0],
  ];

  const pixelSize = size / 8;

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      {pattern.map((row, y) =>
        row.map((colorIdx, x) => {
          if (colorIdx === 0) return null;
          return (
            <rect
              key={`${x}-${y}`}
              x={x * pixelSize}
              y={y * pixelSize}
              width={pixelSize}
              height={pixelSize}
              fill={colors[colorIdx - 1] || colors[0]}
            />
          );
        })
      )}
    </svg>
  );
}

interface UserSettingsProps {
  isOpen: boolean;
  onClose: () => void;
}

export function UserSettings({ isOpen, onClose }: UserSettingsProps) {
  const session = useRoomStore((state) => state.session);
  const [displayName, setDisplayName] = useState('');
  const [selectedAvatar, setSelectedAvatar] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (session) {
      setDisplayName(session.display_name);
      setSelectedAvatar(session.avatar_id || null);
    }
  }, [session]);

  const handleSave = async () => {
    if (!displayName.trim()) return;

    setIsSaving(true);

    // Send display name update via WebSocket
    wsService.setDisplayName(displayName.trim(), selectedAvatar || undefined);

    // Give the server a moment to process
    await new Promise(resolve => setTimeout(resolve, 300));
    setIsSaving(false);
    onClose();
  };

  // Handle escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative bg-matte-gray rounded-2xl p-6 w-full max-w-md animate-slide-up">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-bold text-white">Your Profile</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Avatar Selection */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-400 mb-3">
            Choose Your Avatar
          </label>
          <div className="grid grid-cols-4 gap-3">
            {PIXEL_AVATARS.map((avatar) => (
              <button
                key={avatar.id}
                onClick={() => setSelectedAvatar(avatar.id)}
                className={`p-2 rounded-xl transition-all ${
                  selectedAvatar === avatar.id
                    ? 'bg-yellow-neon/20 ring-2 ring-yellow-neon scale-110'
                    : 'bg-matte-black hover:bg-matte-black/70'
                }`}
                title={avatar.name}
              >
                <PixelAvatar colors={avatar.colors} size={48} />
              </button>
            ))}
          </div>
        </div>

        {/* Display Name */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-400 mb-2">
            Display Name
          </label>
          <input
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="Enter your singer name..."
            maxLength={32}
            className="w-full px-4 py-3 bg-matte-black rounded-xl text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-yellow-neon"
          />
          <p className="text-xs text-gray-500 mt-2">
            This name will be shown when you add songs to the queue
          </p>
        </div>

        {/* Action Buttons */}
        <div className="flex gap-3">
          <button
            onClick={onClose}
            className="flex-1 py-3 bg-matte-black text-gray-400 font-semibold rounded-xl hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving || !displayName.trim()}
            className="flex-1 py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSaving ? 'Saving...' : 'Save Changes'}
          </button>
        </div>

        {/* Current Session Info */}
        <div className="mt-6 pt-4 border-t border-white/10">
          <p className="text-xs text-gray-500 text-center">
            Session ID: {session?.martyn_key?.slice(0, 8) || 'Unknown'}
          </p>
        </div>
      </div>
    </div>
  );
}

export { PIXEL_AVATARS, PixelAvatar };
