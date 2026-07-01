import { ref } from 'vue';

// 轻量通知计数器：SSE 收到 SSEEventType.TaskLog 通知时递增，组件 watch 检测到变化后发起 HTTP 拉取日志
const logVersion = ref(0);

/** SSE 通知前端某任务日志有更新，递增版本号 */
export function notifyTaskLog() {
  logVersion.value++;
}

export { logVersion };
