import { computed, ref, watch, type Ref } from 'vue';
import type { QVueGlobals } from 'quasar';
import type HlsJs from 'hls.js';
import {
  ensureDirWritable,
  forgetDir,
  fsDirSupported,
  fsSaveSupported,
  loadSavedDir,
  openWritableInDir,
  pickDir,
  pickSaveFile,
} from 'src/utils/downloadDir';

// 外部链接播放逻辑（磁力链 / 视频链接 / 分片链接）
// 磁力链复用 useTorrentDownload，视频直链与 HLS 分片链在此处理。
// 分片链支持：解析 m3u8 → 列出分片 → 删除指定分片（如广告）→ 播放剩余分片。

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

const LINK_TAB_VALUES: string[] = LINK_TABS.map((t) => t.value);

/** 读取上次选择的 tab；脏数据或已下线的 tab 回退到默认「磁力链」 */
function readStoredLinkTab(): LinkTab {
  try {
    const saved = localStorage.getItem(LINK_TAB_STORAGE_KEY);
    if (saved && LINK_TAB_VALUES.includes(saved)) return saved as LinkTab;
  } catch {
    // 隐私模式 / 存储被禁用时忽略
  }
  return 'magnet';
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

/** 下载落地目标：优先文件系统访问 API，退化到内存 Blob */
interface DownloadSink {
  write: (chunk: Bytes) => Promise<void>;
  close: () => Promise<void>;
  abort: () => Promise<void>;
  /** 落盘位置的补充说明，用于下载完成提示 */
  target: string;
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
}

const HTTP_URL_RE = /^https?:\/\//i;
const HLS_URL_RE = /\.m3u8(\?|#|$)/i;
const M3U8_MIME = 'application/vnd.apple.mpegurl';
/** 下载并发数：分批并发拉取、按序写出，兼顾速度与内存占用 */
const DOWNLOAD_BATCH_SIZE = 4;

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

/** 分片序号转 128 位大端 IV（#EXT-X-KEY 未显式给 IV 时的默认值） */
function sequenceToIv(sequence: number): Bytes {
  const iv = new Uint8Array(16);
  let value = Math.max(0, Math.trunc(sequence));
  for (let i = 15; i >= 0 && value > 0; i--) {
    iv[i] = value % 256;
    value = Math.floor(value / 256);
  }
  return iv;
}

function parseMediaSequence(header: string[]): number {
  const line = header.find((item) => item.startsWith('#EXT-X-MEDIA-SEQUENCE'));
  if (!line) return 0;
  const value = Number(line.split(':')[1]);
  return Number.isFinite(value) ? value : 0;
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
        if (segments.length === 0) header.push(absolutizeTagUris(line, baseUrl));
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

function triggerBrowserDownload(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  link.remove();
  // 立即 revoke 会让部分浏览器拿不到文件，延迟释放
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

/**
 * 创建下载落地目标，按以下顺序择优：
 * 1. 记住的下载目录且仍有写权限 → 直接落盘，全程无对话框（「不用重复选目录」）；
 * 2. 「另存为」对话框，默认定位到记住的目录，点一下保存即可；
 * 3. 环境不支持（非安全上下文 / 非 Chromium）→ 内存 Blob + 浏览器下载。
 * 返回 null 表示用户取消了保存对话框。
 */
async function createDownloadSink(
  fileName: string,
): Promise<DownloadSink | null> {
  const savedDir = await loadSavedDir();

  // 1. 记忆中的目录：有权限就直接写文件
  if (savedDir && (await ensureDirWritable(savedDir))) {
    try {
      const { stream, fileName: actual } = await openWritableInDir(
        savedDir,
        fileName,
      );
      return {
        write: (chunk) => stream.write(chunk),
        close: () => stream.close(),
        abort: () => stream.abort(),
        target: `已存入 ${savedDir.name}/${actual}`,
      };
    } catch {
      // 目录被删除或改名，记忆已失效；清掉后继续走对话框
      await forgetDir();
    }
  }

  // 2. 「另存为」对话框：id 让浏览器记住上次目录，startIn 直接定位
  if (fsSaveSupported) {
    const picked = await pickSaveFile(fileName, savedDir ?? 'downloads');
    if (picked.status === 'cancelled') return null;
    if (picked.status === 'ok') {
      const stream = await picked.handle.createWritable();
      return {
        write: (chunk) => stream.write(chunk),
        close: () => stream.close(),
        abort: () => stream.abort(),
        target: `已保存为 ${picked.handle.name}`,
      };
    }
  }

  // 3. 非安全上下文 / 不支持该 API：内存 Blob + 浏览器下载
  const chunks: Bytes[] = [];
  return {
    async write(chunk) {
      chunks.push(chunk);
    },
    async close() {
      triggerBrowserDownload(
        new Blob(chunks, { type: 'application/octet-stream' }),
        fileName,
      );
      chunks.length = 0;
    },
    async abort() {
      chunks.length = 0;
    },
    target: `已交给浏览器下载 ${fileName}`,
  };
}

/** AES-128 分片解密（WebCrypto 只在 https / localhost 等安全上下文可用） */
async function decryptAes128(
  data: Bytes,
  keyBytes: Bytes,
  iv: Bytes,
): Promise<Bytes> {
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    keyBytes,
    { name: 'AES-CBC' },
    false,
    ['decrypt'],
  );
  const plain = await crypto.subtle.decrypt(
    { name: 'AES-CBC', iv },
    cryptoKey,
    data,
  );
  return new Uint8Array(plain);
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

  // ── 下载（另存为）状态 ───────────────────────────────────────────────────────
  const hlsDownloading = ref(false);
  const hlsDownloadProgress = ref(0);
  let downloadAborted = false;
  let activeSink: DownloadSink | null = null;

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

  // ── 记住的下载目录（设置后下载直接落盘，不再弹「另存为」） ─────────────────────
  /** 已记住的目录名，空串表示尚未设置 */
  const hlsDownloadDir = ref('');

  /** 读取已保存的目录名（初始化时调用一次） */
  async function refreshDownloadDir() {
    const dir = await loadSavedDir();
    hlsDownloadDir.value = dir?.name ?? '';
  }

  /** 选择 / 更换下载目录，之后下载不再需要逐次选目录 */
  async function pickHlsDownloadDir() {
    try {
      const dir = await pickDir();
      if (!dir) {
        notifyNegative('当前浏览器不支持选择目录，将沿用另存为对话框');
        return;
      }
      hlsDownloadDir.value = dir.name;
      $q.notify({
        type: 'positive',
        message: `已记住下载目录：${dir.name}`,
        caption: '之后的下载会直接保存到该目录，不再询问',
        position: 'top',
      });
    } catch (e) {
      // 用户取消目录选择，静默处理
      if ((e as DOMException)?.name !== 'AbortError') {
        notifyNegative('选择目录失败：' + (e as Error).message);
      }
    }
  }

  void refreshDownloadDir();

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

    const name = fileNameFromUrl(hlsParsedUrl.value || hlsURL.value);
    let source = hlsParsedUrl.value || hlsURL.value.trim();
    if (removed > 0) {
      const text = buildPlaylistText(parsed, hlsSegments.value);
      blobUrl = URL.createObjectURL(new Blob([text], { type: M3U8_MIME }));
      source = blobUrl;
    }
    await startHlsPlayback(source, name);
  }

  /** 拉取单个分片并按需解密 */
  async function fetchSegmentChunk(
    segment: HlsSegment,
    key: SegmentKey | null,
    sequence: number,
    keyCache: Map<string, Bytes>,
  ): Promise<Bytes> {
    const headers: Record<string, string> = {};
    if (segment.byteRange) {
      const { offset, length } = segment.byteRange;
      headers.Range = `bytes=${offset}-${offset + length - 1}`;
    }
    const res = await fetch(segment.url, { credentials: 'omit', headers });
    if (!res.ok) {
      throw new Error(`分片 #${segment.index} 下载失败 HTTP ${res.status}`);
    }
    const bytes = new Uint8Array(await res.arrayBuffer());
    if (!key || key.method !== 'AES-128' || !key.url) return bytes;

    let keyBytes = keyCache.get(key.url);
    if (!keyBytes) {
      const keyRes = await fetch(key.url, { credentials: 'omit' });
      if (!keyRes.ok) {
        throw new Error(`解密密钥获取失败 HTTP ${keyRes.status}`);
      }
      keyBytes = new Uint8Array(await keyRes.arrayBuffer());
      keyCache.set(key.url, keyBytes);
    }
    return await decryptAes128(
      bytes,
      keyBytes,
      key.iv ?? sequenceToIv(sequence),
    );
  }

  /** fMP4 的初始化段（#EXT-X-MAP）需要拼在最前面才能播放 */
  async function fetchInitSegment(
    parsed: ParsedPlaylist,
  ): Promise<Bytes | null> {
    const line = parsed.header.find((item) => item.startsWith('#EXT-X-MAP'));
    if (!line) return null;
    const attrs = tagAttributes(line);
    if (!attrs.URI) return null;
    const headers: Record<string, string> = {};
    const range = parseByteRangeValue(attrs.BYTERANGE, 0);
    if (range) {
      headers.Range = `bytes=${range.offset}-${range.offset + range.length - 1}`;
    }
    const res = await fetch(attrs.URI, { credentials: 'omit', headers });
    if (!res.ok) throw new Error(`初始化段下载失败 HTTP ${res.status}`);
    return new Uint8Array(await res.arrayBuffer());
  }

  function cancelHlsDownload() {
    if (!hlsDownloading.value) return;
    downloadAborted = true;
  }

  /** 把保留下来的分片按序合并，另存为本地文件 */
  async function downloadHls() {
    const parsed = parsedPlaylist.value;
    const segments = hlsSegments.value;
    if (!parsed || segments.length === 0) {
      notifyNegative('没有可下载的分片');
      return;
    }
    if (hlsDownloading.value) return;

    const keys = resolveSegmentKeys(parsed);
    if (segments.some((seg) => keys[seg.id - 1]?.method === 'SAMPLE-AES')) {
      notifyNegative('该视频使用 SAMPLE-AES 加密，暂不支持下载');
      return;
    }
    const encrypted = segments.some(
      (seg) => keys[seg.id - 1]?.method === 'AES-128',
    );
    if (encrypted && !globalThis.crypto?.subtle) {
      notifyNegative('当前环境无法解密分片（需 https 或 localhost 访问）');
      return;
    }

    // 用户填了名字就用它（自动补扩展名），留空则回退到默认名
    const fileName = resolveDownloadName(
      hlsDownloadName.value,
      hlsDefaultDownloadName.value,
      hlsDownloadExt.value,
    );
    const sequenceBase = parseMediaSequence(parsed.header);
    const keyCache = new Map<string, Bytes>();

    downloadAborted = false;
    hlsDownloadProgress.value = 0;
    hlsDownloading.value = true;
    try {
      // 保存对话框必须在用户手势内弹出，故先建 sink 再拉数据
      const sink = await createDownloadSink(fileName);
      if (!sink) return;
      activeSink = sink;

      const initChunk = await fetchInitSegment(parsed);
      if (initChunk) await sink.write(initChunk);

      let written = 0;
      for (let i = 0; i < segments.length; i += DOWNLOAD_BATCH_SIZE) {
        if (downloadAborted) throw new Error('已取消下载');
        const batch = segments.slice(i, i + DOWNLOAD_BATCH_SIZE);
        const chunks = await Promise.all(
          batch.map((seg) =>
            fetchSegmentChunk(
              seg,
              keys[seg.id - 1],
              sequenceBase + seg.id - 1,
              keyCache,
            ),
          ),
        );
        for (const chunk of chunks) {
          await sink.write(chunk);
          written++;
        }
        hlsDownloadProgress.value = Math.round(
          (written / segments.length) * 100,
        );
      }

      if (downloadAborted) throw new Error('已取消下载');
      await sink.close();
      $q.notify({
        type: 'positive',
        message: `已保存 ${written} 个分片 · ${hlsKeptDuration.value}`,
        caption: sink.target,
        position: 'top',
      });
    } catch (e) {
      if (activeSink) {
        try {
          await activeSink.abort();
        } catch {
          /* 忽略中止异常 */
        }
      }
      if (!downloadAborted) notifyNegative('下载失败：' + (e as Error).message);
    } finally {
      activeSink = null;
      hlsDownloading.value = false;
      hlsDownloadProgress.value = 0;
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

  function cleanup() {
    downloadAborted = true;
    if (activeSink) {
      void activeSink.abort().catch(() => {
        /* 忽略中止异常 */
      });
      activeSink = null;
    }
    destroyHls();
    revokeBlobUrl();
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
    // 下载（另存为）
    hlsDownloading,
    hlsDownloadProgress,
    hlsDownloadName,
    hlsDefaultDownloadName,
    hlsDownloadDir,
    fsDirSupported,
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
    pickHlsDownloadDir,
    destroyHls,
    cleanup,
  };
}
