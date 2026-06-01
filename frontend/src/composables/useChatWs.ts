import { ref, onBeforeUnmount } from 'vue';

export interface OnlineUser {
  username: string;
  role: string;
  ip: string;
  loginTime: string;
}

export interface ChatMessage {
  type: 'online' | 'chat' | 'system';
  username?: string;
  role?: string;
  content?: string;
  time: string;
  onlineUsers?: OnlineUser[];
}

const WS_RECONNECT_BASE = 2000;
const WS_RECONNECT_MAX = 30000;

export function useChatWs() {
  const ws = ref<WebSocket | null>(null);
  const connected = ref(false);
  const onlineUsers = ref<OnlineUser[]>([]);
  const messages = ref<ChatMessage[]>([]);
  const reconnectTimer = ref<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttempt = ref(0);

  const getWsUrl = (): string => {
    const token = localStorage.getItem('authToken');
    const isSecure = location.protocol === 'https:';
    const wsProtocol = isSecure ? 'wss:' : 'ws:';
    const host = location.host;
    return `${wsProtocol}//${host}/api/ws?token=${encodeURIComponent(token || '')}`;
  };

  const connect = () => {
    if (ws.value && (ws.value.readyState === WebSocket.OPEN || ws.value.readyState === WebSocket.CONNECTING)) {
      return;
    }

    const url = getWsUrl();
    ws.value = new WebSocket(url);

    ws.value.onopen = () => {
      connected.value = true;
      reconnectAttempt.value = 0;
    };

    ws.value.onmessage = (event) => {
      try {
        const msg: ChatMessage = JSON.parse(event.data);
        if (msg.type === 'online') {
          onlineUsers.value = msg.onlineUsers || [];
        } else {
          messages.value.push(msg);
        }
      } catch (e) {
        console.error('WS 消息解析失败:', e);
      }
    };

    ws.value.onclose = () => {
      connected.value = false;
      ws.value = null;
      scheduleReconnect();
    };

    ws.value.onerror = () => {
      // onclose 会接着触发，统一在 onclose 中重连
    };
  };

  const scheduleReconnect = () => {
    if (reconnectTimer.value) return;
    const delay = Math.min(
      WS_RECONNECT_BASE * Math.pow(2, reconnectAttempt.value),
      WS_RECONNECT_MAX
    );
    reconnectAttempt.value++;
    reconnectTimer.value = setTimeout(() => {
      reconnectTimer.value = null;
      connect();
    }, delay);
  };

  const disconnect = () => {
    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value);
      reconnectTimer.value = null;
    }
    if (ws.value) {
      ws.value.onclose = null; // 阻止重连
      ws.value.close();
      ws.value = null;
    }
    connected.value = false;
  };

  const sendChat = (content: string) => {
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) return;
    ws.value.send(JSON.stringify({
      type: 'chat',
      content: content,
    }));
  };

  return {
    connected,
    onlineUsers,
    messages,
    connect,
    disconnect,
    sendChat,
  };
}
