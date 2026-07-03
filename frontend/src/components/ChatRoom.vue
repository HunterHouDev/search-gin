<template>
  <q-dialog v-model="visible" persistent :maximized="isMaximized">
    <q-card class="chat-card">
      <!-- 标题栏 -->
      <q-card-section class="row items-center q-pb-none">
        <div class="text-h6">
          <q-icon :name="activeTab === 'chat' ? 'chat' : 'videocam'" size="sm" class="q-mr-xs" />
          {{ activeTab === 'chat' ? '聊天室' : '视频会议' }}
        </div>
        <q-space />
        <q-badge color="positive" v-if="connected" outline>
          {{ onlineUsers.length }} 人在线
        </q-badge>
        <q-badge color="grey" v-else-if="!connectionFailed" outline>连接中...</q-badge>
        <q-badge color="negative" v-else outline>
          连接失败
          <q-btn flat dense round icon="refresh" size="xs" @click="retryConnect" class="q-ml-xs" />
        </q-badge>
        <q-btn flat round dense icon="close" v-close-popup class="q-ml-sm" />
      </q-card-section>

      <q-separator />

      <!-- Tab 切换 -->
      <q-tabs v-model="activeTab" dense active-color="primary" indicator-color="primary">
        <q-tab name="chat" icon="chat" label="聊天" no-caps />
        <q-tab name="video" icon="videocam" label="视频" no-caps />
      </q-tabs>

      <q-separator />

      <!-- ====== 聊天 Tab ====== -->
      <template v-if="activeTab === 'chat'">
        <!-- 在线用户列表 -->
        <q-card-section class="online-section q-pb-none" v-if="onlineUsers.length > 0">
          <div class="text-caption text-grey q-mb-xs">在线用户</div>
          <div class="online-list">
            <q-chip
              v-for="(user, idx) in onlineUsers"
              :key="idx"
              size="sm"
              :color="user.Role === 'super_admin' ? 'orange' : 'primary'"
              text-color="white"
            >
              <q-avatar icon="person" size="xs" />
              {{ user.Username }}
            </q-chip>
          </div>
        </q-card-section>
        <q-separator v-if="onlineUsers.length > 0" />

        <!-- 消息列表 -->
        <q-card-section class="message-section">
          <div ref="messageContainer" class="message-list">
            <div v-if="messages.length === 0" class="text-center text-grey q-pa-lg">暂无消息，来打个招呼吧</div>
            <div
              v-for="(msg, idx) in messages"
              :key="idx"
              :class="['message-item', msg.username === currentUser && msg.ip === currentIP ? 'message-self' : '']"
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
              <q-btn flat round dense icon="send" @click="sendMessage" :disable="!connected || !inputText.trim()" />
            </template>
          </q-input>
        </q-card-section>
      </template>

      <!-- ====== 视频 Tab ====== -->
      <template v-if="activeTab === 'video'">
        <q-card-section class="video-section">
          <!-- 视频网格 -->
          <div class="video-grid">
            <!-- 本地视频 -->
            <div class="video-cell local-cell" v-if="localStream">
              <video ref="localVideoRef" autoplay muted playsinline class="video-player"></video>
              <div class="video-label">我 ({{ currentUser }})</div>
            </div>

            <!-- 远程视频 -->
            <div
              v-for="(stream, username) in remoteStreams"
              :key="username"
              class="video-cell"
            >
              <video :ref="(el: Element | ComponentPublicInstance | null) => setRemoteVideo(el, stream)" autoplay playsinline class="video-player"></video>
              <div class="video-label">{{ username }}</div>
            </div>

            <!-- 等待加入 -->
            <div v-if="!inCall" class="video-cell video-empty">
              <q-icon name="videocam" size="48px" color="grey-4" />
              <div class="text-grey-6 q-mt-sm">点击下方按钮加入会议</div>
            </div>

            <!-- 已加入但无人时的提示 -->
            <div v-if="inCall && Object.keys(remoteStreams).length === 0" class="video-cell video-empty">
              <q-spinner-dots size="32px" color="primary" />
              <div class="text-grey-6 q-mt-sm">等待其他人加入...</div>
            </div>
          </div>

          <!-- 错误提示 -->
          <div v-if="videoError" class="text-negative text-center text-caption q-mt-xs">
            {{ videoError }}
          </div>
        </q-card-section>

        <!-- 底部控制栏 -->
        <q-card-section class="video-controls q-pt-sm">
          <div class="row justify-center items-center q-gutter-sm">
            <!-- 加入/挂断 -->
            <q-btn
              v-if="!inCall"
              round
              color="positive"
              icon="call"
              size="lg"
              @click="joinVideo"
              :loading="videoLoading"
            >
              <q-tooltip>加入会议</q-tooltip>
            </q-btn>
            <q-btn
              v-else
              round
              color="negative"
              icon="call_end"
              size="lg"
              @click="leaveVideo"
            >
              <q-tooltip>挂断</q-tooltip>
            </q-btn>

            <!-- 麦克风 -->
            <q-btn
              round
              :color="micActive ? 'primary' : 'grey'"
              :icon="micActive ? 'mic' : 'mic_off'"
              size="md"
              @click="toggleVideoMic"
              :disable="!inCall"
            >
              <q-tooltip>{{ micActive ? '关闭麦克风' : '开启麦克风' }}</q-tooltip>
            </q-btn>

            <!-- 摄像头 -->
            <q-btn
              round
              :color="camActive ? 'primary' : 'grey'"
              :icon="camActive ? 'videocam' : 'videocam_off'"
              size="md"
              @click="toggleVideoCam"
              :disable="!inCall"
            >
              <q-tooltip>{{ camActive ? '关闭摄像头' : '开启摄像头' }}</q-tooltip>
            </q-btn>
          </div>
        </q-card-section>
      </template>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, type ComponentPublicInstance } from 'vue';
