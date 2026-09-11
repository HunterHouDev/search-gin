import { computed, ref, watch, type Ref } from 'vue';
import type { QVueGlobals } from 'quasar';
import type HlsJs from 'hls.js';
import {
  DelTransferTasksInfo,
  HlsCancelAPI,
  HlsDownloadAPI,
  TransferTasksInfo,
} from 'src/components/api/searchAPI';

// 外部链接播放逻辑（磁力链 / 视频链接 / 分片链接）
// 磁力链复用 useTorrentDownload，视频直链与 HLS 分片链在此处理。
// 分片链支持：解析 m3u8 → 列出分片 → 删除指定分片（如广告）→ 播放剩余分片。
// 下载：交由服务端异步任务执行（/api/hlsDownload），前端只提交与同步进度，
//       因此关闭弹窗、刷新页面、甚至关掉浏览器都不会中断下载。

export type LinkTab = 'magnet' | 'video' | 'hls';

export interface LinkTabItem {
  value: LinkTab;
  label: string;
  icon: string;
  placeholder: string;
  tooltip: string;
}

export const LINK_TABS: LinkTabItem[] = [
  {
    value: 'magnet',
    label: '磁力链',
    icon: 'link',
    placeholder: '粘贴磁力链 magnet:?xt=urn:btih:...',
    tooltip: '解析磁力链并选择文件',
  },
  {
    value: 'video',
    label: '视频链接',
    icon: 'movie',
    placeholder: '粘贴视频直链 https://.../video.mp4',
    tooltip: '播放视频链接',
  },
  {
    value: 'hls',
    label: '分片链接',
    icon: 'playlist_play',
    placeholder: '粘贴分片链接 https://.../index.m3u8',
    tooltip: '解析分片列表',
  },
];

/** 链接类型 Tab 的本地存储 key（刷新后保留上次选择） */
const LINK_TAB_STORAGE_KEY = 'immersive.linkTab';

/** 全部链接类型值，宿主可按需筛选展示（如弹窗只保留视频 / 分片链接） */
export const LINK_TAB_VALUES: LinkTab[] = LINK_TABS.map((item) => item.value);

/** 读取上次选择的 tab；脏数据或已下线的 tab 回退到默认「磁力链」 */
function readStoredLinkTab(): LinkTab {
  try {
    const saved = localStorage.getItem(LINK_TAB_STORAGE_KEY);
    if (saved && LINK_TAB_VALUES.includes(saved as LinkTab))
      return saved as LinkTab;
  } catch {
    // 隐私模式 / 存储被禁用时忽略
  }
  return 'magnet';
}

/** 读取上次选择的服务端下载目录；空串表示使用服务端默认目录 */
function readStoredDownloadDir(): string {
  try {
    return localStorage.getItem(DOWNLOAD_DIR_STORAGE_KEY) ?? '';
  } catch {
    return '';
  }
}

/** 解析出的单个 HLS 分片 */
export interface HlsSegment {
  /** 稳定 id（等于原始序号） */
  id: number;
  /** 原始序号，从 1 开始 */
  index: number;
  /** #EXTINF 时长（秒），无则为 0 */
  duration: number;
  /** 绝对地址 */
  url: string;
  /** 该分片前的段级标签（#EXT-X-DISCONTINUITY / #EXT-X-BYTERANGE 等） */
  tags: string[];
  /** #EXTINF 原始行，无则为 null */
  extinf: string | null;
  /** #EXT-X-BYTERANGE 解析结果，null 表示整个资源就是一个分片 */
  byteRange: { length: number; offset: number } | null;
}

interface ParsedPlaylist {
  header: string[];
  segments: HlsSegment[];
  hasEndList: boolean;
}

/** #EXT-X-KEY 的加密方式 */
type KeyMethod = 'AES-128' | 'SAMPLE-AES' | 'NONE';

/** 明确底层为 ArrayBuffer 的字节数组（WebCrypto / File System Access API 的要求） */
type Bytes = Uint8Array<ArrayBuffer>;

interface SegmentKey {
  method: KeyMethod;
  /** 密钥地址（已绝对化） */
  url: string;
  /** 显式 IV，null 表示用分片序号推导 */
  iv: Bytes | null;
}

/** 下载任务状态：下载中 / 已完成 / 已取消 / 失败 */
export type HlsDownloadStatus = 'downloading' | 'done' | 'canceled' | 'failed';

/** 服务端返回的传输任务（此处只用到分片下载相关字段） */
interface HlsServerTask {
  ID: string;
  Type: string;
  Name: string;
  /** 服务端保存的完整路径 */
  Path: string;
  URL: string;
  Segments: number;
  TotalSegments: number;
  Progress: number;
  /** 已写入字节数 */
  Size: number;
  Duration: string;
  Status: string;
  Log: string;
  CreateTime: string;
  FinishTime?: string;
}

/** 服务端分片下载任务的 Type 值，与后端 TaskTypeHls 保持一致 */
const HLS_TASK_TYPE = '分片下载';

