import type {
  MessageType,
  WebSocketMessage,
  WelcomePayload,
  RoomState,
  VocalAssistLevel,
  SearchResult,
  ClientInfo,
  AvatarConfig,
} from '../types';

const MARTYN_KEY_STORAGE = 'songmartyn_key';
const RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 30000;

type MessageHandler = {
  welcome: (payload: WelcomePayload) => void;
  state_update: (payload: RoomState) => void;
  search_result: (payload: SearchResult[]) => void;
  error: (payload: { message: string }) => void;
  client_list: (payload: ClientInfo[]) => void;
  kicked: (payload: { reason: string }) => void;
};

class WebSocketService {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectDelay = RECONNECT_DELAY;
  private handlers: Partial<MessageHandler> = {};
  private isConnecting = false;
  private displayName: string = '';
  private wasKicked = false; // Prevent reconnection after being kicked/blocked

  constructor() {
    // Default to current host for WebSocket
    // Always use wss:// since server always runs HTTPS
    const protocol = 'wss:';
    const host = import.meta.env.DEV ? 'localhost:8443' : window.location.host;
    this.url = `${protocol}//${host}/ws`;
  }

  // Get stored MartynKey for session persistence
  private getMartynKey(): string | null {
    return localStorage.getItem(MARTYN_KEY_STORAGE);
  }

  // Store MartynKey for session persistence
  private setMartynKey(key: string): void {
    localStorage.setItem(MARTYN_KEY_STORAGE, key);
  }

