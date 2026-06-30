import { ref } from 'vue';
import { useTaskLogStore } from 'src/stores/taskLog';

export interface SSEEvent {
  Type: string;
  Data: any;
}

const SSE_MAX_BACKOFF = 30_000;

// 全局单例 — 不绑定组件生命周期，路由切换不断连
let eventSource: EventSource | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let backoffMs = 3_000;
let reconnecting = false;

const isConnected = ref(false);

function connect() {
  if (eventSource) {
    eventSource.close();
  }

  const url = `${window.location.origin}/api/events`;
  eventSource = new EventSource(url);

  eventSource.onopen = () => {
    isConnected.value = true;
    reconnecting = false;
    backoffMs = 3_000;
  };

  eventSource.onmessage = (e) => {
    try {
      const event: SSEEvent = JSON.parse(e.data);
      handleEvent(event);
    } catch (err) {
      console.error('SSE parse error:', err);
    }
  };

  eventSource.onerror = () => {
    if (reconnecting) return;
    reconnecting = true;
    isConnected.value = false;

    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }

    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
    }

    reconnectTimer = setTimeout(() => {
      connect();
      backoffMs = Math.min(backoffMs * 2, SSE_MAX_BACKOFF);
    }, backoffMs);
  };
}

function disconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  isConnected.value = false;
}

// 事件路由
function handleEvent(event: SSEEvent) {
  switch (event.Type) {
    case 'task_log': {
      const data = event.Data;
      if (data?.taskKey && data?.line !== undefined) {
        const store = useTaskLogStore();
        store.appendLine(data.taskKey, data.line);
      }
      break;
    }
    // index_update 等其他事件由原有 useSSE 组件处理，这里不重复分发
  }
}

export function useGlobalSSE() {
  return {
    isConnected,
    connect,
    disconnect,
  };
}
