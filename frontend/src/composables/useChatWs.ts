import { ref } from 'vue';
import { commonAxios, api } from 'src/boot/axios';

export interface OnlineUser {
  username: string;
  role: string;
  deviceCount: number;
  ips?: string[];
}

export interface ChatMessage {
  type: 'online' | 'chat' | 'system' | 'signal';
  username?: string;
  role?: string;
  content?: string;
  from?: string;
  action?: string;
  data?: unknown;
  time: string;
  onlineUsers?: OnlineUser[];
}

const WS_RECONNECT_BASE = 2000;
const WS_RECONNECT_MAX = 30000;
const WS_CONNECT_TIMEOUT = 10000;
const WS_MAX_RETRY = 5;

// 单例状态
const ws = ref<WebSocket | null>(null);
const connected = ref(false);
const connectionFailed = ref(false);
const onlineUsers = ref<OnlineUser[]>([]);
const messages = ref<ChatMessage[]>([]);

// 信令回调
type SignalHandler = (msg: ChatMessage) => void;
const signalHandlers: SignalHandler[] = [];

let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempt = 0;
let connectTimer: ReturnType<typeof setTimeout> | null = null;

function redirectToLogin() {
  sessionStorage.removeItem('authToken');
  sessionStorage.removeItem('isAuthenticated');
  sessionStorage.removeItem('userRole');
  sessionStorage.removeItem('username');
  window.location.href = '/#/login';
}

function getWsUrl(): string {
  const token = sessionStorage.getItem('authToken');
  const apiUrl = api.defaults.baseURL || `http://${location.host}`;
  const apiHost = apiUrl.replace(/^https?:\/\//, '');
  const isSecure = apiUrl.startsWith('https:');
  const wsProtocol = isSecure ? 'wss:' : 'ws:';
  return `${wsProtocol}//${apiHost}/api/ws?token=${encodeURIComponent(token || '')}`;
}

function scheduleReconnect() {
  if (reconnectTimer) return;
  reconnectAttempt++;
  if (reconnectAttempt > WS_MAX_RETRY) {
    connectionFailed.value = true;
    const token = sessionStorage.getItem('authToken');
    if (token) {
      commonAxios().get('/api/heartBeat').catch((err) => {
        if (err?.response?.status === 401) {
          redirectToLogin();
        }
      });
    }
    return;
  }
  const delay = Math.min(
    WS_RECONNECT_BASE * Math.pow(2, reconnectAttempt - 1),
    WS_RECONNECT_MAX
  );
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connectSingleton();
  }, delay);
}

function connectSingleton() {
  if (ws.value && (ws.value.readyState === WebSocket.OPEN || ws.value.readyState === WebSocket.CONNECTING)) {
    return;
  }

  connectionFailed.value = false;

  const url = getWsUrl();
  ws.value = new WebSocket(url);

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
    connectionFailed.value = false;
    reconnectAttempt = 0;
  };

  ws.value.onmessage = (event) => {
    try {
      const msg: ChatMessage = JSON.parse(event.data);

      if (msg.type === 'online') {
        onlineUsers.value = msg.onlineUsers || [];
      } else if (msg.type === 'signal') {
        // 分发给视频会议的回调
        signalHandlers.forEach(fn => fn(msg));
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
    if (!connectionFailed.value) {
      scheduleReconnect();
    }
  };

  ws.value.onerror = () => {
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
  connectionFailed.value = false;
  reconnectAttempt = 0;
}

export function useChatWs() {
  const sendChat = (content: string) => {
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) return;
    ws.value.send(JSON.stringify({
      type: 'chat',
      content,
    }));
  };

  const sendJSON = (data: object) => {
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) return false;
    ws.value.send(JSON.stringify(data));
    return true;
  };

  const onSignal = (handler: SignalHandler) => {
    signalHandlers.push(handler);
    return () => {
      const idx = signalHandlers.indexOf(handler);
      if (idx >= 0) signalHandlers.splice(idx, 1);
    };
  };

  const retryConnect = () => {
    connectionFailed.value = false;
    reconnectAttempt = 0;
    connectSingleton();
  };

  return {
    connected,
    connectionFailed,
    onlineUsers,
    messages,
    connect: connectSingleton,
    disconnect: disconnectSingleton,
    sendChat,
    sendJSON,
    onSignal,
    retryConnect,
  };
}
