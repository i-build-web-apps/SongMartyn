import { ConnectionStatus } from './ConnectionStatus';

export function Header() {
  return (
    <header className="flex items-center justify-between px-4 py-3 bg-matte-gray/50 backdrop-blur-sm border-b border-white/5">
      {/* Logo */}
      <div className="flex items-center gap-2">
        <img src="/logo.jpeg" alt="SongMartyn" className="w-10 h-10 rounded-lg object-cover" />
        <span className="text-xl font-bold text-white">
          Song<span className="text-yellow-neon">Martyn</span>
        </span>
      </div>

      {/* Connection status */}
      <ConnectionStatus />
    </header>
  );
}