/** 服务端任务状态 → 下载列表状态 */
function mapTaskStatus(status: string): HlsDownloadStatus {
  if (status === '执行中' || status === '等待') return 'downloading';
  if (status === '完成') return 'done';
  if (status === '取消') return 'canceled';
  return 'failed';
}

/** 下载列表中的一条记录：来自服务端任务列表，关闭弹窗 / 刷新页面后依旧存在 */
export interface HlsDownloadItem {
  /** 服务端任务 ID */
  id: string;
  /** 保存到本地的文件名 */
  name: string;
  /** 服务端保存的完整路径（用于页面内回放） */
  path: string;
  /** 已写入的分片数（下载中实时增长，完成即分片总数） */
  segmentCount: number;
  /** 本次下载的分片总数 */
  totalCount: number;
  /** 进度百分比（0~100） */
  progress: number;
  /** 任务状态 */
  status: HlsDownloadStatus;
  /** 本次下载分片的总时长文本 */
  duration: string;
  /** 写入字节数的人类可读文本，未完成时为空串 */
  sizeText: string;
  /** 落地位置 / 失败原因说明 */
  target: string;
  /** 创建时间戳 */
  createdAt: number;
  /** 是否可在页面内回放（已完成且路径可访问） */
  playable: boolean;
  /** 来源播放列表地址，便于区分不同链接的下载 */
  sourceUrl: string;
}

export interface LinkPlaybackOptions {
  /** 磁力链输入框绑定（来自 useTorrentDownload） */
  magnetURI: Ref<string>;
  /** 提交磁力链（来自 useTorrentDownload） */
  submitMagnet: () => void | Promise<void>;
  /** 获取 video DOM 元素 */
  getVideoEl: () => HTMLVideoElement | null;
  /** 获取当前音量（0~1） */
  getVolume: () => number;
  /** 通知页面开始播放：src 为空表示由 HLS 实例接管 */
  onPlay: (src: string, name: string, isHls: boolean) => void;
  /** 可选：服务端可选的下载目录列表（来自系统设置里的媒体目录） */
  getDownloadDirs?: () => string[];
}

const HTTP_URL_RE = /^https?:\/\//i;
const HLS_URL_RE = /\.m3u8(\?|#|$)/i;
const M3U8_MIME = 'application/vnd.apple.mpegurl';
/** 记住用户选择的服务端下载目录 */
const DOWNLOAD_DIR_STORAGE_KEY = 'immersive.serverDownloadDir';
/** 下载中任务的轮询间隔（毫秒） */
const DOWNLOAD_POLL_INTERVAL = 2000;

/** 媒体播放列表的全局标签：重建播放列表时原样保留在文件头 */
const GLOBAL_TAGS = [
  '#EXTM3U',
  '#EXT-X-VERSION',
  '#EXT-X-TARGETDURATION',
  '#EXT-X-MEDIA-SEQUENCE',
  '#EXT-X-DISCONTINUITY-SEQUENCE',
  '#EXT-X-PLAYLIST-TYPE',
  '#EXT-X-KEY',
  '#EXT-X-SESSION-KEY',
  '#EXT-X-MAP',
  '#EXT-X-MEDIA',
  '#EXT-X-START',
  '#EXT-X-DEFINE',
  '#EXT-X-INDEPENDENT-SEGMENTS',
  '#EXT-X-I-FRAMES-ONLY',
];

function isHttpUrl(url: string): boolean {
  return HTTP_URL_RE.test(url);
}

function fileNameFromUrl(url: string): string {
  try {
    const parsed = new URL(url);
    const base = parsed.pathname.split('/').filter(Boolean).pop();
    return base ? decodeURIComponent(base) : parsed.hostname;
  } catch {
    return '网络视频';
  }
}

/** 相对地址转绝对地址（分片、密钥、初始化段都需要） */
function toAbsoluteUrl(uri: string, baseUrl: string): string {
  try {
    return new URL(uri, baseUrl).href;
  } catch {
    return uri;
  }
}

/** 把标签行内的 URI="..." 属性转为绝对地址（#EXT-X-KEY / #EXT-X-MAP / #EXT-X-MEDIA） */
function absolutizeTagUris(line: string, baseUrl: string): string {
  return line.replace(
    /URI="([^"]*)"/gi,
    (_match, uri: string) => `URI="${toAbsoluteUrl(uri, baseUrl)}"`,
  );
}

function parseExtinfDuration(line: string | null): number {
  if (!line) return 0;
  const matched = /^#EXTINF:\s*([\d.]+)/.exec(line);
  return matched ? Number(matched[1]) || 0 : 0;
}

/** 解析标签的 KEY=VALUE 属性（值可能是带引号的字符串） */
function parseAttributes(input: string): Record<string, string> {
  const attrs: Record<string, string> = {};
  const re = /([A-Za-z0-9-]+)=("[^"]*"|[^,]*)/g;
  let matched = re.exec(input);
  while (matched) {
    attrs[matched[1].toUpperCase()] = matched[2].replace(/^"|"$/g, '');
    matched = re.exec(input);
  }
  return attrs;
}

