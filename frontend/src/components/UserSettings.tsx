import { useState, useEffect } from 'react';
import { useRoomStore } from '../stores/roomStore';
import { wsService } from '../services/websocket';
import { AvatarCreator, type AvatarConfig } from './AvatarCreator';

interface UserSettingsProps {
  isOpen: boolean;
  onClose: () => void;
}

export function UserSettings({ isOpen, onClose }: UserSettingsProps) {
  const session = useRoomStore((state) => state.session);
  const [displayName, setDisplayName] = useState('');
  const [avatarConfig, setAvatarConfig] = useState<AvatarConfig | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [randomizeTrigger, setRandomizeTrigger] = useState(0);

  useEffect(() => {
    if (session) {
      setDisplayName(session.display_name);
      if (session.avatar_config) {
        setAvatarConfig(session.avatar_config);
      }
    }
  }, [session]);

  const handleSaveAvatar = (config: AvatarConfig) => {
    setAvatarConfig(config);
  };

  const handleSave = async () => {
    if (!displayName.trim()) return;

    setIsSaving(true);

    // Send display name and avatar config update via WebSocket
    wsService.setDisplayName(displayName.trim(), undefined, avatarConfig || undefined);

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
      <div className="relative bg-matte-gray rounded-2xl p-6 w-full max-w-md animate-slide-up max-h-[90vh] overflow-y-auto">
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

        {/* Avatar Creator */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-400 mb-3">
            Create Your Avatar
          </label>
          <AvatarCreator
            initialConfig={avatarConfig || undefined}
            onSave={handleSaveAvatar}
            size={140}
            showButtons={false}
            randomizeTrigger={randomizeTrigger}
          />
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
            onClick={() => setRandomizeTrigger(prev => prev + 1)}
            className="flex-1 py-3 bg-matte-black text-gray-400 font-semibold rounded-xl hover:text-white transition-colors flex items-center justify-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Random
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving || !displayName.trim()}
            className="flex-1 py-3 bg-yellow-neon text-indigo-deep font-semibold rounded-xl hover:scale-[1.02] transition-transform disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSaving ? 'Saving...' : 'Save'}
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
