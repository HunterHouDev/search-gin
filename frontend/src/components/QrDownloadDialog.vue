<template>
  <q-dialog v-model="visible" @before-hide="onClose" maximized>
    <q-card class="qr-dialog-card">
      <q-card-section class="row items-center q-pb-none">
        <div class="text-h6">扫码下载</div>
        <q-space />
        <q-btn flat round dense icon="close" v-close-popup />
      </q-card-section>

      <q-card-section class="q-pt-md">
        <!-- 文件信息 -->
        <div class="file-info">
          <div class="file-title">{{ item?.Title || item?.Name }}</div>
          <div class="file-meta">
            <span v-if="item?.SizeStr">{{ item.SizeStr }}</span>
            <span v-if="item?.Code" class="q-ml-sm">{{ item.Code }}</span>
          </div>
        </div>

        <!-- 二维码区域 -->
        <div class="qr-wrapper">
          <canvas ref="qrCanvasRef"></canvas>
          <div class="qr-hint">请用手机扫码下载或播放</div>
        </div>

        <!-- 下载链接（可复制） -->
        <div class="url-row">
          <q-input
            v-model="downloadUrl"
            readonly
            dense
            outlined
            label="下载链接"
            class="url-input"
            @click="copyUrl"
          >
            <template v-slot:append>
              <q-btn flat dense icon="content_copy" @click="copyUrl" />
            </template>
          </q-input>
        </div>

        <!-- 提示 -->
        <q-banner class="bg-grey-2 q-mt-md rounded-borders" dense>
          <template v-slot:avatar>
            <q-icon name="info" color="primary" />
          </template>
          <span class="text-caption">
            手机与电脑需在同一局域网（同一 WiFi）才能下载。<br />
            手机扫码后选择「在浏览器中打开」即可开始。
          </span>
        </q-banner>
      </q-card-section>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue';
import QRCode from 'qrcode';
import { useClipboard } from '@vueuse/core';
import { useQuasar } from 'quasar';
import type { FileItem } from 'src/types';

const $q = useQuasar();
const { copy: copyToClipboard } = useClipboard();

const props = defineProps<{
  modelValue: boolean;
  item?: FileItem | null;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void;
}>();

const visible = ref(false);
const qrCanvasRef = ref<HTMLCanvasElement | null>(null);
const downloadUrl = ref('');

watch(() => props.modelValue, (val) => {
  visible.value = val;
  if (val && props.item) {
    generateQr();
  }
});

watch(visible, (val) => {
  emit('update:modelValue', val);
});

function generateQr() {
  const item = props.item;
  if (!item?.StreamUrl) return;

  // 使用 StreamUrl 作为二维码内容
  downloadUrl.value = item.StreamUrl;

  nextTick(() => {
    if (qrCanvasRef.value) {
      QRCode.toCanvas(qrCanvasRef.value, item.StreamUrl, {
        width: 240,
        margin: 2,
        color: {
          dark: '#1a1a2e',
          light: '#ffffff',
        },
      }, (err: any) => {
        if (err) {
          console.error('QR 生成失败:', err);
        }
      });
    }
  });
}

async function copyUrl() {
  try {
    copyToClipboard(downloadUrl.value);
    $q.notify({ type: 'positive', message: '链接已复制', position: 'top', timeout: 1500 });
  } catch {
    $q.notify({ type: 'negative', message: '复制失败', position: 'top' });
  }
}

function onClose() {
  downloadUrl.value = '';
}
</script>

<style scoped>
.qr-dialog-card {
  max-width: 420px;
  margin: 0 auto;
  border-radius: 16px;
}

.file-info {
  text-align: center;
  margin-bottom: 16px;
}

.file-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--q-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta {
  font-size: 13px;
  color: #888;
  margin-top: 4px;
}

.qr-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0;
}

.qr-wrapper canvas {
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.qr-hint {
  margin-top: 12px;
  font-size: 13px;
  color: #888;
}

.url-row {
  margin-top: 8px;
}

.url-input :deep(.q-field__native) {
  font-size: 12px;
  color: #555;
}
</style>
