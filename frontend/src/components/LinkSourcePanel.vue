<template>
  <div class="link-source-panel">
    <!-- 链接类型 Tab -->
    <div class="link-tabs">
      <button
        v-for="tab in visibleTabs"
        :key="tab.value"
        type="button"
        class="link-tab"
        :class="{ 'link-tab-active': linkTab === tab.value }"
        @click="switchLinkTab(tab.value)"
      >
        <q-icon :name="tab.icon" size="15px" />
        <span>{{ tab.label }}</span>
      </button>
    </div>

    <!-- 链接输入框 -->
    <div
      class="magnet-input-wrapper"
      :class="{ 'magnet-focused': linkFocused }"
    >
      <q-icon
        :name="activeLinkTab.icon"
        color="indigo-4"
        size="20px"
        class="magnet-icon"
      />
      <q-input
        v-model="activeLinkValue"
        :placeholder="activeLinkTab.placeholder"
        dark
        dense
        borderless
        class="magnet-input"
        @keyup.enter="submitLink"
        @focus="linkFocused = true"
        @blur="linkFocused = false"
      />
      <q-btn
        flat
        dense
        no-caps
        color="indigo-4"
        :icon="linkActionIcon"
        :label="linkActionLabel"
        size="sm"
        @click="submitLink"
        :disable="!canSubmitLink"
        :loading="linkActionLoading"
        class="magnet-submit-btn"
      >
        <q-tooltip class="bg-dark text-white">{{
          linkActionTooltip
        }}</q-tooltip>
      </q-btn>
    </div>

    <!-- 分片列表：解析后可删除分片（如广告）再播放剩余部分 -->
    <transition name="fade">
      <div
        v-if="linkTab === 'hls' && (hlsParsed || hlsDownloadList.length)"
        class="hls-segment-panel"
      >
        <template v-if="hlsParsed">
          <div class="hls-segment-header">
            <q-icon name="playlist_play" size="16px" color="indigo-4" />
            <span class="hls-segment-summary">
              保留 {{ hlsKeptCount }}/{{ hlsTotalCount }} 个分片 ·
              {{ hlsKeptDuration }}
              <template v-if="hlsRemovedCount"
                >（已删除 {{ hlsRemovedCount }}）</template
              >
            </span>
            <q-space />
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="indigo-4"
              icon="restart_alt"
              label="恢复"
              :disable="!hlsRemovedCount"
              @click="restoreHlsSegments"
            >
              <q-tooltip class="bg-dark text-white"
                >恢复全部已删除分片</q-tooltip
              >
            </q-btn>
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="indigo-4"
              icon="link"
              label="复制链接"
              :disable="!hlsKeptCount"
              @click="copyAllSegmentUrls"
            >
              <q-tooltip class="bg-dark text-white"
                >复制保留的 {{ hlsKeptCount }} 个分片链接（每行一个）</q-tooltip
              >
            </q-btn>
            <q-btn
              unelevated
              dense
              no-caps
              size="sm"
              color="indigo-6"
              icon="play_arrow"
              label="播放"
              class="hls-play-btn"
              :loading="hlsLoading"
              :disable="!hlsKeptCount"
              @click="playHlsRemaining"
            >
              <q-tooltip class="bg-dark text-white"
                >播放剩余 {{ hlsKeptCount }} 个分片</q-tooltip
              >
            </q-btn>
          </div>
          <!-- 下载设置：文件名 + 目录 + 下载按钮，撑满一行（文件名自适应剩余宽度） -->
          <div class="hls-download-row">
            <q-input
              v-model="hlsDownloadName"
              dark
              dense
              borderless
              class="hls-filename-input"
              :placeholder="hlsDefaultDownloadName"
              :disable="!hlsKeptCount"
              maxlength="120"
              @keyup.enter="downloadHls"
            >
              <template #prepend>
                <q-icon name="edit_note" size="16px" color="indigo-4" />
              </template>
              <q-tooltip class="bg-dark text-white">
                自定义保存文件名，留空则用默认名
                {{ hlsDefaultDownloadName }}
              </q-tooltip>
            </q-input>
            <q-btn-dropdown
              flat
              dense
              no-caps
              size="sm"
              class="hls-dir-btn"
              :color="hlsDownloadDir ? 'green-4' : 'indigo-4'"
              :icon="hlsDownloadDir ? 'folder_special' : 'create_new_folder'"
              :label="hlsDownloadDir || '下载目录'"
            >
              <q-list dense class="hls-dir-menu">
                <q-item clickable v-close-popup @click="chooseHlsDownloadDir('')">
                  <q-item-section>
                    <q-item-label>默认目录</q-item-label>
                    <q-item-label caption
                      >服务端默认保存位置（第一个媒体目录）</q-item-label
                    >
                  </q-item-section>
                </q-item>
                <q-item
                  v-for="dir in hlsDownloadDirOptions"
                  :key="dir"
                  clickable
                  v-close-popup
                  @click="chooseHlsDownloadDir(dir)"
                >
                  <q-item-section>
                    <q-item-label class="hls-dir-menu-label">{{
                      dir
                    }}</q-item-label>
                  </q-item-section>
                </q-item>
              </q-list>
              <q-tooltip class="bg-dark text-white">
                {{
                  hlsDownloadDir
                    ? `下载由服务端存入「${hlsDownloadDir}」；点击可更换`
                    : '选择服务端保存目录，下载由服务端执行'
                }}
              </q-tooltip>
            </q-btn-dropdown>
            <!-- 点下载后任务交给服务端，这里保持可用以便继续提交下载 -->
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="indigo-4"
              icon="download"
              label="下载"
              class="hls-download-btn"
              :disable="!hlsKeptCount"
              @click="downloadHls"
            >
              <q-tooltip class="bg-dark text-white"
                >由服务端下载 {{ hlsKeptCount }} 个分片并合并保存，关闭窗口 /
                刷新页面都不会中断</q-tooltip
              >
            </q-btn>
          </div>
          <div class="hls-segment-list">
            <div
              v-for="seg in hlsSegments"
              :key="seg.id"
              class="hls-segment-item"
            >
              <span class="hls-segment-index">#{{ seg.index }}</span>
              <span class="hls-segment-duration">{{
                seg.duration ? seg.duration.toFixed(1) + 's' : '--'
              }}</span>
              <span
                class="hls-segment-url"
                :title="seg.url"
                @click="copySegmentUrl(seg)"
                >{{ seg.url }}</span
              >
              <q-btn
                flat
                round
                dense
                size="sm"
                color="indigo-4"
                icon="content_copy"
                @click="copySegmentUrl(seg)"
              >
                <q-tooltip class="bg-dark text-white">复制该分片链接</q-tooltip>
              </q-btn>
              <q-btn
                flat
                round
                dense
                size="sm"
                color="deep-orange-4"
                icon="delete_sweep"
                @click="removeHlsSimilarSegments(seg.id)"
              >
                <q-tooltip class="bg-dark text-white">
                  删除同类分片（最后一节不同、前面都相同的全部删除）
                </q-tooltip>
              </q-btn>
              <q-btn
                flat
                round
                dense
                size="sm"
                color="red-4"
                icon="delete_outline"
                @click="removeHlsSegment(seg.id)"
              >
                <q-tooltip class="bg-dark text-white">删除该分片</q-tooltip>
              </q-btn>
            </div>
          </div>
        </template>

        <!-- 下载列表：点「下载」即出现在这里（取消 / 播放）；重新添加链接不会清空 -->
        <div v-if="hlsDownloadList.length" class="hls-download-list-panel">
          <div class="hls-download-list-header">
            <q-icon name="download_done" size="16px" color="green-4" />
            <span class="hls-download-list-title"
              >下载列表 · {{ hlsDownloadList.length }}</span
            >
            <q-space />
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="grey-5"
              icon="delete_sweep"
              label="清空"
              @click="clearHlsDownloads"
            >
              <q-tooltip class="bg-dark text-white"
                >删除服务端的下载任务记录，不会删除已下载的文件</q-tooltip
              >
            </q-btn>
          </div>
          <div class="hls-download-list">
            <div
              v-for="item in hlsDownloadList"
              :key="item.id"
              class="hls-download-item"
              :class="{
                'hls-download-item-playing': item.id === hlsPlayingDownloadId,
              }"
            >
              <q-icon
                name="movie"
                size="16px"
                color="indigo-4"
                class="hls-download-item-icon"
              />
              <div class="hls-download-item-info">
                <span class="hls-download-item-name" :title="item.target">{{
                  item.name
                }}</span>
                <span class="hls-download-item-meta">{{
                  hlsDownloadMeta(item)
                }}</span>
                <q-linear-progress
                  v-if="item.status === 'downloading'"
                  :value="item.progress / 100"
                  size="3px"
                  color="indigo-4"
                  track-color="rgba(99, 102, 241, 0.18)"
                  class="hls-download-item-bar"
                />
              </div>
              <!-- 取消：下载中中止任务，已结束则移除这条记录 -->
              <q-btn
                flat
                round
                dense
                size="sm"
                :color="item.status === 'downloading' ? 'red-4' : 'grey-5'"
                :icon="
                  item.status === 'downloading' ? 'close' : 'delete_outline'
                "
                @click="cancelHlsDownload(item.id)"
              >
                <q-tooltip class="bg-dark text-white">
                  {{
                    item.status === 'downloading'
                      ? '取消服务端下载任务'
                      : '移除该任务（不删除已下载的文件）'
                  }}
                </q-tooltip>
              </q-btn>
              <!-- 播放：下载完成后回放本地文件 -->
              <q-btn
                flat
                round
                dense
                size="sm"
                color="green-4"
                icon="play_arrow"
                :disable="item.status !== 'done' || !item.playable"
                @click="playHlsDownload(item)"
              >
                <q-tooltip class="bg-dark text-white">
                  {{
                    item.status !== 'done'
                      ? '服务端下载完成后可播放'
                      : item.playable
                        ? '播放已下载的视频'
                        : '该文件不在媒体目录内，无法在页面内回放'
                  }}
                </q-tooltip>
              </q-btn>
            </div>
          </div>
        </div>
      </div>
    </transition>

    <!-- 内嵌播放器：宿主不提供 video 时（如批量编辑弹窗）由组件自己承接播放 -->
    <div v-if="embedded" class="link-source-player">
      <video
        ref="internalVideoRef"
        class="link-source-video"
        controls
        playsinline
        preload="metadata"
      ></video>
      <div v-if="!embeddedActive" class="link-source-player-hint">
        <q-icon name="ondemand_video" size="34px" color="indigo-4" />
        <span>视频链接 / 分片播放的内容会显示在这里</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useQuasar } from 'quasar';