/** 解析 #EXT-X-BYTERANGE:<长度>[@<偏移>]，偏移省略时接在上一个子区间之后 */
function parseByteRangeValue(
  value: string | undefined,
  previousEnd: number,
): { length: number; offset: number } | null {
  if (!value) return null;
  const [lengthText, offsetText] = value.split('@');
  const length = Number(lengthText);
  if (!Number.isFinite(length) || length <= 0) return null;
  const offset = offsetText === undefined ? previousEnd : Number(offsetText);
  if (!Number.isFinite(offset) || offset < 0) return null;
  return { length, offset };
}

/** 从标签行取冒号后的属性串 */
function tagAttributes(line: string): Record<string, string> {
  const colon = line.indexOf(':');
  return parseAttributes(colon >= 0 ? line.slice(colon + 1) : '');
}

function hexToBytes(hex: string): Bytes | null {
  const clean = hex.trim().replace(/^0x/i, '');
  if (clean.length === 0 || clean.length % 2 !== 0) return null;
  if (!/^[0-9a-f]+$/i.test(clean)) return null;
  const bytes = new Uint8Array(clean.length / 2);
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = Number.parseInt(clean.slice(i * 2, i * 2 + 2), 16);
  }
  return bytes;
}

function parseKeyTag(line: string): SegmentKey | null {
  const attrs = tagAttributes(line);
  const rawMethod = (attrs.METHOD || 'NONE').toUpperCase();
  if (rawMethod === 'NONE') return null;
  return {
    method: rawMethod === 'AES-128' ? 'AES-128' : 'SAMPLE-AES',
    url: attrs.URI || '',
    iv: attrs.IV ? hexToBytes(attrs.IV) : null,
  };
}

/** 逐个分片解析生效的密钥（#EXT-X-KEY 可出现在文件头，也可出现在分片之前） */
function resolveSegmentKeys(parsed: ParsedPlaylist): (SegmentKey | null)[] {
  let current: SegmentKey | null = null;
  for (const line of parsed.header) {
    if (line.startsWith('#EXT-X-KEY')) current = parseKeyTag(line);
  }
  const keys: (SegmentKey | null)[] = [];
  for (const segment of parsed.segments) {
    for (const tag of segment.tags) {
      if (tag.startsWith('#EXT-X-KEY')) current = parseKeyTag(tag);
    }
    keys.push(current);
  }
  return keys;
}

/** m3u8 文件名换成下载用的文件名（去掉 .m3u8 后缀，换成实际容器后缀） */
function buildDownloadName(source: string, ext: string): string {
  const stem = source.replace(/\.[^./\\]+$/, '') || 'video';
  return `${stem}.${ext}`;
}

/** 文件名中不允许出现的字符（按 Windows 规则处理，跨平台都安全） */
const INVALID_FILENAME_CHARS = /[\\/:*?"<>|]/g;

/** 已自带扩展名（如 .mp4 / .ts / .m4s）时不再追加容器后缀 */
const HAS_EXTENSION_RE = /\.[A-Za-z0-9]{1,8}$/;

/** 文件名长度上限，避免超出文件系统限制 */
const MAX_FILENAME_LENGTH = 120;

/**
 * 把用户在输入框里填的名字整理成可用的下载文件名：
 * 清洗非法字符与首尾空白/点，留空时回退到默认名，未写扩展名时补上容器后缀。
 */
function resolveDownloadName(
  raw: string,
  fallback: string,
  ext: string,
): string {
  const cleaned = raw
    .replace(INVALID_FILENAME_CHARS, '_')
    .replace(/\s+/g, ' ')
    .replace(/^[.\s]+/, '')
    .replace(/[.\s]+$/, '')
    .slice(0, MAX_FILENAME_LENGTH);
  if (!cleaned) return fallback;
  return HAS_EXTENSION_RE.test(cleaned) ? cleaned : `${cleaned}.${ext}`;
}

/**
 * 分片的「类」：忽略 query/hash 后，取最后一个 / 之前的前缀。
 * 前缀相同即同类——典型场景是正片与广告分属不同目录，只有最后一节文件名不同。
 */
function segmentClassKey(url: string): string {
  const path = url.split('#')[0].split('?')[0];
  const slash = path.lastIndexOf('/');
  return slash >= 0 ? path.slice(0, slash + 1) : path;
}

/** 解析 m3u8：分离全局头、分片（含段级标签），并把所有地址绝对化 */
function parsePlaylist(text: string, baseUrl: string): ParsedPlaylist {
  const lines = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line.length > 0);

  const header: string[] = [];
  const segments: HlsSegment[] = [];
  let pendingTags: string[] = [];
  let pendingExtinf: string | null = null;
  let pendingByteRange: string | null = null;
  let hasEndList = false;
  // 记录上一个子区间的结束位置：同一资源内 #EXT-X-BYTERANGE 省略偏移时接着往下切
  let lastRangeEnd = 0;
  let lastRangeUrl = '';

  for (const line of lines) {
    if (line.startsWith('#')) {
      if (line.startsWith('#EXTINF')) {
        pendingExtinf = line;
        continue;
      }
      if (line.startsWith('#EXT-X-ENDLIST')) {
        hasEndList = true;
        continue;
      }
      if (line.startsWith('#EXT-X-BYTERANGE')) {
        pendingByteRange = line;
        pendingTags.push(line);
        continue;
      }
      // #EXT-X-KEY 可能中途换密钥，位置必须保留：分片之后出现的挂到下一个分片上
      if (line.startsWith('#EXT-X-KEY')) {
        if (segments.length === 0)
          header.push(absolutizeTagUris(line, baseUrl));
        else pendingTags.push(absolutizeTagUris(line, baseUrl));
        continue;
      }
      if (GLOBAL_TAGS.some((tag) => line.startsWith(tag))) {
        header.push(absolutizeTagUris(line, baseUrl));
        continue;
      }
      // 段级标签（#EXT-X-DISCONTINUITY / #EXT-X-PROGRAM-DATE-TIME ...）
      pendingTags.push(absolutizeTagUris(line, baseUrl));
      continue;
    }

    // 非 # 开头即分片地址行
    const url = toAbsoluteUrl(line, baseUrl);
    const previousEnd = lastRangeUrl === url ? lastRangeEnd : 0;
    const byteRange = pendingByteRange
      ? parseByteRangeValue(
          pendingByteRange.slice(pendingByteRange.indexOf(':') + 1),
          previousEnd,
        )
      : null;
    if (byteRange) {
      lastRangeEnd = byteRange.offset + byteRange.length;
      lastRangeUrl = url;
    }

    segments.push({
      id: segments.length + 1,
      index: segments.length + 1,
      duration: parseExtinfDuration(pendingExtinf),
      url,
      tags: pendingTags,
      extinf: pendingExtinf,
      byteRange,
    });
    pendingTags = [];
    pendingExtinf = null;
    pendingByteRange = null;
  }

  return { header, segments, hasEndList };
}

