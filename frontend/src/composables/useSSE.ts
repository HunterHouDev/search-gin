import { ref, onMounted, onUnmounted } from 'vue';

export interface SSEEvent {
  Type: string;
  Data: unknown;
}

const SSE_MAX_BACKOFF = 30_000; // 最大退避 30s

export function useSSE(onEvent: (event: SSEEvent) => void) {
  const eventSource = ref<EventSource | null>(null);
  const isConnected = ref(false);
  const reconnectTimer = ref<number | null>(null);
  let backoffMs = 3_000;           // 初始退避 3s
  let reconnecting = false;         // 防风暴：一次 error 只触发一次重连

  const connect = () => {
    if (eventSource.value) {
      eventSource.value.close();
    }

    const url = `${window.location.origin}/api/events`;
    eventSource.value = new EventSource(url);

    eventSource.value.onopen = () => {
      isConnected.value = true;
      reconnecting = false;
      backoffMs = 3_000; // 重连成功，重置退避

    };

    eventSource.value.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data) as SSEEvent;
        onEvent(event);
      } catch (err) {
        console.error('SSE parse error:', err);
      }
    };

    eventSource.value.onerror = () => {
      // 防风暴：已经在重连中则忽略后续 error
      if (reconnecting) return;
      reconnecting = true;
      isConnected.value = false;

      if (eventSource.value) {
        eventSource.value.close();
        eventSource.value = null;
      }

      if (reconnectTimer.value) {
        clearTimeout(reconnectTimer.value);
      }


      reconnectTimer.value = window.setTimeout(() => {
        connect();
        // 指数增长，上限 30s
        backoffMs = Math.min(backoffMs * 2, SSE_MAX_BACKOFF);
      }, backoffMs);
    };
  };

  const disconnect = () => {
    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value);
      reconnectTimer.value = null;
    }

    if (eventSource.value) {
      eventSource.value.close();
      eventSource.value = null;
    }

    isConnected.value = false;
  };

  onMounted(() => {
    connect();
  });

  onUnmounted(() => {
    disconnect();
  });

  return {
    isConnected,
    connect,
    disconnect,
  };
}