import { useClipboard } from '@vueuse/core';
import {
  LINK_TABS,
  useLinkPlayback,
  type HlsSegment,
  type LinkTab,
} from 'src/composables/useLinkPlayback';
import { useSystemProperty } from 'src/stores/System';

// 外部链接面板（视频直链 / HLS 分片链，磁力链可选）
// 业务逻辑全部来自 useLinkPlayback，本组件只负责 UI 与「播放落到哪里」：
//   1. 宿主提供 video 元素（沉浸式播放页）→ onPlay / getVideoEl / getVolume 由宿主注入；
//   2. 宿主不提供（批量编辑弹窗）→ embedded 打开自带播放器，自己接管播放。

interface Props {
  /** 需要展示的链接类型 */
  tabs?: LinkTab[];
  /** 是否渲染并使用组件自带的内嵌播放器 */
  embedded?: boolean;
  /** 磁力链输入值（v-model:magnet-uri） */
  magnetUri?: string;
  /** 提交磁力链（解析种子） */
  submitMagnet?: () => void | Promise<void>;
  /** 宿主提供播放元素 */
  getVideoEl?: () => HTMLVideoElement | null;
  /** 宿主当前音量（0~1） */
  getVolume?: () => number;
  /** 宿主自行处理播放请求 */
  onPlay?: (src: string, name: string, isHls: boolean) => void;
}

