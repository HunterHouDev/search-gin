import { ref } from 'vue';

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
const WS_CONNECT_TIMEOUT = 10000; // 连接超时：10 秒

// 单例状态，所有 useChatWs() 调用共享同一个 WebSocket 连接
const ws = ref<WebSocket | null>(null);
const connected = ref(false);
const onlineUsers = ref<OnlineUser[]>([]);
const messages = ref<ChatMessage[]>([]);
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempt = 0;
let connectTimer: ReturnType<typeof setTimeout> | null = null;

function getWsUrl(): string {
  const token = localStorage.getItem('authToken');
  const isSecure = location.protocol === 'https:';
  const wsProtocol = isSecure ? 'wss:' : 'ws:';
  const host = location.host;
  return `${wsProtocol}//${host}/api/ws?token=${encodeURIComponent(token || '')}`;
}

function scheduleReconnect() {
  if (reconnectTimer) return;
  const delay = Math.min(
    WS_RECONNECT_BASE * Math.pow(2, reconnectAttempt),
    WS_RECONNECT_MAX
  );
  reconnectAttempt++;
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connectSingleton();
  }, delay);
}

function connectSingleton() {
  if (ws.value && (ws.value.readyState === WebSocket.OPEN || ws.value.readyState === WebSocket.CONNECTING)) {
    return;
  }

  const url = getWsUrl();
  ws.value = new WebSocket(url);

  // 连接超时：10 秒内未响应则主动关闭并重连
  if (connectTimer) clearTimeout(connectTimer);
  connectTimer = setTimeout(() => {
    if (ws.value && ws.value.readyState === WebSocket.CONNECTING) {
      console.warn('WebSocket 连接超时，主动关闭并重连');
      ws.value.close();
    }
  }, WS_CONNECT_TIMEOUT);

  ws.value.onopen = () => {
    if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
    connected.value = true;
    reconnectAttempt = 0;
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
    if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
    connected.value = false;
    ws.value = null;
    scheduleReconnect();
  };

  ws.value.onerror = (event) => {
    console.error('WebSocket 连接错误:', event);
    // onclose 会接着触发，统一在 onclose 中重连
  };
}

function disconnectSingleton() {
  if (connectTimer) { clearTimeout(connectTimer); connectTimer = null; }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (ws.value) {
    ws.value.onclose = null;
    ws.value.close();
    ws.value = null;
  }
  connected.value = false;
}

export function useChatWs() {
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
    connect: connectSingleton,
    disconnect: disconnectSingleton,
    sendChat,
  };
}
