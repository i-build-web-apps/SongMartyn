export function Footer() {
  return (
    <footer className="fixed bottom-0 left-0 right-0 py-1 text-center text-xs text-white/70 bg-matte-black/50 backdrop-blur-sm z-10 pointer-events-none">
      <span className="font-mono">Build: {__BUILD_HASH__}</span>
    </footer>
  );
}
