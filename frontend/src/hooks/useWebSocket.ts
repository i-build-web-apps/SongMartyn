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
      store.setSession(payload.session);
      store.updateState(payload.room_state);
      console.log('Session restored:', payload.session.display_name);
    });

    const unsubState = wsService.on('state_update', (state) => {
      store.updateState(state);
    });

    const unsubError = wsService.on('error', (payload) => {
      console.error('Server error:', payload.message);
    });

    // Connect
    wsService.connect();

    // Cleanup
    return () => {
      unsubWelcome();
      unsubState();
      unsubError();
      wsService.disconnect();
    };
  }, []);
}