  // Connect to WebSocket server
  connect(displayName?: string): void {
    if (this.isConnecting || this.ws?.readyState === WebSocket.OPEN) {
      console.log('[WS DEBUG] Connect skipped - already connecting or connected');
      return;
    }

    this.isConnecting = true;
    this.displayName = displayName || '';

    console.log(`[WS DEBUG] Connecting to ${this.url}...`);

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        console.log('[WS DEBUG] WebSocket connected successfully');
        this.isConnecting = false;
        this.reconnectDelay = RECONNECT_DELAY;

        // The Martyn Handshake - send stored key for session resumption
        const martynKey = this.getMartynKey() || '';
        console.log(`[WS DEBUG] Sending handshake (key: ${martynKey ? 'exists' : 'new'})`);
        this.send('handshake', {
          martyn_key: martynKey,
          display_name: this.displayName,
        });
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          console.log(`[WS DEBUG] Received message: ${message.type}`);
          this.handleMessage(message);
        } catch (err) {
          console.error('[WS ERROR] Failed to parse message:', err, event.data);
        }
      };

      this.ws.onclose = (event) => {
        console.log(`[WS DEBUG] WebSocket closed - code: ${event.code}, reason: "${event.reason}", wasClean: ${event.wasClean}`);
        this.isConnecting = false;
        this.scheduleReconnect();
      };

      this.ws.onerror = (error) => {
        console.error('[WS ERROR] WebSocket error:', error);
        this.isConnecting = false;
      };
    } catch (err) {
      console.error('[WS ERROR] Failed to create WebSocket:', err);
      this.isConnecting = false;
      this.scheduleReconnect();
    }
  }

  // Schedule reconnection with exponential backoff
  private scheduleReconnect(): void {
    // Don't reconnect if user was kicked/blocked
    if (this.wasKicked) {
      console.log('[WS DEBUG] Not reconnecting - user was kicked/blocked');
      return;
    }
    setTimeout(() => {
      console.log(`Reconnecting in ${this.reconnectDelay}ms...`);
      this.connect(this.displayName);
      this.reconnectDelay = Math.min(
        this.reconnectDelay * 2,
        MAX_RECONNECT_DELAY
      );
    }, this.reconnectDelay);
  }

  // Handle incoming messages
  private handleMessage(message: WebSocketMessage): void {
    switch (message.type) {
      case 'welcome': {
        const payload = message.payload as WelcomePayload;
        // Store MartynKey for session persistence
        this.setMartynKey(payload.session.martyn_key);
        this.handlers.welcome?.(payload);
        break;
      }
      case 'state_update':
        this.handlers.state_update?.(message.payload as RoomState);
        break;
      case 'search_result':
        this.handlers.search_result?.(message.payload as SearchResult[]);
        break;
      case 'error':
        this.handlers.error?.(message.payload as { message: string });
        break;
      case 'client_list':
        this.handlers.client_list?.(message.payload as ClientInfo[]);
        break;
      case 'kicked':
        this.wasKicked = true; // Prevent auto-reconnection
        this.handlers.kicked?.(message.payload as { reason: string });
        // Don't clear MartynKey - keep identity for potential unblock
        break;
    }
  }

  // Send a message to the server
  private send<T>(type: MessageType, payload: T): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, payload }));
    }
  }

  // Register message handlers
  on<K extends keyof MessageHandler>(
    type: K,
    handler: MessageHandler[K]
  ): () => void {
    this.handlers[type] = handler as MessageHandler[K];
    return () => {
      delete this.handlers[type];
    };
  }

  // API Methods
  search(query: string): void {
    this.send('search', query);
  }

  queueAdd(songId: string, vocalAssist?: string): void {
    this.send('queue_add', { song_id: songId, vocal_assist: vocalAssist || 'OFF' });
  }

  queueRemove(songId: string): void {
    this.send('queue_remove', songId);
  }

  queueMove(fromIndex: number, toIndex: number): void {
    this.send('queue_move', { from: fromIndex, to: toIndex });
  }

  queueClear(): void {
    this.send('queue_clear', null);
  }

  queueShuffle(): void {
    this.send('queue_shuffle', null);
  }

  queueRequeue(songId: string, martynKey: string): void {
    this.send('queue_requeue', { song_id: songId, martyn_key: martynKey });
  }

  setAFK(isAFK: boolean): void {
    this.send('set_afk', isAFK);
  }

  play(): void {
    this.send('play', null);
  }

  pause(): void {
    this.send('pause', null);
  }

  skip(): void {
    this.send('skip', null);
  }

  seek(position: number): void {
    this.send('seek', position);
  }

  setVocalAssist(level: VocalAssistLevel): void {
    this.send('vocal_assist', level);
  }

  setVolume(volume: number): void {
    this.send('volume', volume);
  }

  setAutoplay(enabled: boolean): void {
    this.send('autoplay', enabled);
  }

  setDisplayName(name: string, avatarId?: string, avatarConfig?: AvatarConfig): void {
    this.send('set_display_name', {
      display_name: name,
      avatar_id: avatarId || '',
      avatar_config: avatarConfig,
    });
  }

  setAvatarConfig(config: AvatarConfig): void {
    this.send('set_display_name', {
      display_name: '',  // Empty means keep current name
      avatar_config: config,
    });
  }

  // Admin methods
  adminSetAdmin(martynKey: string, isAdmin: boolean): void {
    this.send('admin_set_admin', { martyn_key: martynKey, is_admin: isAdmin });
  }

  adminKick(martynKey: string, reason?: string): void {
    this.send('admin_kick', { martyn_key: martynKey, reason: reason || '' });
  }

  adminBlock(martynKey: string, durationMinutes: number, reason?: string): void {
    this.send('admin_block', { martyn_key: martynKey, duration: durationMinutes, reason: reason || '' });
  }

  adminUnblock(martynKey: string): void {
    this.send('admin_unblock', { martyn_key: martynKey });
  }

  adminSetAFK(martynKey: string, isAFK: boolean): void {
    this.send('admin_set_afk', { martyn_key: martynKey, is_afk: isAFK });
  }

  adminPlayNext(): void {
    this.send('admin_play_next', null);
  }

  adminStop(): void {
    this.send('admin_stop', null);
  }

  // Disconnect
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  // Check connection status
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// Singleton instance
export const wsService = new WebSocketService();
