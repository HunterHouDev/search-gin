import { defineStore } from 'pinia';
import { ref } from 'vue';

export interface TaskLogLine {
  taskKey: string;
  line: string;
}

export const useTaskLogStore = defineStore('taskLog', () => {
  // taskKey → lines[]，只保留最近 N 行防 OOM
  const MAX_LINES = 2000;
  const logMap = ref<Record<string, string[]>>({});

  function appendLine(taskKey: string, line: string) {
    if (!logMap.value[taskKey]) {
      logMap.value[taskKey] = [];
    }
    const lines = logMap.value[taskKey];
    lines.push(line);
    if (lines.length > MAX_LINES) {
      lines.splice(0, lines.length - MAX_LINES);
    }
  }

  function getLogs(taskKey: string): string[] {
    return logMap.value[taskKey] || [];
  }

  function clearLogs(taskKey: string) {
    delete logMap.value[taskKey];
  }

  function clearAll() {
    logMap.value = {};
  }

  return {
    logMap,
    appendLine,
    getLogs,
    clearLogs,
    clearAll,
  };
});
