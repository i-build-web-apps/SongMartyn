import { useEffect } from 'react';
import { wsService } from '../services/websocket';
import { useRoomStore } from '../stores/roomStore';

// Module-level flag — survives React StrictMode cleanup/remount cycles
let wsInitialized = false;

// Track the last song ID we notified about to avoid repeats
let lastNotifiedNextUpId: string | null = null;

function checkNextUpNotification(store: ReturnType<typeof useRoomStore.getState>) {
  const { queue, session, playback } = store;
  if (!session || !queue.songs.length) return;

  // Find the next song after current position
  const nextSong = queue.songs[queue.position + 1];
  if (!nextSong) return;

  // Only notify if it's the current user's song
  if (nextSong.added_by !== session.martyn_key) {
    lastNotifiedNextUpId = null;
    return;
  }

  // Don't re-notify for the same song
  if (lastNotifiedNextUpId === nextSong.id) return;
  lastNotifiedNextUpId = nextSong.id;

  // Singer mode: more actionable notification
  const isSingerMode = playback?.mode === 'singer';
  const inAppMessage = isSingerMode
    ? `You're up next! Tap "Start Your Song" when ready`
    : `You're up next! "${nextSong.title}" is coming up`;
  const browserTitle = isSingerMode
    ? 'Your turn! Tap to start'
    : 'You\'re up next! 🎤';
  const browserBody = isSingerMode
    ? `"${nextSong.title}" — tap Start when you're ready`
    : `"${nextSong.title}" by ${nextSong.artist}`;

  // Show in-app notification
  store.addNotification('info', inAppMessage);

  // Fire browser notification if permitted
  if ('Notification' in window && Notification.permission === 'granted') {
    try {
      new Notification(browserTitle, {
        body: browserBody,
        icon: nextSong.thumbnail_url || '/favicon.ico',
        tag: 'next-up', // Replaces previous next-up notification
      });
    } catch {
      // Browser notification failed (e.g. mobile Safari) — in-app is enough
    }
  }
}

export function useWebSocket() {
  useEffect(() => {
    if (wsInitialized) return;
    wsInitialized = true;

    const store = useRoomStore.getState();
    store.setConnecting(true);

    // Setup message handlers
    const unsubWelcome = wsService.on('welcome', (payload) => {
      store.setConnected(true);
      store.setConnecting(false);
      store.setBlocked(false);
      store.setSession(payload.session);
      store.updateState(payload.room_state);
      store.addNotification('success', `Welcome, ${payload.session.display_name}!`);
      console.log('Session restored:', payload.session.display_name);
    });

    const unsubState = wsService.on('state_update', (state) => {
      const currentStore = useRoomStore.getState();
      currentStore.updateState(state);
      // Check if user's song just moved to "next up" position
      checkNextUpNotification(useRoomStore.getState());
    });

    const unsubError = wsService.on('error', (payload) => {
      console.error('Server error:', payload.message);
      store.addNotification('error', payload.message || 'An error occurred');
    });

    const unsubKicked = wsService.on('kicked', (payload) => {
      console.log('You have been kicked/blocked:', payload.reason);
      store.setConnected(false);
      // Check if this is a block message
      if (payload.reason.includes('blocked')) {
        store.setBlocked(true, payload.reason);
      }
    });

    // Connect
    wsService.connect();

    // Cleanup
    return () => {
      unsubWelcome();
      unsubState();
      unsubError();
      unsubKicked();
      wsService.disconnect();
    };
  }, []);
}
