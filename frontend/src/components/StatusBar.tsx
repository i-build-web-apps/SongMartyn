import { useEffect } from 'react';
import { useRoomStore, selectNotifications, type Notification } from '../stores/roomStore';

const NOTIFICATION_DURATION = 4000; // Auto-dismiss after 4 seconds

function NotificationItem({ notification, onDismiss }: {
  notification: Notification;
  onDismiss: () => void;
}) {
  useEffect(() => {
    const timer = setTimeout(onDismiss, NOTIFICATION_DURATION);
    return () => clearTimeout(timer);
  }, [notification.id, onDismiss]);

  const typeStyles = {
    success: 'bg-green-500/20 border-green-500/50 text-green-400',
    info: 'bg-blue-500/20 border-blue-500/50 text-blue-400',
    warning: 'bg-yellow-500/20 border-yellow-500/50 text-yellow-400',
    error: 'bg-red-500/20 border-red-500/50 text-red-400',
  };

  const icons = {
    success: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
      </svg>
    ),
    info: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    warning: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
    ),
    error: (
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
      </svg>
    ),
  };

  return (
    <div
      className={`flex items-center gap-2 px-3 py-2 rounded-lg border backdrop-blur-sm animate-slide-in ${typeStyles[notification.type]}`}
      onClick={onDismiss}
    >
      {icons[notification.type]}
      <span className="text-sm font-medium">{notification.message}</span>
    </div>
  );
}

export function StatusBar() {
  const notifications = useRoomStore(selectNotifications);
  const removeNotification = useRoomStore((state) => state.removeNotification);

  if (notifications.length === 0) {
    return null;
  }

  return (
    <div className="fixed top-16 left-0 right-0 z-40 pointer-events-none">
      <div className="max-w-lg mx-auto px-4 space-y-2 pointer-events-auto">
        {notifications.map((notification) => (
          <NotificationItem
            key={notification.id}
            notification={notification}
            onDismiss={() => removeNotification(notification.id)}
          />
        ))}
      </div>
    </div>
  );
}