/** 用保留下来的分片重建播放列表文本 */
function buildPlaylistText(
  parsed: ParsedPlaylist,
  segments: HlsSegment[],
): string {
  const out: string[] = [...parsed.header];
  for (const segment of segments) {
    out.push(...segment.tags);
    if (segment.extinf) out.push(segment.extinf);
    out.push(segment.url);
  }
  if (parsed.hasEndList) out.push('#EXT-X-ENDLIST');
  return out.join('\n') + '\n';
}

function formatSeconds(total: number): string {
  if (!Number.isFinite(total) || total <= 0) return '00:00';
  const seconds = Math.round(total);
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  const pad = (n: number) => String(n).padStart(2, '0');
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`;
}

/** 字节数转人类可读大小，用于下载列表展示 */
function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${unit === 0 || value >= 100 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`;
}

export function useLinkPlayback($q: QVueGlobals, opts: LinkPlaybackOptions) {
  const { magnetURI, submitMagnet, getVideoEl, getVolume, onPlay } = opts;

  const linkTab = ref<LinkTab>(readStoredLinkTab());
  const linkFocused = ref(false);

  // 记住用户选择的链接类型，刷新页面后不再回到默认 tab
  watch(linkTab, (val) => {
    try {
      localStorage.setItem(LINK_TAB_STORAGE_KEY, val);
    } catch {
      // 写入失败（隐私模式）时忽略，不影响功能
    }
  });

  const videoURL = ref('');
  const hlsURL = ref('');
  const hlsLoading = ref(false);
  const hlsParsing = ref(false);
  let hlsInstance: HlsJs | null = null;
  let blobUrl: string | null = null;

  // ── 下载列表（服务端任务镜像） ───────────────────────────────────────────────
  /**
   * 分片下载任务数据源为服务端任务列表，因此关闭弹窗、刷新页面后依旧存在。
   * 本组件只负责提交任务与同步进度。
   */
  const hlsDownloadList = ref<HlsDownloadItem[]>([]);
  /** 正在页面内回放的下载项 id，用于列表高亮 */
  const hlsPlayingDownloadId = ref('');
  /** 是否存在进行中的下载（决定是否轮询服务端进度） */
  const hlsDownloadActive = computed(() =>
    hlsDownloadList.value.some((item) => item.status === 'downloading'),
  );
  let downloadPollTimer: ReturnType<typeof setInterval> | null = null;

  // ── 分片解析状态 ────────────────────────────────────────────────────────────
  const parsedPlaylist = ref<ParsedPlaylist | null>(null);
  const hlsParsedUrl = ref('');
  /** 解析出的全量分片（用于「恢复」） */
  const hlsAllSegments = ref<HlsSegment[]>([]);
  /** 当前保留的分片（删除后即时减少） */
  const hlsSegments = ref<HlsSegment[]>([]);

  const hlsParsed = computed(() => parsedPlaylist.value !== null);
  const hlsTotalCount = computed(() => hlsAllSegments.value.length);
  const hlsKeptCount = computed(() => hlsSegments.value.length);
  const hlsRemovedCount = computed(
    () => hlsTotalCount.value - hlsKeptCount.value,
  );
  const hlsKeptDuration = computed(() =>
    formatSeconds(
      hlsSegments.value.reduce((sum, seg) => sum + seg.duration, 0),
    ),
  );

  // ── 下载文件名（留空则用默认名） ─────────────────────────────────────────────
  /** 用户自定义的保存文件名，空串表示使用默认名 */
  const hlsDownloadName = ref('');

  /** 下载容器后缀：带 #EXT-X-MAP 初始化段的播放列表实际是 fMP4 */
  const hlsDownloadExt = computed(() =>
    parsedPlaylist.value?.header.some((item) => item.startsWith('#EXT-X-MAP'))
      ? 'mp4'
      : 'ts',
  );

  /** 默认下载文件名（输入框留空时使用），随解析到的地址变化 */
  const hlsDefaultDownloadName = computed(() =>
    buildDownloadName(
      fileNameFromUrl(hlsParsedUrl.value || hlsURL.value),
      hlsDownloadExt.value,
    ),
  );

  // ── 服务端下载目录（留空则用服务端默认目录） ─────────────────────────────────
  /** 已选择的服务端保存目录，空串表示使用服务端默认目录 */
  const hlsDownloadDir = ref(readStoredDownloadDir());
  /** 可选的服务端目录（来自系统设置里的媒体目录） */
  const hlsDownloadDirOptions = computed(() => opts.getDownloadDirs?.() ?? []);

  // 记住用户的目录选择，刷新后继续沿用
  watch(hlsDownloadDir, (val) => {
    try {
      if (val) localStorage.setItem(DOWNLOAD_DIR_STORAGE_KEY, val);
      else localStorage.removeItem(DOWNLOAD_DIR_STORAGE_KEY);
    } catch {
      // 隐私模式写入失败时忽略
    }
  });

  /** 选择下载目录 */
  function chooseHlsDownloadDir(dir: string) {
    hlsDownloadDir.value = dir;
  }

  const activeLinkTab = computed<LinkTabItem>(
    () => LINK_TABS.find((t) => t.value === linkTab.value) ?? LINK_TABS[0],
  );

  const activeLinkValue = computed<string>({
    get() {
      if (linkTab.value === 'magnet') return magnetURI.value;
      if (linkTab.value === 'video') return videoURL.value;
      return hlsURL.value;
    },
    set(val: string) {
      if (linkTab.value === 'magnet') magnetURI.value = val;
      else if (linkTab.value === 'video') videoURL.value = val;
      else hlsURL.value = val;
    },
  });

  const canSubmitLink = computed(() => activeLinkValue.value.trim().length > 0);

  /** 分片 tab 的输入框按钮只负责解析（播放按钮位于分片列表头部） */
  // 磁力链按钮只负责「解析」，之后在弹窗里选文件再决定播放还是下载；
  // 视频直链是直接播放；分片链按钮负责解析播放列表
  const linkActionLabel = computed(() => {
    if (linkTab.value === 'magnet') return '解析';
    if (linkTab.value === 'video') return '播放';
    return hlsParsed.value ? '重新解析' : '解析';
  });

  const linkActionIcon = computed(() => {
    if (linkTab.value === 'magnet') return 'troubleshoot';
    if (linkTab.value === 'video') return 'play_circle_filled';
    return hlsParsed.value ? 'refresh' : 'troubleshoot';
  });

  const linkActionTooltip = computed(() => {
    if (linkTab.value === 'magnet') return '解析磁力链并选择要播放或下载的文件';
    if (linkTab.value === 'video') return activeLinkTab.value.tooltip;
    if (!hlsParsed.value) return '解析分片列表';
    return `重新拉取并解析播放列表（当前保留 ${hlsKeptCount.value} 个分片）`;
  });

  const linkActionLoading = computed(
    () => hlsParsing.value || hlsLoading.value,
  );

  function notifyNegative(message: string) {
    $q.notify({ type: 'negative', message, position: 'top' });
  }

  function switchLinkTab(tab: LinkTab) {
    linkTab.value = tab;
  }

  /** 销毁 HLS 实例（切换资源 / 停止播放 / 卸载组件时调用） */
  function destroyHls() {
    if (hlsInstance) {
      try {
        hlsInstance.destroy();
      } catch {
        /* 忽略销毁异常 */
      }
      hlsInstance = null;
    }
  }

  /** 释放上一次播放生成的 blob 地址 */
  function revokeBlobUrl() {
    if (blobUrl) {
      URL.revokeObjectURL(blobUrl);
      blobUrl = null;
    }
  }

  /** 清掉下载项的列表高亮（切换回放源 / 移除条目 / 卸载时调用） */
  function releaseDownloadPlaybackUrl() {
    hlsPlayingDownloadId.value = '';
  }

  /**
   * 清空解析结果。注意：不动 hlsDownloadList——下载列表独立于当前解析结果，
   * 重新添加 / 解析视频链接后列表必须保留，仍可回放。
   */
  function resetHlsParse() {
    parsedPlaylist.value = null;
    hlsParsedUrl.value = '';
    hlsAllSegments.value = [];
    hlsSegments.value = [];
    // 地址变化后旧的下载文件名不再适用，回到默认名
    hlsDownloadName.value = '';
  }

  function removeHlsSegment(id: number) {
    hlsSegments.value = hlsSegments.value.filter((seg) => seg.id !== id);
  }

  /** 删除同类分片：与目标分片「最后一节不同、前面都相同」的分片一并删除 */
  function removeHlsSimilarSegments(id: number) {
    const target = hlsSegments.value.find((seg) => seg.id === id);
    if (!target) return;
    const key = segmentClassKey(target.url);
    const remain = hlsSegments.value.filter(
      (seg) => segmentClassKey(seg.url) !== key,
    );
    const removed = hlsSegments.value.length - remain.length;
    if (removed === 0) return;
    hlsSegments.value = remain;
    $q.notify({
      type: 'warning',
      message: `已删除 ${removed} 个同类分片`,
      position: 'top',
    });
  }

  function restoreHlsSegments() {
    hlsSegments.value = [...hlsAllSegments.value];
  }

  /** 拉取 m3u8 文本（需要源站允许跨域） */
  async function fetchPlaylistText(url: string): Promise<string> {
    const res = await fetch(url, { credentials: 'omit' });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return await res.text();
  }

  /** 解析分片链接，把分片列表列出来 */
  async function parseHls() {
    const url = hlsURL.value.trim();
    if (!isHttpUrl(url)) {
      notifyNegative('请输入有效的分片链接（http/https）');
      return;
    }
    hlsParsing.value = true;
    try {
      const text = await fetchPlaylistText(url);
      if (!text.includes('#EXTM3U')) {
        throw new Error('内容不是有效的 m3u8 播放列表');
      }
      const parsed = parsePlaylist(text, url);
      if (parsed.segments.length === 0) {
        throw new Error('播放列表中没有可用分片');
      }
      parsedPlaylist.value = parsed;
      hlsParsedUrl.value = url;
      hlsAllSegments.value = parsed.segments;
      hlsSegments.value = [...parsed.segments];
      $q.notify({
        type: 'positive',
        message: `解析完成，共 ${parsed.segments.length} 个分片`,
        position: 'top',
      });
    } catch (e) {
      resetHlsParse();
      notifyNegative('解析失败：' + (e as Error).message);
    } finally {
      hlsParsing.value = false;
    }
  }

  /** 用 video 标签或 hls.js 播放给定的播放列表地址 */
  async function startHlsPlayback(source: string, name: string) {
    const videoEl = getVideoEl();
    if (!videoEl) return;

    // Safari / iOS 原生支持 HLS，直接交给 video 标签
    if (videoEl.canPlayType(M3U8_MIME)) {
      onPlay(source, name, false);
      // 部分浏览器不支持用 blob 地址走原生 HLS 管线，失败时给出提示
      if (source.startsWith('blob:')) {
        videoEl.addEventListener(
          'error',
          () =>
            notifyNegative(
              '当前浏览器不支持播放过滤后的分片列表，可尝试 Chrome/Edge',
            ),
          { once: true },
        );
      }
      return;
    }

    hlsLoading.value = true;
    try {
      const { default: Hls } = await import('hls.js');
      if (!Hls.isSupported()) {
        hlsLoading.value = false;
        notifyNegative('当前浏览器不支持 HLS 播放');
        return;
      }
      // 先通知页面重置播放状态（内部会销毁上一个 HLS 实例）
      onPlay('', name, true);
      const instance = new Hls({ enableWorker: true });
      hlsInstance = instance;
      instance.on(Hls.Events.MANIFEST_PARSED, () => {
        hlsLoading.value = false;
        videoEl.muted = false;
        videoEl.volume = getVolume() > 0 ? getVolume() : 0.8;
        videoEl.play().catch((e: Error) => {
          console.warn('HLS autoplay blocked:', e.message);
        });
      });
      instance.on(Hls.Events.ERROR, (_event, data) => {
        if (!data.fatal) return;
        if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
          instance.startLoad();
        } else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
          instance.recoverMediaError();
        } else {
          notifyNegative('分片视频加载失败');
          destroyHls();
        }
      });
      instance.loadSource(source);
      instance.attachMedia(videoEl);
    } catch (e) {
      hlsLoading.value = false;
      notifyNegative('HLS 播放器加载失败: ' + (e as Error).message);
    }
  }

  /** 播放剩余分片：有删除时用重建后的播放列表，否则直接播原始地址 */
  async function playHlsRemaining() {
    const parsed = parsedPlaylist.value;
    if (!parsed || hlsSegments.value.length === 0) {
      notifyNegative('没有可播放的分片');
      return;
    }
    const removed = hlsRemovedCount.value;
    if (removed > 0 && !parsed.hasEndList) {
      $q.notify({
        type: 'warning',
        message: '该播放列表疑似直播流，删除分片可能导致时间轴异常',
        position: 'top',
      });
    }

    destroyHls();
    revokeBlobUrl();
    releaseDownloadPlaybackUrl();

    const name = fileNameFromUrl(hlsParsedUrl.value || hlsURL.value);
    let source = hlsParsedUrl.value || hlsURL.value.trim();
    if (removed > 0) {
      const text = buildPlaylistText(parsed, hlsSegments.value);
      blobUrl = URL.createObjectURL(new Blob([text], { type: M3U8_MIME }));
      source = blobUrl;
    }
    await startHlsPlayback(source, name);
  }

  /** 服务端任务 → 下载列表条目 */
  function toDownloadItem(task: HlsServerTask): HlsDownloadItem {
    const status = mapTaskStatus(task.Status);
    const total = task.TotalSegments || 0;
    const done = task.Segments || 0;
    return {
      id: task.ID,
      name: task.Name || '未命名',
      path: task.Path || '',
      segmentCount: done,
      totalCount: total,
      progress: status === 'done' ? 100 : task.Progress || 0,
      status,
      duration: task.Duration || '',
      sizeText: task.Size ? formatSize(task.Size) : '',
      target:
        status === 'done'
          ? task.Path || task.Log
          : status === 'failed'
            ? task.Log || '下载失败'
            : status === 'canceled'
              ? '已取消'
              : '正在服务端下载…',
      createdAt: new Date(task.CreateTime).getTime() || Date.now(),
      playable: status === 'done' && Boolean(task.Path),
      sourceUrl: task.URL || '',
    };
  }

  /** 同步服务端分片下载任务到下载列表 */
  async function refreshHlsDownloads() {
    try {
      const res = await TransferTasksInfo();
      const tasks: HlsServerTask[] = res?.Data?.tasks ?? [];
      hlsDownloadList.value = tasks
        .filter((task) => task.Type === HLS_TASK_TYPE)
        .map(toDownloadItem);
      syncDownloadPolling();
    } catch {
      // 拉取失败时保留上次结果，等待下次轮询
    }
  }

  /** 有进行中的任务时才轮询，避免空闲时无谓请求 */
  function syncDownloadPolling() {
    if (hlsDownloadActive.value) {
      if (!downloadPollTimer) {
        downloadPollTimer = setInterval(() => {
          void refreshHlsDownloads();
        }, DOWNLOAD_POLL_INTERVAL);
      }
      return;
    }
    if (downloadPollTimer) {
      clearInterval(downloadPollTimer);
      downloadPollTimer = null;
    }
  }

  /** 下载列表项的副标题：按任务状态展示进度或结果 */
  function hlsDownloadMeta(item: HlsDownloadItem): string {
    if (item.status === 'downloading') {
      return `${item.segmentCount}/${item.totalCount} 个分片 · ${item.progress}%`;
    }
    if (item.status === 'done') {
      const parts = [`${item.segmentCount} 个分片`, item.duration];
      if (item.sizeText) parts.push(item.sizeText);
      return parts.filter(Boolean).join(' · ');
    }
    const label = item.status === 'canceled' ? '已取消' : '下载失败';
    return `${label} · 已写入 ${item.segmentCount}/${item.totalCount} 个分片`;
  }

  /** 从下载列表移除一条记录：一并删除服务端任务；下载中会先取消 */
  async function removeHlsDownload(id: string) {
    if (hlsPlayingDownloadId.value === id) releaseDownloadPlaybackUrl();
    hlsDownloadList.value = hlsDownloadList.value.filter(
      (item) => item.id !== id,
    );
    try {
      await DelTransferTasksInfo(id);
    } catch {
      // 忽略删除失败，下面仍会刷新一次列表
    }
    void refreshHlsDownloads();
  }

  /** 列表项按钮：下载中 → 先请求取消再删除；已结束 → 直接移除记录 */
  async function cancelHlsDownload(id: string) {
    const item = hlsDownloadList.value.find((it) => it.id === id);
    if (item?.status === 'downloading') {
      try {
        await HlsCancelAPI(id);
      } catch {
        // 取消失败也继续尝试删除
      }
      // 给服务端协程一点收尾时间，避免边写边删
      setTimeout(() => void removeHlsDownload(id), 600);
      return;
    }
    await removeHlsDownload(id);
  }

  /** 清空下载列表：删除所有分片下载任务（不删除已下载到服务端的文件） */
  async function clearHlsDownloads() {
    const ids = hlsDownloadList.value.map((item) => item.id);
    hlsDownloadList.value = [];
    releaseDownloadPlaybackUrl();
    for (const id of ids) {
      try {
        await HlsCancelAPI(id);
      } catch {
        /* 已完成的任务取消失败可忽略 */
      }
      try {
        await DelTransferTasksInfo(id);
      } catch {
        /* 忽略删除失败 */
      }
    }
    void refreshHlsDownloads();
  }

  /**
   * 播放下载列表里的视频：文件已在服务端，直接走流式接口回放。
   * 注意：保存目录需位于系统设置里的媒体目录内，否则会被路径校验拦截。
   */
  async function playHlsDownload(item: HlsDownloadItem) {
    if (!item.path) {
      notifyNegative('找不到文件路径，无法在页面内回放');
      return;
    }
    releaseDownloadPlaybackUrl();
    revokeBlobUrl();
    hlsPlayingDownloadId.value = item.id;
    onPlay(
      `/api/stream/GetFileByPathUseEncode/${encodeURIComponent(item.path)}`,
      item.name,
      false,
    );
  }

  /**
   * 提交分片下载任务：由服务端拉取并合并分片。
   * 面板不再持有下载状态，因此关闭弹窗 / 刷新页面都不会中断下载。
   */
  async function downloadHls() {
    const parsed = parsedPlaylist.value;
    const segments = hlsSegments.value;
    if (!parsed || segments.length === 0) {
      notifyNegative('没有可下载的分片');
      return;
    }

    const keys = resolveSegmentKeys(parsed);
    if (segments.some((seg) => keys[seg.id - 1]?.method === 'SAMPLE-AES')) {
      notifyNegative('该视频使用 SAMPLE-AES 加密，暂不支持下载');
      return;
    }

    // 用户填了名字就用它（自动补扩展名），留空则回退到默认名
    const fileName = resolveDownloadName(
      hlsDownloadName.value,
      hlsDefaultDownloadName.value,
      hlsDownloadExt.value,
    );
    const sourceUrl = hlsParsedUrl.value || hlsURL.value.trim();
    // 重建播放列表（含分片删除结果），交给服务端按序拉取
    const playlist = buildPlaylistText(parsed, segments);
    const total = segments.length;

    hlsLoading.value = true;
    try {
      const res = await HlsDownloadAPI({
        playlist,
        sourceUrl,
        fileName,
        dir: hlsDownloadDir.value,
      });
      if (res?.Code !== 200) {
        notifyNegative(res?.Message || '创建下载任务失败');
        return;
      }
      $q.notify({
        type: 'positive',
        message: `已提交服务端下载 · ${total} 个分片`,
        caption: '关闭弹窗或刷新页面都不会中断下载',
        position: 'top',
      });
      // 提交后立刻同步一次，让新任务出现在列表里
      await refreshHlsDownloads();
      syncDownloadPolling();
    } catch (e) {
      notifyNegative('创建下载任务失败：' + (e as Error).message);
    } finally {
      hlsLoading.value = false;
    }
  }

  function submitVideo() {
    const url = videoURL.value.trim();
    if (!isHttpUrl(url)) {
      notifyNegative('请输入有效的视频链接（http/https）');
      return;
    }
    // 误把分片链粘到视频链接时自动切到分片 tab
    if (HLS_URL_RE.test(url)) {
      hlsURL.value = url;
      linkTab.value = 'hls';
      void parseHls();
      return;
    }
    releaseDownloadPlaybackUrl();
    onPlay(url, fileNameFromUrl(url), false);
  }

  async function submitLink() {
    if (linkTab.value === 'magnet') {
      await submitMagnet();
      return;
    }
    if (linkTab.value === 'video') {
      submitVideo();
      return;
    }
    // 分片 tab：输入框按钮只负责解析，播放由分片列表头部的播放按钮触发
    await parseHls();
  }

  // 输入地址变化后，之前的解析结果失效
  watch(hlsURL, () => {
    if (hlsParsed.value && hlsURL.value.trim() !== hlsParsedUrl.value) {
      resetHlsParse();
    }
  });

  // 首次进入时同步一次服务端下载任务：重开弹窗即可看到进行中的下载
  void refreshHlsDownloads();

  function cleanup() {
    // 只释放与播放相关的本地资源；服务端下载任务照常继续
    if (downloadPollTimer) {
      clearInterval(downloadPollTimer);
      downloadPollTimer = null;
    }
    destroyHls();
    revokeBlobUrl();
    releaseDownloadPlaybackUrl();
    resetHlsParse();
  }

  return {
    // state
    linkTab,
    linkTabs: LINK_TABS,
    linkFocused,
    activeLinkTab,
    activeLinkValue,
    canSubmitLink,
    hlsLoading,
    hlsParsing,
    linkActionLabel,
    linkActionIcon,
    linkActionTooltip,
    linkActionLoading,
    // 分片列表
    hlsParsed,
    hlsSegments,
    hlsTotalCount,
    hlsKeptCount,
    hlsRemovedCount,
    hlsKeptDuration,
    // 下载（服务端任务）
    hlsDownloadName,
    hlsDefaultDownloadName,
    hlsDownloadDir,
    hlsDownloadDirOptions,
    hlsDownloadActive,
    // 下载列表（服务端任务镜像，含进度 / 状态 / 回放）
    hlsDownloadList,
    hlsPlayingDownloadId,
    hlsDownloadMeta,
    // actions
    switchLinkTab,
    submitLink,
    parseHls,
    playHlsRemaining,
    removeHlsSegment,
    removeHlsSimilarSegments,
    restoreHlsSegments,
    downloadHls,
    cancelHlsDownload,
    chooseHlsDownloadDir,
    refreshHlsDownloads,
    playHlsDownload,
    clearHlsDownloads,
    destroyHls,
    cleanup,
  };
}