const props = withDefaults(defineProps<Props>(), {
  // 默认只展示视频链接与分片链接（磁力链需宿主提供 magnet-uri / submit-magnet）
  tabs: () => ['video', 'hls'] as LinkTab[],
  embedded: false,
  magnetUri: '',
});

const emit = defineEmits<{
  (e: 'update:magnetUri', value: string): void;
}>();

const $q = useQuasar();
const systemProperty = useSystemProperty();
// 剪贴板：legacy 兜底——非安全上下文（http 局域网访问）下 navigator.clipboard 不存在
const { copy: copyText } = useClipboard({ legacy: true });

// ── 内嵌播放器 ────────────────────────────────────────────────────────────────
const internalVideoRef = ref<HTMLVideoElement | null>(null);
/** 内嵌播放器是否已有播放内容（无内容时显示提示遮罩） */
const embeddedActive = ref(false);

function getVideoEl(): HTMLVideoElement | null {
  return props.getVideoEl?.() ?? internalVideoRef.value;
}

function getVolume(): number {
  return props.getVolume?.() ?? internalVideoRef.value?.volume ?? 0.8;
}

/** 播放请求：宿主有 onPlay 交给宿主，否则写入内嵌播放器 */
function handlePlay(src: string, name: string, isHls: boolean) {
  if (props.onPlay) {
    props.onPlay(src, name, isHls);
    return;
  }
  const videoEl = internalVideoRef.value;
  if (!videoEl) return;
  videoEl.pause();
  videoEl.removeAttribute('src');
  videoEl.load();
  videoEl.title = name;
  embeddedActive.value = isHls || src.length > 0;
  // src 为空表示由 composable 内部的 hls.js 实例接管（attachMedia 处理）
  if (!src) return;
  videoEl.src = src;
  videoEl.play().catch((e: Error) => {
    console.warn('Autoplay blocked:', e.message);
  });
}

