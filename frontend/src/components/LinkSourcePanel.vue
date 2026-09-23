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
      :class="{
        'magnet-focused': linkFocused,
        'magnet-input-multi': linkTab === 'hls',
      }"
    >
      <q-icon
        :name="activeLinkTab.icon"
        color="indigo-4"
        size="20px"
        class="magnet-icon"
      />
      <!-- 分片链接：多行输入，每行一个地址（带排序号，输入后自动生成下一行） -->
      <div v-if="linkTab === 'hls'" class="hls-url-rows">
        <div
          v-for="(row, index) in hlsUrlRows"
          :key="row.id"
          class="hls-url-row"
        >
          <span class="hls-url-order">{{ index + 1 }}</span>
          <q-input
            :model-value="row.value"
            :placeholder="
              index === 0
                ? activeLinkTab.placeholder
                : '继续输入或粘贴 m3u8 地址（可一次粘贴多个）'
            "
            dark
            dense
            borderless
            class="hls-url-field"
            :disable="hlsParsing"
            @update:model-value="
              (val) => updateHlsUrlRow(row.id, String(val ?? ''))
            "
            @keyup.enter="submitLink"
            @focus="linkFocused = true"
            @blur="linkFocused = false"
          />
        </div>
      </div>
      <q-input
        v-else
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
        v-if="linkTab === 'hls' && canSubmitLink"
        flat
        dense
        round
        size="sm"
        color="indigo-4"
        icon="backspace"
        :disable="hlsParsing"
        @click="clearHlsUrlRows"
        class="hls-url-clear-btn"
      >
        <q-tooltip class="bg-dark text-white">清空全部地址</q-tooltip>
      </q-btn>
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
              <template v-if="hlsSourceCount > 1"
                >{{ hlsSourceCount }} 个源 · </template
              >保留 {{ hlsKeptCount }}/{{ hlsTotalCount }} 个分片 ·
              {{ hlsKeptDuration }}
              <template v-if="hlsRemovedCount"
                >（已删除 {{ hlsRemovedCount }}）</template
              >
              <template v-if="hlsAdCount"
                >（广告 {{ hlsAdCount }}，已排除）</template
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
                >恢复全部已删除分片（广告仍保持置灰并排除）</q-tooltip
              >
            </q-btn>
            <!-- 广告黑名单：可查看 / 取消单条 / 清空 -->
            <q-btn
              v-if="hlsAdBlacklist.length"
              flat
              dense
              no-caps
              size="sm"
              color="negative"
              icon="block"
              :label="`广告黑名单 ${hlsAdBlacklist.length}`"
            >
              <q-tooltip class="bg-dark text-white"
                >被标记为广告的分片类：列表中置灰，下载 / 播放时忽略</q-tooltip
              >
              <q-menu dark class="hls-ad-menu">
                <q-list dense>
                  <q-item
                    v-for="key in hlsAdBlacklist"
                    :key="key"
                    clickable
                    v-close-popup
                    @click="unmarkHlsAdSegments(key)"
                  >
                    <q-item-section>
                      <q-item-label class="hls-ad-menu-label">{{
                        key
                      }}</q-item-label>
                    </q-item-section>
                    <q-item-section side>
                      <q-icon name="undo" size="16px" />
                    </q-item-section>
                  </q-item>
                  <q-separator dark />
                  <q-item clickable v-close-popup @click="clearHlsAdBlacklist">
                    <q-item-section>
                      <q-item-label>清空黑名单</q-item-label>
                    </q-item-section>
                    <q-item-section side>
                      <q-icon name="delete_forever" size="16px" />
                    </q-item-section>
                  </q-item>
                </q-list>
              </q-menu>
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
              <q-tooltip class="bg-dark text-white">{{
                hlsSourceCount > 1
                  ? `按顺序合并播放 ${hlsSourceCount} 个源的 ${hlsKeptCount} 个分片`
                  : `播放剩余 ${hlsKeptCount} 个分片`
              }}</q-tooltip>
            </q-btn>
          </div>

          <!-- 源列表：多个 m3u8 按这里显示的顺序合并，可上移 / 下移 / 移除 -->
          <div v-if="hlsSourceList.length > 1" class="hls-source-list">
            <div
              v-for="(item, index) in hlsSourceList"
              :key="item.source.id"
              class="hls-source-item"
            >
              <span class="hls-source-order">{{ index + 1 }}</span>
              <span class="hls-source-url" :title="item.source.url">{{
                item.source.url
              }}</span>
              <span class="hls-source-meta"
                >{{ item.keptCount }}/{{ item.totalCount }} 个分片 ·
                {{ item.duration }}</span
              >
              <q-btn
                flat
                round
                dense
                size="sm"
                color="indigo-4"
                icon="arrow_upward"
                :disable="index === 0"
                @click="moveHlsSource(item.source.id, -1)"
              >
                <q-tooltip class="bg-dark text-white">上移</q-tooltip>
              </q-btn>
              <q-btn
                flat
                round
                dense
                size="sm"
                color="indigo-4"
                icon="arrow_downward"
                :disable="index === hlsSourceList.length - 1"
                @click="moveHlsSource(item.source.id, 1)"
              >
                <q-tooltip class="bg-dark text-white">下移</q-tooltip>
              </q-btn>
              <q-btn
                flat
                round
                dense
                size="sm"
                color="red-4"
                icon="delete_outline"
                @click="removeHlsSource(item.source.id)"
              >
                <q-tooltip class="bg-dark text-white"
                  >移除该源及其分片</q-tooltip
                >
              </q-btn>
            </div>
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
                <q-item
                  clickable
                  v-close-popup
                  @click="chooseHlsDownloadDir('')"
                >
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
            <!-- 点下载后任务交给服务端执行，本地分片列表随之清空，可继续粘贴下一组地址 -->
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
            <!-- 浏览器直下（前端备用）：放在服务端下载按钮旁边，不经服务端直接拉当前分片列表合并另存 -->
            <q-btn
              flat
              dense
              no-caps
              size="sm"
              color="deep-orange-4"
              icon="cloud_download"
              :label="browserNowLabel"
              class="hls-download-btn"
              :disable="!hlsKeptCount || browserNowInProgress()"
              @click="downloadHlsInBrowserNow"
            >
              <q-tooltip class="bg-dark text-white"
                >浏览器直接下载当前 {{ hlsKeptCount }} 个分片并合并另存，不经服务端；源站禁止跨域时失败，此为本页关闭后下载即中断</q-tooltip
              >
            </q-btn>
          </div>
          <div class="hls-segment-list">
            <template
              v-for="(group, groupIndex) in hlsSegmentGroups"
              :key="group.sourceId"
            >
              <!-- 多源时标出每段的边界，便于分辨删除的是哪一个源的分片 -->
              <div v-if="hlsSourceCount > 1" class="hls-segment-group">
                <span class="hls-segment-group-label"
                  >源 {{ groupIndex + 1 }}</span
                >
                <span class="hls-segment-group-url" :title="group.url">{{
                  group.url
                }}</span>
                <span class="hls-segment-group-count"
                  >{{ group.segments.length }} 个分片</span
                >
              </div>
              <!-- 黑名单（广告）分片仍在列表中，但置灰 + 未勾选，下载 / 播放时忽略 -->
              <div
                v-for="seg in group.segments"
                :key="seg.id"
                class="hls-segment-item"
                :class="{ 'hls-segment-item--ad': isHlsAdSegment(seg) }"
              >
                <q-icon
                  class="hls-segment-check"
                  :name="
                    isHlsAdSegment(seg)
                      ? 'check_box_outline_blank'
                      : 'check_box'
                  "
                  size="16px"
                  :color="isHlsAdSegment(seg) ? 'grey' : 'indigo-4'"
                >
                  <q-tooltip class="bg-dark text-white">{{
                    isHlsAdSegment(seg)
                      ? '广告（黑名单）：不参与播放 / 下载'
                      : '参与播放 / 下载'
                  }}</q-tooltip>
                </q-icon>
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
                  <q-tooltip class="bg-dark text-white"
                    >复制该分片链接</q-tooltip
                  >
                </q-btn>
                <!-- 标记 / 取消广告：同类分片进黑名单，列表置灰且下载时忽略 -->
                <q-btn
                  flat
                  round
                  dense
                  size="sm"
                  :color="isHlsAdSegment(seg) ? 'grey' : 'negative'"
                  :icon="isHlsAdSegment(seg) ? 'undo' : 'block'"
                  @click="toggleHlsAdSegment(seg.id)"
                >
                  <q-tooltip class="bg-dark text-white">{{
                    isHlsAdSegment(seg)
                      ? '取消广告标记：该类分片恢复参与播放 / 下载'
                      : '标记为广告：同类分片加入黑名单（列表置灰，下载时忽略）'
                  }}</q-tooltip>
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
            </template>
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
                  v-if="item.status === 'downloading' || browserDownloadInProgress(item)"
                  :value="
                    item.status === 'downloading'
                      ? item.progress / 100
                      : (browserProgressRatio(item) ?? 0)
                  "
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
              <!-- 重启：失败/已取消的任务复用服务端保存的播放列表重新下载 -->
              <q-btn
                v-if="item.status === 'failed' || item.status === 'canceled'"
                flat
                round
                dense
                size="sm"
                color="orange-4"
                icon="restart_alt"
                @click="restartHlsDownload(item.id)"
              >
                <q-tooltip class="bg-dark text-white"
                  >重新启动该下载任务</q-tooltip
                >
              </q-btn>
              <!-- 浏览器直下（前端备用）：浏览器直接从源站拉分片、解密合并另存，不经服务端 -->
              <q-btn
                flat
                round
                dense
                size="sm"
                color="indigo-4"
                icon="file_download"
                :disable="browserDownloadInProgress(item)"
                @click="downloadHlsInBrowser(item)"
              >
                <q-tooltip class="bg-dark text-white">
                  浏览器直接下载（前端备用，不经服务端；源站禁止跨域时失败）
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
  hlsUrlRows,
  updateHlsUrlRow,
  clearHlsUrlRows,
  hlsLoading,
  linkActionLabel,
  linkActionIcon,
  linkActionTooltip,
  linkActionLoading,
  hlsParsed,
  hlsSegments,
  hlsSegmentGroups,
  hlsSourceList,
  hlsSourceCount,
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
  hlsAdBlacklist,
  hlsAdCount,
  isHlsAdSegment,
  unmarkHlsAdSegments,
  toggleHlsAdSegment,
  clearHlsAdBlacklist,
  removeHlsSource,
  moveHlsSource,
  downloadHls,
  cancelHlsDownload,
  restartHlsDownload,
  downloadHlsInBrowser,
  downloadHlsInBrowserNow,
  browserDownloadInProgress,
  browserNowInProgress,
  browserNowProgressRatio,
  browserProgressRatio,
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

