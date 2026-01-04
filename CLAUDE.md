# SongMartyn Development Guidelines

This file contains important notes and conventions for AI assistants (Claude) working on this codebase.

## Project Overview

SongMartyn is a self-hosted karaoke party system with:
- **Backend**: Go with WebSocket, SQLite, mpv integration
- **Frontend**: React + TypeScript + Vite + Tailwind CSS + Zustand

## Important Conventions

### Admin Panel Help Modals

**CRITICAL**: Every complex admin feature MUST have an associated help modal.

When adding new admin features that involve:
- External services (APIs, integrations)
- Network/infrastructure configuration
- Security settings
- File system paths or configurations
- Any setting that requires external setup steps

You MUST:

1. **Add help content** to `frontend/src/components/HelpModal.tsx` in the `HELP_CONTENT` object:
```typescript
export const HELP_CONTENT = {
  // Add new topic here
  myNewFeature: {
    title: 'Feature Name',
    sections: [
      {
        heading: 'What is this?',
        content: `Explanation of what this feature does...`,
      },
      {
        heading: 'How to Set Up',
        content: `Step by step instructions...`,
        links: [
          { label: 'Documentation', url: 'https://...' },
        ],
      },
    ],
  },
};
```

2. **Add a HelpButton** next to the feature in the admin UI:
```tsx
import { HelpButton, useHelpModal, HelpModal } from '../components/HelpModal';

function MyComponent() {
  const { activeHelp, openHelp, closeHelp } = useHelpModal();

  return (
    <>
      <div className="flex items-center gap-2">
        <label>Feature Name</label>
        <HelpButton onClick={() => openHelp('myNewFeature')} />
      </div>
      {/* ... component content ... */}
      {activeHelp && (
        <HelpModal topic={activeHelp} isOpen={true} onClose={closeHelp} />
      )}
    </>
  );
}
```

### Current Help Topics

| Topic | Feature | Location |
|-------|---------|----------|
| `network` | Network interface selection | Network tab |
| `youtubeApi` | YouTube Data API setup | Settings > YouTube API Key |
| `certificates` | TLS/HTTPS certificates | Settings > Server Configuration |
| `adminPin` | Admin PIN & remote access | Settings > Admin PIN |
| `videoPlayer` | mpv player setup | Settings > Video Player Path |
| `library` | Song library management | Library tab |

### Future Features Requiring Help Modals

When implementing these roadmap items, remember to add help:
- Audio separation (Demucs/Spleeter) - explain installation, GPU requirements
- Multi-room support - explain networking implications
- Chromecast/AirPlay - explain discovery, pairing process
- Lyrics display - explain timing file formats

## Code Style

### Frontend
- Use functional components with hooks
- State management via Zustand stores
- Tailwind CSS for styling
- Keep components focused and composable

### Backend
- Standard Go project layout
- WebSocket for real-time communication
- SQLite for persistence
- Clean separation between packages

## Testing Checklist

Before committing admin panel changes:
- [ ] Help modal added for any new complex feature
- [ ] Help content includes "What is this?" section
- [ ] Help content includes setup/configuration steps
- [ ] External links are included where relevant
- [ ] HelpButton positioned next to the feature label
- [ ] HelpModal renders correctly on mobile

## File Locations

| Component | Path |
|-----------|------|
| Help system | `frontend/src/components/HelpModal.tsx` |
| Admin panel | `frontend/src/pages/Admin.tsx` |
| WebSocket service | `frontend/src/services/websocket.ts` |
| Backend main | `backend/cmd/songmartyn/main.go` |
| Admin auth | `backend/internal/admin/admin.go` |