/** 磁力链输入框双向绑定（宿主未开放磁力链 tab 时不会被使用） */
const magnetModel = computed<string>({
  get: () => props.magnetUri,
  set: (value) => emit('update:magnetUri', value),
});

const {
  linkTab,
  linkFocused,
  activeLinkTab,
  activeLinkValue,
  canSubmitLink,
  hlsLoading,
  linkActionLabel,
  linkActionIcon,
  linkActionTooltip,
  linkActionLoading,
  hlsParsed,
  hlsSegments,
  hlsTotalCount,
  hlsKeptCount,
  hlsRemovedCount,
  hlsKeptDuration,
  hlsDownloadName,
  hlsDefaultDownloadName,
  hlsDownloadDir,
  hlsDownloadDirOptions,
  hlsDownloadList,
  hlsPlayingDownloadId,
  hlsDownloadMeta,
  switchLinkTab,
  submitLink,
  playHlsRemaining,
  removeHlsSegment,
  removeHlsSimilarSegments,
  restoreHlsSegments,
  downloadHls,
  cancelHlsDownload,
  chooseHlsDownloadDir,
  playHlsDownload,
  clearHlsDownloads,
  destroyHls,
  cleanup,
} = useLinkPlayback($q, {
  magnetURI: magnetModel,
  submitMagnet: () => props.submitMagnet?.(),
  getVideoEl,
  getVolume,
  onPlay: handlePlay,
  // 服务端可选保存目录：来自系统设置里的媒体目录
  getDownloadDirs: () => systemProperty.getSettingInfo?.Dirs ?? [],
});

// ── 可见的链接类型 ────────────────────────────────────────────────────────────
const visibleTabs = computed(() =>
  LINK_TABS.filter((item) => props.tabs.includes(item.value)),
);

// 宿主未开放上次记住的 tab（如弹窗里没有磁力链）时回退到第一个可用 tab
watch(
  visibleTabs,
  (list) => {
    const first = list[0];
    if (!first || list.some((item) => item.value === linkTab.value)) return;
    switchLinkTab(first.value);
  },
  { immediate: true },
);

// ── 分片链接复制 ──────────────────────────────────────────────────────────────
async function copySegmentUrl(seg: HlsSegment) {
  if (!seg?.url) return;
  await copyText(seg.url);
  $q.notify({
    type: 'positive',
    message: `已复制第 ${seg.index} 个分片链接`,
    position: 'top',
    timeout: 1200,
  });
}

async function copyAllSegmentUrls() {
  const urls = hlsSegments.value.map((seg) => seg.url);
  if (urls.length === 0) {
    $q.notify({
      type: 'warning',
      message: '没有可复制的分片链接',
      position: 'top',
    });
    return;
  }
  await copyText(urls.join('\n'));
  $q.notify({
    type: 'positive',
    message: `已复制 ${urls.length} 个分片链接`,
    position: 'top',
    timeout: 1500,
  });
}

// ── 停止播放 / 卸载 ───────────────────────────────────────────────────────────
/** 销毁 HLS 实例并清空播放地址（宿主切换资源 / 关闭弹窗时调用） */
function stopPlayback() {
  destroyHls();
  const videoEl = internalVideoRef.value;
  if (!videoEl) return;
  try {
    videoEl.pause();
  } catch {
    /* 忽略暂停异常 */
  }
  videoEl.removeAttribute('src');
  videoEl.load();
  embeddedActive.value = false;
}

