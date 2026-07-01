import { ref } from 'vue';
import { useDialogPluginComponent } from 'quasar';
import { notifyTaskLog } from 'src/stores/taskLog';
import { SSEEventType } from 'src/types';

/**
 * 弹窗壳：提供 show/hide 生命周期 + 共享的 SSE task_log 连接。
 * 弹窗打开时自动连接 SSE，关闭时自动断开。
 */
export function useDialogShell(onHide?: () => void) {
  const show = ref(false);

  const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } = useDialogPluginComponent();

  // ── SSE ──────────────────────────────────────────────────────────
  let taskLogEventSource: EventSource | null = null;

  function openTaskLogSSE() {
    closeTaskLogSSE();
    taskLogEventSource = new EventSource(`${window.location.origin}/api/events`);
    taskLogEventSource.onmessage = (e) => {
      try {
        const event = JSON.parse(e.data);
        if (event.Type === SSEEventType.TaskLog && event.Data?.taskKey) {
          notifyTaskLog();
        }
      } catch { /* 忽略 */ }
    };
    taskLogEventSource.onerror = () => closeTaskLogSSE();
  }

  function closeTaskLogSSE() {
    if (taskLogEventSource) {
      taskLogEventSource.close();
      taskLogEventSource = null;
    }
  }

  const dialogHide = async () => {
    closeTaskLogSSE();
    onHide?.();
    onDialogCancel();
    onDialogOK();
    onDialogHide();
  };

  const beforeShow = () => {
    openTaskLogSSE();
  };

  return {
    show,
    dialogRef,
    dialogHide,
    beforeShow,
    onDialogHide,
    onDialogOK,
    onDialogCancel,
  };
}