/** 浏览器直下按钮文案：未进行时为固定文案，进行中显示实时百分比 */
const browserNowLabel = computed(() => {
  const r = browserNowProgressRatio();
  return r == null ? '浏览器直下' : `直下中 ${Math.round(r * 100)}%`;
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
  const urls = hlsSegments.value
    .filter((seg) => !isHlsAdSegment(seg))
    .map((seg) => seg.url);
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

/* 分片链接多行输入：胶囊形改圆角矩形，图标 / 按钮与首行对齐 */
.magnet-input-wrapper.magnet-input-multi {
  align-items: flex-start;
  border-radius: 18px;
  padding: 10px 8px 10px 14px;
}

.magnet-input-wrapper.magnet-input-multi .magnet-icon {
  margin-top: 6px;
}

.magnet-input-wrapper.magnet-input-multi .magnet-submit-btn {
  margin-top: 2px;
}

.hls-url-rows {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  /* 地址较多时内部滚动，输入框高度不无限增长 */
  max-height: 112px;
  overflow-y: auto;
  padding: 2px 0;
}

.hls-url-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.hls-url-field {
  flex: 1;
  min-width: 0;
}

.hls-url-field :deep(.q-field__control) {
  background: transparent;
  border: none;
  min-height: 26px;
}

.hls-url-field :deep(.q-field__native) {
  color: #c4b5fd;
  font-size: 0.86rem;
}

.hls-url-field :deep(.q-field__native::placeholder) {
  color: rgba(165, 148, 249, 0.38);
}

.hls-url-clear-btn {
  flex-shrink: 0;
  margin-top: 2px;
  opacity: 0.75;
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

/* 广告黑名单菜单：URL 前缀较长，限宽并允许折行 */
.hls-ad-menu {
  max-width: 360px;
  max-height: 300px;
  overflow-y: auto;
}

.hls-ad-menu-label {
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

/* ── 源列表（多个 m3u8 的合并顺序） ────────────────────────────────────────── */
.hls-source-list {
  max-height: 132px;
  overflow-y: auto;
  padding: 4px 0;
  border-bottom: 1px solid rgba(99, 102, 241, 0.18);
  background: rgba(139, 92, 246, 0.06);
}

.hls-source-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 3px 10px;
  font-size: 0.74rem;
  color: rgba(196, 181, 253, 0.75);
  transition: background 0.15s;
}

.hls-source-item:hover {
  background: rgba(139, 92, 246, 0.14);
}

.hls-source-order,
.hls-url-order {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  font-size: 0.66rem;
  color: #e0e7ff;
  background: rgba(99, 102, 241, 0.35);
  border-radius: 50%;
  font-variant-numeric: tabular-nums;
}

.hls-source-url {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
}

.hls-source-meta {
  flex-shrink: 0;
  color: rgba(165, 148, 249, 0.8);
  font-variant-numeric: tabular-nums;
}

/* ── 分片列表里的源分组标题 ────────────────────────────────────────────────── */
.hls-segment-group {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  padding: 2px 10px;
  font-size: 0.68rem;
  color: rgba(148, 163, 184, 0.9);
  background: rgba(99, 102, 241, 0.08);
}

.hls-segment-group-label {
  flex-shrink: 0;
  color: rgba(129, 140, 248, 0.95);
}

.hls-segment-group-url {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
}

.hls-segment-group-count {
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
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

/* 广告（黑名单）分片：仍在列表中，但置灰 + 未勾选，表示不参与播放 / 下载 */
.hls-segment-item--ad {
  opacity: 0.45;
  filter: grayscale(1);
}

.hls-segment-item--ad .hls-segment-url {
  text-decoration: line-through;
  cursor: default;
}

.hls-segment-item--ad:hover {
  background: rgba(148, 163, 184, 0.1);
}

/* 勾选状态图标：不参与下载的分片显示未勾选框 */
.hls-segment-check {
  flex-shrink: 0;
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

  /* 小屏下源列表同样收窄，避免挤掉分片列表 */
  .hls-source-list {
    max-height: 88px;
  }

  .hls-url-rows {
    max-height: 88px;
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