onBeforeUnmount(() => {
  stopPlayback();
  cleanup();
});

defineExpose({ destroyHls, stopPlayback, cleanup });
</script>

<style scoped lang="scss">
.link-source-panel {
  width: 100%;
  /* 宿主容器更宽时（如弹窗）居中收窄，与播放器宽度对齐 */
  max-width: 900px;
  margin: 0 auto;
}

/* ── 内嵌播放器（批量编辑弹窗等宿主用） ─────────────────────────────────────── */
.link-source-player {
  position: relative;
  width: 100%;
  max-width: 900px;
  height: 42vh;
  min-height: 180px;
  margin: 12px auto 0;
  background: #06060c;
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 14px;
  overflow: hidden;
}

.link-source-video {
  display: block;
  width: 100%;
  height: 100%;
  background: #000;
}

.link-source-player-hint {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 16px;
  text-align: center;
  font-size: 0.78rem;
  color: rgba(165, 148, 249, 0.7);
  background: rgba(8, 8, 16, 0.94);
}

/* ── 链接类型 Tab ──────────────────────────────────────────────────────────── */
.link-tabs {
  display: flex;
  justify-content: center;
  gap: 6px;
  margin-bottom: 10px;
}

.link-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 14px;
  font-family: inherit;
  font-size: 0.78rem;
  white-space: nowrap;
  color: rgba(196, 181, 253, 0.7);
  background: rgba(12, 12, 24, 0.6);
  border: 1px solid rgba(99, 102, 241, 0.22);
  border-radius: 20px;
  cursor: pointer;
  transition:
    color 0.2s,
    background 0.2s,
    border-color 0.2s,
    box-shadow 0.2s;
}

.link-tab:hover {
  color: #e0e7ff;
  background: rgba(99, 102, 241, 0.16);
  border-color: rgba(99, 102, 241, 0.45);
}

.link-tab-active {
  color: #fff;
  background: linear-gradient(
    135deg,
    rgba(99, 102, 241, 0.5),
    rgba(139, 92, 246, 0.45)
  );
  border-color: rgba(139, 92, 246, 0.7);
  box-shadow: 0 0 14px rgba(99, 102, 241, 0.35);
}

/* ── 链接输入框 ────────────────────────────────────────────────────────────── */
.magnet-input-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(12, 12, 24, 0.75);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 40px;
  padding: 8px 8px 8px 18px;
  transition:
    border-color 0.3s,
    box-shadow 0.3s;
}

.magnet-input-wrapper.magnet-focused {
  border-color: rgba(139, 92, 246, 0.65);
  box-shadow:
    0 0 0 3px rgba(99, 102, 241, 0.12),
    0 0 30px rgba(99, 102, 241, 0.18);
}

.magnet-icon {
  flex-shrink: 0;
  opacity: 0.8;
}

.magnet-input {
  flex: 1;
}

.magnet-input :deep(.q-field__control) {
  background: transparent;
  border: none;
}

.magnet-input :deep(.q-field__native) {
  color: #c4b5fd;
  font-size: 0.88rem;
}

.magnet-input :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.38);
}

.magnet-submit-btn {
  flex-shrink: 0;
  padding: 0 14px;
  border-radius: 24px;
  background: rgba(99, 102, 241, 0.2);
  border: 1px solid rgba(99, 102, 241, 0.4);
  transition:
    background 0.2s,
    box-shadow 0.2s;
}

.magnet-submit-btn:hover:not([disabled]) {
  background: rgba(99, 102, 241, 0.4);
  box-shadow: 0 0 16px rgba(99, 102, 241, 0.4);
}

/* ── 分片列表（解析后可删除广告分片） ──────────────────────────────────────── */
.hls-segment-panel {
  margin-top: 10px;
  background: rgba(12, 12, 24, 0.85);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(99, 102, 241, 0.28);
  border-radius: 14px;
  overflow: hidden;
}

.hls-segment-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
}

/* 下载设置行（文件名 + 目录 + 下载按钮）：独立一行，文件名自适应剩余宽度以撑满整行 */
.hls-download-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
  background: rgba(99, 102, 241, 0.05);
}

.hls-play-btn {
  flex-shrink: 0;
  min-height: 24px;
  padding: 0 10px;
  font-size: 0.74rem;
}

