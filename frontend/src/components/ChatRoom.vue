<template>
  <q-dialog v-model="visible" persistent :maximized="$q.screen.lt.md">
    <q-card class="chat-card">
      <q-card-section class="row items-center q-pb-none">
        <div class="text-h6">聊天室</div>
        <q-space />
        <q-badge color="positive" v-if="connected" outline>
          {{ onlineUsers.length }} 人在线
        </q-badge>
        <q-badge color="grey" v-else outline>
          连接中...
        </q-badge>
        <q-btn flat round dense icon="close" v-close-popup class="q-ml-sm" />
      </q-card-section>

      <q-separator />

      <!-- 在线用户列表 -->
      <q-card-section class="online-section q-pb-none" v-if="onlineUsers.length > 0">
        <div class="text-caption text-grey q-mb-xs">在线用户</div>
        <div class="online-list">
          <q-chip
            v-for="user in onlineUsers"
            :key="user.username"
            size="sm"
            :color="user.role === 'super_admin' ? 'orange' : 'primary'"
            text-color="white"
          >
            <q-avatar icon="person" size="xs" />
            {{ user.username }}
          </q-chip>
        </div>
      </q-card-section>

      <q-separator v-if="onlineUsers.length > 0" />

      <!-- 消息列表 -->
      <q-card-section class="message-section">
        <div ref="messageContainer" class="message-list">
          <div v-if="messages.length === 0" class="text-center text-grey q-pa-lg">
            暂无消息，来打个招呼吧
          </div>
          <div
            v-for="(msg, idx) in messages"
            :key="idx"
            :class="['message-item', msg.username === currentUser ? 'message-self' : '']"
          >
            <div class="message-meta">
              <span class="message-user" :class="{ 'text-orange': msg.role === 'super_admin' }">
                {{ msg.username || '系统' }}
              </span>
              <span class="message-time">{{ formatTime(msg.time) }}</span>
            </div>
            <div class="message-content">{{ msg.content }}</div>
          </div>
        </div>
      </q-card-section>

      <q-separator />

      <!-- 输入框 -->
      <q-card-section class="input-section q-pt-sm">
        <q-input
          v-model="inputText"
          outlined
          dense
          placeholder="输入消息..."
          @keyup.enter="sendMessage"
          :disable="!connected"
        >
          <template v-slot:append>
            <q-btn
              flat
              round
              dense
              icon="send"
              @click="sendMessage"
              :disable="!connected || !inputText.trim()"
            />
          </template>
        </q-input>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue';
import { useChatWs, type ChatMessage } from 'src/composables/useChatWs';

const visible = ref(false);
const inputText = ref('');
const messageContainer = ref<HTMLElement | null>(null);
const currentUser = localStorage.getItem('username') || '';

const { connected, onlineUsers, messages, connect, disconnect, sendChat } = useChatWs();

// 自动滚动到底部
const scrollToBottom = () => {
  nextTick(() => {
    if (messageContainer.value) {
      messageContainer.value.scrollTop = messageContainer.value.scrollHeight;
    }
  });
};

watch(messages, () => {
  scrollToBottom();
}, { deep: true });

watch(visible, (val) => {
  if (val) {
    connect();
  } else {
    // 对话框关闭不主动断开，保持后台连接
  }
});

const sendMessage = () => {
  const text = inputText.value.trim();
  if (!text || !connected.value) return;
  sendChat(text);
  inputText.value = '';
};

const formatTime = (timeStr: string) => {
  if (!timeStr) return '';
  const d = new Date(timeStr);
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  const ss = String(d.getSeconds()).padStart(2, '0');
  return `${hh}:${mm}:${ss}`;
};

const open = () => {
  visible.value = true;
};

defineExpose({ open });
</script>

<style lang="scss" scoped>
.chat-card {
  width: 450px;
  max-width: 90vw;
  height: 550px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.online-section {
  flex-shrink: 0;
}

.online-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.message-section {
  flex: 1;
  overflow: hidden;
  padding-top: 8px;
}

.message-list {
  height: 100%;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 4px;
}

.message-item {
  max-width: 80%;
  align-self: flex-start;
}

.message-self {
  align-self: flex-end;
}

.message-self .message-content {
  background: var(--q-primary);
  color: white;
  border-radius: 12px 12px 4px 12px;
}

.message-content {
  background: var(--q-bg-secondary, #f0f0f0);
  padding: 8px 12px;
  border-radius: 4px 12px 12px 12px;
  word-break: break-word;
  line-height: 1.4;
}

.message-meta {
  font-size: 11px;
  color: var(--q-grey);
  margin-bottom: 2px;
  padding: 0 4px;
  display: flex;
  gap: 8px;
}

.message-user {
  font-weight: 600;
}

.message-self .message-meta {
  text-align: right;
  justify-content: flex-end;
}

.input-section {
  flex-shrink: 0;
}

.message-list::-webkit-scrollbar {
  width: 4px;
}
.message-list::-webkit-scrollbar-thumb {
  background: var(--q-grey-5);
  border-radius: 2px;
}
</style>