import { useChatWs } from 'src/composables/useChatWs';
import { useVideoConference } from 'src/composables/useVideoConference';
import { useQuasar } from 'quasar';

const $q = useQuasar();

// ── 聊天 ──
const visible = ref(false);
const inputText = ref('');
const messageContainer = ref<HTMLElement | null>(null);
const currentUser = sessionStorage.getItem('username') || '';
const currentIP = ref('');
const activeTab = ref('chat');

const {
  connected, connectionFailed, onlineUsers, messages,
  connect, sendChat, retryConnect,
} = useChatWs();

// 从在线用户列表中找到自己的 IP
watch(onlineUsers, (users) => {
  const self = users.find(u => u.Username === currentUser);
  if (self && self.IP) {
    currentIP.value = self.IP;
  }
}, { immediate: true });

// ── 视频会议 ──
const {
  localStream, remoteStreams, inCall,
  micEnabled, camEnabled,
  error: videoError,
  join: joinVideoCall, leave: leaveVideoCall,
  toggleMic: toggleVideoMic, toggleCam: toggleVideoCam,
} = useVideoConference();

const localVideoRef = ref<HTMLVideoElement | null>(null);
const videoLoading = ref(false);

const isMaximized = computed(() => $q.screen.lt.md);
const micActive = computed(() => micEnabled.value && inCall.value);
const camActive = computed(() => camEnabled.value && inCall.value);

// 将本地流绑定到 video 元素
watch(localStream, (stream) => {
  if (localVideoRef.value && stream) {
    localVideoRef.value.srcObject = stream;
  }
}, { immediate: true });

// 绑定远程流
function setRemoteVideo(el: Element | ComponentPublicInstance | null, stream: MediaStream) {
  if (el && stream) {
    (el as HTMLVideoElement).srcObject = stream;
  }
}

// 加入视频会议
async function joinVideo() {
  videoLoading.value = true;
  await joinVideoCall();
  videoLoading.value = false;
}

// 离开视频会议
function leaveVideo() {
  leaveVideoCall();
}

// ── 聊天功能 ──
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
  width: 700px;
  max-width: 90vw;
  height: 600px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

// ── 聊天 ──
.online-section { flex-shrink: 0; }

.online-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  position: relative;
}

.device-badge {
  font-size: 9px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
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

.message-item { max-width: 80%; align-self: flex-start; }
.message-self { align-self: flex-end; }
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

.message-user { font-weight: 600; }
.message-self .message-meta {
  text-align: right;
  justify-content: flex-end;
}

.input-section { flex-shrink: 0; }

// ── 视频 ──
.video-section {
  flex: 1;
  overflow: hidden;
  padding: 4px;
  display: flex;
  flex-direction: column;
}

.video-grid {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-content: flex-start;
  overflow-y: auto;
  justify-content: center;
}

.video-cell {
  position: relative;
  width: 200px;
  height: 150px;
  background: #1a1a2e;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
}

.video-cell.local-cell {
  border: 2px solid var(--q-primary);
}

.video-player {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.video-label {
  position: absolute;
  bottom: 4px;
  left: 4px;
  background: rgba(0,0,0,0.6);
  color: white;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  max-width: calc(100% - 8px);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.video-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
}

.video-controls {
  flex-shrink: 0;
  padding-bottom: 8px;
}

// ── 滚动条 ──
.message-list::-webkit-scrollbar,
.video-grid::-webkit-scrollbar {
  width: 4px;
}
.message-list::-webkit-scrollbar-thumb,
.video-grid::-webkit-scrollbar-thumb {
  background: var(--q-grey-5);
  border-radius: 2px;
}
</style>
