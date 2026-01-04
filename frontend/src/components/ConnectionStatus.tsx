import { useRoomStore } from '../stores/roomStore';

export function ConnectionStatus() {
  const isConnected = useRoomStore((state) => state.isConnected);
  const isConnecting = useRoomStore((state) => state.isConnecting);
  const session = useRoomStore((state) => state.session);

  if (isConnecting) {
    return (
      <div className="flex items-center gap-2 text-yellow-neon">
        <div className="w-2 h-2 bg-yellow-neon rounded-full animate-pulse" />
        <span className="text-sm">Connecting...</span>
      </div>
    );
  }

  if (!isConnected) {
    return (
      <div className="flex items-center gap-2 text-red-400">
        <div className="w-2 h-2 bg-red-400 rounded-full" />
        <span className="text-sm">Disconnected</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 text-green-400">
      <div className="w-2 h-2 bg-green-400 rounded-full" />
      <span className="text-sm">{session?.display_name || 'Connected'}</span>
    </div>
  );
}
