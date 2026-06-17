import { ref, onMounted, onUnmounted } from 'vue';

export interface SSEEvent {
  Type: string;
  Data: any;
}

export function useSSE(onEvent: (event: SSEEvent) => void) {
  const eventSource = ref<EventSource | null>(null);
  const isConnected = ref(false);
  const reconnectTimer = ref<number | null>(null);

  const connect = () => {
    if (eventSource.value) {
      eventSource.value.close();
    }

    const url = `${window.location.origin}/api/events`;
    eventSource.value = new EventSource(url);

    eventSource.value.onopen = () => {
      isConnected.value = true;
      console.log('SSE connected');
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
      isConnected.value = false;
      console.log('SSE disconnected, reconnecting in 3s...');
      
      if (eventSource.value) {
        eventSource.value.close();
        eventSource.value = null;
      }

      if (reconnectTimer.value) {
        clearTimeout(reconnectTimer.value);
      }

      reconnectTimer.value = window.setTimeout(() => {
        connect();
      }, 3000);
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
