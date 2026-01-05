import { useEffect, useRef } from 'react';
import { wsService } from '../services/websocket';
import { useRoomStore } from '../stores/roomStore';

export function useWebSocket() {
  const isInitialized = useRef(false);

  useEffect(() => {
    if (isInitialized.current) return;
    isInitialized.current = true;

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
      store.updateState(state);
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
