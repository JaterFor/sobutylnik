import { useEffect, useRef } from 'react';
import { WS_BASE_URL } from '../api/config';
import { getAccessToken } from '../storage/tokens';
import { ChatMessage } from '../api/types';

interface Handlers {
  onNewMessage: (msg: ChatMessage) => void;
  onMessagesRead: (messageIds: string[]) => void;
}

// Opens one WebSocket per screen for live chat delivery (see chat.Hub on the
// backend: in-process, keyed by user ID across all the user's matches).
export function useChatSocket(handlers: Handlers) {
  const handlersRef = useRef(handlers);
  handlersRef.current = handlers;

  useEffect(() => {
    let socket: WebSocket | null = null;
    let cancelled = false;

    (async () => {
      const token = await getAccessToken();
      if (!token || cancelled) return;

      socket = new WebSocket(`${WS_BASE_URL}/chat/ws?token=${encodeURIComponent(token)}`);
      socket.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data);
          if (parsed.type === 'new_message') {
            handlersRef.current.onNewMessage(parsed.payload as ChatMessage);
          } else if (parsed.type === 'messages_read') {
            handlersRef.current.onMessagesRead(parsed.payload.message_ids as string[]);
          }
        } catch {
          // ignore malformed frames
        }
      };
    })();

    return () => {
      cancelled = true;
      socket?.close();
    };
  }, []);
}