.hls-segment-summary {
  font-size: 0.76rem;
  color: rgba(196, 181, 253, 0.85);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 自定义下载文件名输入框：独占剩余宽度，把同行按钮顶到右边 */
.hls-filename-input {
  flex: 1 1 auto;
  min-width: 120px;
}

/* 下载按钮：固定宽度不参与压缩，保证与目录/文件名同排 */
.hls-download-btn {
  flex-shrink: 0;
  min-height: 24px;
  padding: 0 10px;
  font-size: 0.74rem;
}

/* 「下载目录」按钮：目录名可能较长，超出省略 */
.hls-dir-btn {
  flex-shrink: 0;
  min-height: 24px;
  max-width: 150px;
  padding: 0 8px;
  font-size: 0.74rem;
}

.hls-dir-btn :deep(.q-btn__content) {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

/* 下载目录下拉菜单：路径可能较长，限制宽度并允许折行 */
.hls-dir-menu {
  max-width: 320px;
  max-height: 260px;
  overflow-y: auto;
}

.hls-dir-menu-label {
  word-break: break-all;
  font-size: 0.76rem;
}

.hls-filename-input :deep(.q-field__control) {
  height: 26px;
  min-height: 26px;
  padding: 0 6px;
  border-radius: 8px;
  background: rgba(99, 102, 241, 0.12);
}

.hls-filename-input :deep(.q-field__native) {
  font-size: 0.74rem;
  color: rgba(224, 231, 255, 0.95);
  padding: 0;
}

.hls-filename-input :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.45);
}

.hls-filename-input :deep(.q-field__prepend) {
  padding-right: 4px;
}

.hls-segment-list {
  max-height: 220px;
  overflow-y: auto;
  padding: 4px 0;
}

.hls-segment-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 3px 10px;
  font-size: 0.74rem;
  color: rgba(196, 181, 253, 0.75);
  transition: background 0.15s;
}

.hls-segment-item:hover {
  background: rgba(99, 102, 241, 0.12);
}

.hls-segment-index {
  flex-shrink: 0;
  width: 42px;
  color: rgba(129, 140, 248, 0.7);
  font-variant-numeric: tabular-nums;
}

.hls-segment-duration {
  flex-shrink: 0;
  width: 52px;
  color: rgba(165, 148, 249, 0.8);
  font-variant-numeric: tabular-nums;
}

.hls-segment-url {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
  cursor: pointer;
  transition: color 0.15s;
}

.hls-segment-url:hover {
  color: rgba(165, 180, 252, 1);
  text-decoration: underline;
}

/* ── 下载列表（已完成的分片视频，可回放） ──────────────────────────────────── */
.hls-download-list-panel {
  border-top: 1px solid rgba(99, 102, 241, 0.22);
  background: rgba(16, 32, 24, 0.35);
}

.hls-download-list-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid rgba(99, 102, 241, 0.16);
}

.hls-download-list-title {
  font-size: 0.76rem;
  color: rgba(134, 239, 172, 0.9);
  white-space: nowrap;
}

.hls-download-list {
  max-height: 180px;
  overflow-y: auto;
  padding: 4px 0;
}

.hls-download-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  font-size: 0.74rem;
  color: rgba(196, 181, 253, 0.8);
  transition: background 0.15s;
}

.hls-download-item:hover {
  background: rgba(34, 197, 94, 0.1);
}

/* 当前正在回放的条目：高亮提示，避免误播 */
.hls-download-item-playing {
  background: rgba(34, 197, 94, 0.14);
}

.hls-download-item-icon {
  flex-shrink: 0;
}

.hls-download-item-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.hls-download-item-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: rgba(224, 231, 255, 0.95);
}

.hls-download-item-meta {
  font-size: 0.68rem;
  color: rgba(148, 163, 184, 0.85);
  font-variant-numeric: tabular-nums;
}

/* 下载中的进度条：贴在文件名 / 进度文本下方 */
.hls-download-item-bar {
  margin-top: 3px;
  border-radius: 2px;
}

@media (max-width: 768px) {
  .link-source-player {
    height: 32vh;
    min-height: 140px;
  }

  .hls-segment-list {
    max-height: 150px;
  }

  .hls-segment-url {
    display: none;
  }

  .link-tabs {
    gap: 4px;
  }

  .link-tab {
    padding: 4px 10px;
    font-size: 0.72rem;
  }
}
</style>
