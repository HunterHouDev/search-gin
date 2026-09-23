import { computed, ref, watch, type Ref } from 'vue';
import type { QVueGlobals } from 'quasar';
import type HlsJs from 'hls.js';
import {
  DelTransferTasksInfo,
  HlsCancelAPI,
  HlsDownloadAPI,
  HlsRestartAPI,
  HlsPlaylistAPI,
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

/** 下载完成后自动转码的合法取值，'' 表示不转码 */
const HLS_XCODE_VALUES = ['', 'copy', 'h264', 'h265'] as const;

/** 读取上次选择的下载后转码方式；脏数据按不转码处理 */
function readStoredDownloadXcode(): string {
  try {
    const saved = localStorage.getItem(DOWNLOAD_XCODE_STORAGE_KEY) ?? '';
    return (HLS_XCODE_VALUES as readonly string[]).includes(saved)
      ? saved
      : '';
  } catch {
    return '';
  }
}

/** 读取广告分片黑名单（分片「类」前缀数组）；脏数据忽略 */
function readHlsAdBlacklist(): string[] {
  try {
    const saved = localStorage.getItem(HLS_AD_BLACKLIST_KEY);
    if (!saved) return [];
    const list: unknown = JSON.parse(saved);
    if (!Array.isArray(list)) return [];
    return list.filter((item): item is string => typeof item === 'string' && !!item);
  } catch {
    // 存储被禁用或内容损坏时忽略
    return [];
  }
}

/** 解析出的单个 HLS 分片 */
export interface HlsSegment {
  /** 稳定 id（跨源全局唯一，用于删除 / 恢复） */
  id: number;
  /** 在所属源内的序号，从 1 开始 */
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
  /** 所属源 id（多源合并时用于分组展示与重建） */
  sourceId: string;
}

interface ParsedPlaylist {
  header: string[];
  segments: HlsSegment[];
  hasEndList: boolean;
  /** #EXT-X-MEDIA-SEQUENCE，缺省 IV（AES-128）的推导基准 */
  mediaSeq: number;
  /** #EXT-X-VERSION，合并多个源时取最大值 */
  version: number;
}

/**
 * 一个 m3u8 源。支持添加多个源并按顺序合并：
 * 播放时拼成一条带 #EXT-X-DISCONTINUITY 的播放列表，下载时由服务端按序拼接成一个文件。
 */
export interface HlsSource {
  /** 稳定 id（同一地址重复解析时复用） */
  id: string;
  /** 源地址 */
  url: string;
  /** 该源的解析结果（文件头 + 全量分片） */
  parsed: ParsedPlaylist;
  /** 全量分片，与 parsed.segments 指向同一批对象 */
  allSegments: HlsSegment[];
  /** 该源分片的总时长（秒） */
  totalSeconds: number;
}

/** 分片 id 自增序号：跨源全局唯一，源重排 / 删除后仍然稳定 */
let hlsSegmentIdSeq = 0;

/** 源 id 自增序号 */
let hlsSourceIdSeq = 0;

/** 分片链接输入行：每行一个地址，带稳定 id（避免拆行 / 删行时输入框被重建导致光标丢失） */
export interface HlsUrlRow {
  id: number;
  value: string;
}

let hlsUrlRowIdSeq = 0;

function createHlsUrlRow(value = ''): HlsUrlRow {
  return { id: ++hlsUrlRowIdSeq, value };
}

/**
 * 归一化输入行：丢弃空行（序号不留空档），并保证末尾始终有一个空行——
 * 用户在最后一行输入后会自动出现下一行。
 * keepEmptyId 指定的行即使为空也保留（正在清空重填的那一行，避免焦点丢失）。
 */
function normalizeHlsUrlRows(
  rows: HlsUrlRow[],
  keepEmptyId?: number,
): HlsUrlRow[] {
  const kept = rows.filter(
    (row) => row.value.trim().length > 0 || row.id === keepEmptyId,
  );
  const last = kept[kept.length - 1];
  if (!last || last.value.trim().length > 0) kept.push(createHlsUrlRow());
  return kept;
}

/** 文本 → 输入行（空格 / 换行分隔，每行一个地址） */
function toHlsUrlRows(text: string): HlsUrlRow[] {
  return normalizeHlsUrlRows(
    text
      .split(/\s+/)
      .filter((item) => item.length > 0)
      .map((item) => createHlsUrlRow(item)),
  );
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
/** 记住用户选择的下载后转码方式（'' 表示不转码） */
const DOWNLOAD_XCODE_STORAGE_KEY = 'immersive.serverDownloadXcode';
/** 广告分片黑名单（分片「类」前缀列表），刷新后继续生效 */
const HLS_AD_BLACKLIST_KEY = 'immersive.hlsAdBlacklist';
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

function bytesToHex(bytes: Bytes): string {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}

/** 按分片序号推导 IV（#EXT-X-KEY 未显式给出 IV 时的规范做法） */
function sequenceIV(sequence: number): Bytes {
  const iv = new Uint8Array(16);
  let value = Math.max(0, Math.trunc(sequence));
  for (let i = 15; i >= 0 && value > 0; i--) {
    iv[i] = value % 256;
    value = Math.floor(value / 256);
  }
  return iv;
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
function parsePlaylist(
  text: string,
  baseUrl: string,
  sourceId: string,
): ParsedPlaylist {
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
  let mediaSeq = 0;
  let version = 0;
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
      // 合并多个源时要用到的两个头：分片序号（缺省 IV 推导）与协议版本
      if (line.startsWith('#EXT-X-MEDIA-SEQUENCE')) {
        const seq = Number(line.slice(line.indexOf(':') + 1).trim());
        if (Number.isFinite(seq)) mediaSeq = seq;
      }
      if (line.startsWith('#EXT-X-VERSION')) {
        const ver = Number(line.slice(line.indexOf(':') + 1).trim());
        if (Number.isFinite(ver)) version = ver;
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
      id: ++hlsSegmentIdSeq,
      index: segments.length + 1,
      duration: parseExtinfDuration(pendingExtinf),
      url,
      tags: pendingTags,
      extinf: pendingExtinf,
      byteRange,
      sourceId,
    });
    pendingTags = [];
    pendingExtinf = null;
    pendingByteRange = null;
  }

  return { header, segments, hasEndList, mediaSeq, version };
}

/** 单源：保留该源的原始文件头，只按保留的分片重建（与既有行为一致） */
function buildSingleSourceText(
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

/**
 * 生成分片的密钥行。缺省 IV 时必须显式写出：多个源合并后分片序号整体位移，
 * 播放器 / 服务端按序号推导出的 IV 会与源站加解密时的序号对不上。
 */
function renderSegmentKey(key: SegmentKey | null, sequence: number): string {
  if (!key) return '#EXT-X-KEY:METHOD=NONE';
  const iv = key.iv ?? sequenceIV(sequence);
  return `#EXT-X-KEY:METHOD=${key.method},URI="${key.url}",IV=0x${bytesToHex(iv)}`;
}

/**
 * 重建播放列表文本。
 * - 单源：原样保留文件头（与既往行为一致）；
 * - 多源：按顺序拼接，源之间插入 #EXT-X-DISCONTINUITY，并在每个源的首个分片前
 *   声明该源自己的初始化段（#EXT-X-MAP）与密钥（带显式 IV），
 *   播放器据此在段边界重建解码器，服务端据此按序切换初始化段。
 */
function buildPlaylistText(
  sources: HlsSource[],
  segments: HlsSegment[],
): string {
  const first = sources[0];
  if (!first) return '';
  if (sources.length === 1)
    return buildSingleSourceText(first.parsed, segments);

  const out: string[] = ['#EXTM3U'];
  out.push(
    `#EXT-X-VERSION:${Math.max(
      3,
      ...sources.map((source) => source.parsed.version),
    )}`,
  );
  // 目标时长为合并后的最大分片时长，避免播放器按旧值预加载
  const longest = segments.reduce((max, seg) => Math.max(max, seg.duration), 0);
  out.push(`#EXT-X-TARGETDURATION:${Math.max(1, Math.ceil(longest))}`);
  out.push('#EXT-X-MEDIA-SEQUENCE:0');
  if (
    sources.some((source) =>
      source.parsed.header.some((line) =>
        line.startsWith('#EXT-X-INDEPENDENT-SEGMENTS'),
      ),
    )
  ) {
    out.push('#EXT-X-INDEPENDENT-SEGMENTS');
  }

  let emitted = 0;
  // 初始值取 NONE：源开头本身没有密钥时，不必输出多余的 METHOD=NONE；
  // 该状态必须跨源延续——上一个源加密、下一个源不加密时必须补 METHOD=NONE
  let currentKeyLine = '#EXT-X-KEY:METHOD=NONE';
  for (const source of sources) {
    const kept = segments.filter((seg) => seg.sourceId === source.id);
    if (kept.length === 0) continue;
    // 段与段之间必须标记不连续，播放器才会重建解码器
    if (emitted > 0) out.push('#EXT-X-DISCONTINUITY');
    // 初始化段（fMP4）：排在该源第一个分片之前，且在 DISCONTINUITY 之后
    const mapLine = source.parsed.header.find((line) =>
      line.startsWith('#EXT-X-MAP'),
    );
    if (mapLine) out.push(mapLine);

    const keys = resolveSegmentKeys(source.parsed);
    for (const seg of kept) {
      const keyLine = renderSegmentKey(
        keys[seg.index - 1] ?? null,
        source.parsed.mediaSeq + seg.index - 1,
      );
      if (keyLine !== currentKeyLine) {
        out.push(keyLine);
        currentKeyLine = keyLine;
      }
      // 密钥行已由上面按显式 IV 统一输出，过滤掉原始标签里的 KEY 行
      out.push(...seg.tags.filter((tag) => !tag.startsWith('#EXT-X-KEY')));
      if (seg.extinf) out.push(seg.extinf);
      out.push(seg.url);
      emitted++;
    }
  }
  out.push('#EXT-X-ENDLIST');
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
  /** 分片链接输入行：每行一个 m3u8 地址，末尾自动保留一个空行 */
  const hlsUrlRows = ref<HlsUrlRow[]>([createHlsUrlRow()]);
  /** 分片链接文本（非空行拼接；输入框已改为多行，这里是它的读写视图） */
  const hlsURL = computed<string>({
    get: () =>
      hlsUrlRows.value
        .map((row) => row.value.trim())
        .filter((item) => item.length > 0)
        .join('\n'),
    set: (val: string) => {
      hlsUrlRows.value = toHlsUrlRows(val);
    },
  });
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

  // ── 分片解析状态（支持多个 m3u8 源按顺序合并） ────────────────────────────────
  /** 已添加的源，数组顺序即合并顺序 */
  const hlsSources = ref<HlsSource[]>([]);
  /** 全部源的全量分片（按源顺序扁平化，用于「恢复」） */
  const hlsAllSegments = ref<HlsSegment[]>([]);
  /** 当前保留的分片（删除后即时减少） */
  const hlsSegments = ref<HlsSegment[]>([]);

  const hlsParsed = computed(() => hlsSources.value.length > 0);
  const hlsSourceCount = computed(() => hlsSources.value.length);
  /** 首个源地址：默认文件名、Referer 等以它为准 */
  const hlsPrimaryUrl = computed(() => hlsSources.value[0]?.url ?? '');

  /** 源列表（含保留分片数与时长），供 UI 展示 */
  const hlsSourceList = computed(() =>
    hlsSources.value.map((source) => {
      const kept = hlsUsableSegments.value.filter(
        (seg) => seg.sourceId === source.id,
      );
      return {
        source,
        keptCount: kept.length,
        totalCount: source.allSegments.length,
        duration: formatSeconds(
          kept.reduce((sum, seg) => sum + seg.duration, 0),
        ),
      };
    }),
  );

  /** 分片按源分组，供分片列表渲染 */
  const hlsSegmentGroups = computed(() =>
    hlsSources.value.map((source) => ({
      sourceId: source.id,
      url: source.url,
      segments: hlsSegments.value.filter((seg) => seg.sourceId === source.id),
    })),
  );

  // ── 广告分片黑名单 ──────────────────────────────────────────────────────────
  // 记录被标记为广告的「分片类」（即 segmentClassKey 前缀）。命中黑名单的分片在
  // 解析时即被过滤，后续解析同一站点的其它视频也会自动剔除，无需重复标记。
  const hlsAdBlacklist = ref<string[]>(readHlsAdBlacklist());

  // 黑名单写回本地存储，刷新/重开弹窗后继续生效
  watch(
    hlsAdBlacklist,
    (list) => {
      try {
        if (list.length > 0) {
          localStorage.setItem(HLS_AD_BLACKLIST_KEY, JSON.stringify(list));
        } else {
          localStorage.removeItem(HLS_AD_BLACKLIST_KEY);
        }
      } catch {
        // 隐私模式写入失败时忽略
      }
    },
    { deep: true },
  );

  /** 该分片是否命中广告黑名单 */
  function isHlsAdSegment(seg: HlsSegment): boolean {
    return hlsAdBlacklist.value.includes(segmentClassKey(seg.url));
  }

  /**
   * 实际参与播放 / 下载的分片。
   * 黑名单（广告）分片仍留在列表里展示（置灰、未勾选），但不计入、不使用。
   */
  const hlsUsableSegments = computed(() =>
    hlsSegments.value.filter((seg) => !isHlsAdSegment(seg)),
  );
  /** 当前列表里被标记为广告的分片数（仍可见，但已被排除） */
  const hlsAdCount = computed(
    () => hlsSegments.value.length - hlsUsableSegments.value.length,
  );

  /**
   * 标记广告：与「删除同类」同样的判定（同类 = 最后一节不同、前面都相同），
   * 但只在列表里置灰、不参与下载，并把该类加入黑名单——
   * 之后解析同站点的其它播放列表时，这类分片同样是置灰状态。
   */
  function markHlsAdSegments(id: number) {
    const target = hlsSegments.value.find((seg) => seg.id === id);
    if (!target) return;
    const key = segmentClassKey(target.url);
    if (hlsAdBlacklist.value.includes(key)) {
      $q.notify({
        type: 'warning',
        message: '该类已在广告黑名单中',
        position: 'top',
        timeout: 1500,
      });
      return;
    }
    hlsAdBlacklist.value = [...hlsAdBlacklist.value, key];
    const marked = hlsSegments.value.filter(
      (seg) => segmentClassKey(seg.url) === key,
    ).length;
    $q.notify({
      type: 'warning',
      message: `已标记 ${marked} 个同类分片为广告（列表置灰，下载时忽略）`,
      position: 'top',
    });
  }

  /** 取消某个黑名单条目：该类分片恢复参与播放 / 下载 */
  function unmarkHlsAdSegments(key: string) {
    hlsAdBlacklist.value = hlsAdBlacklist.value.filter((item) => item !== key);
  }

  /** 切换单个分片的广告标记：已标记则取消该类，未标记则标记该类 */
  function toggleHlsAdSegment(id: number) {
    const target = hlsSegments.value.find((seg) => seg.id === id);
    if (!target) return;
    const key = segmentClassKey(target.url);
    if (hlsAdBlacklist.value.includes(key)) unmarkHlsAdSegments(key);
    else markHlsAdSegments(id);
  }

  /** 清空广告黑名单 */
  function clearHlsAdBlacklist() {
    if (hlsAdBlacklist.value.length === 0) return;
    hlsAdBlacklist.value = [];
    $q.notify({
      type: 'positive',
      message: '已清空广告黑名单',
      position: 'top',
      timeout: 2000,
    });
  }

  const hlsTotalCount = computed(() => hlsAllSegments.value.length);
  // 保留数 / 时长只统计实际参与的分片（广告不计入）
  const hlsKeptCount = computed(() => hlsUsableSegments.value.length);
  const hlsRemovedCount = computed(
    () => hlsTotalCount.value - hlsSegments.value.length,
  );
  const hlsKeptDuration = computed(() =>
    formatSeconds(
      hlsUsableSegments.value.reduce((sum, seg) => sum + seg.duration, 0),
    ),
  );

  // ── 下载文件名（留空则用默认名） ─────────────────────────────────────────────
  /** 用户自定义的保存文件名，空串表示使用默认名 */
  const hlsDownloadName = ref('');

  /** 下载容器后缀：任一源带 #EXT-X-MAP 初始化段，合并结果即为 fMP4 */
  const hlsDownloadExt = computed(() =>
    hlsSources.value.some((source) =>
      source.parsed.header.some((item) => item.startsWith('#EXT-X-MAP')),
    )
      ? 'mp4'
      : 'ts',
  );

  /** 默认下载文件名（输入框留空时使用），以首个源地址为准 */
  const hlsDefaultDownloadName = computed(() =>
    buildDownloadName(
      fileNameFromUrl(hlsPrimaryUrl.value || hlsURL.value),
      hlsDownloadExt.value,
    ),
  );

  // ── 下载完成后自动转码 ─────────────────────────────────────────────────────
  /**
   * 下载完成后自动转码的方式：'' 不转码 / copy 仅换封装为 mp4 / h264 / h265。
   * 由服务端在下载落盘后按产物路径直接创建转码任务（不依赖索引）。
   * 初始值取自上次选择，刷新后继续沿用。
   */
  const hlsDownloadXcode = ref(readStoredDownloadXcode());

  // 记住用户的转码选择，下次打开面板默认沿用
  watch(hlsDownloadXcode, (val) => {
    try {
      if (val) localStorage.setItem(DOWNLOAD_XCODE_STORAGE_KEY, val);
      else localStorage.removeItem(DOWNLOAD_XCODE_STORAGE_KEY);
    } catch {
      // 隐私模式写入失败时忽略
    }
  });

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

  /** 分片 tab 的输入框按钮只负责解析 / 追加（播放按钮位于分片列表头部） */
  // 磁力链按钮只负责「解析」，之后在弹窗里选文件再决定播放还是下载；
  // 视频直链是直接播放；分片链按钮负责解析播放列表，已解析时追加为新的源
  const linkActionLabel = computed(() => {
    if (linkTab.value === 'magnet') return '解析';
    if (linkTab.value === 'video') return '播放';
    return hlsParsed.value ? '添加' : '解析';
  });

  const linkActionIcon = computed(() => {
    if (linkTab.value === 'magnet') return 'troubleshoot';
    if (linkTab.value === 'video') return 'play_circle_filled';
    return hlsParsed.value ? 'add' : 'troubleshoot';
  });

  const linkActionTooltip = computed(() => {
    if (linkTab.value === 'magnet') return '解析磁力链并选择要播放或下载的文件';
    if (linkTab.value === 'video') return activeLinkTab.value.tooltip;
    if (!hlsParsed.value)
      return '解析分片列表，可一次粘贴多个 m3u8 地址（空格 / 换行分隔）';
    return `继续添加 m3u8 作为后续片段（当前 ${hlsSourceCount.value} 个源、保留 ${hlsKeptCount.value} 个分片）；同一地址会重新解析`;
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
    hlsSources.value = [];
    hlsAllSegments.value = [];
    hlsSegments.value = [];
    // 地址变化后旧的下载文件名不再适用，回到默认名
    hlsDownloadName.value = '';
  }

  /** 添加 / 刷新一个源：同地址视为重新解析，原位替换且顺序不变 */
  function upsertHlsSource(
    url: string,
    sourceId: string,
    parsed: ParsedPlaylist,
  ) {
    const index = hlsSources.value.findIndex((item) => item.url === url);
    const source: HlsSource = {
      id: sourceId,
      url,
      parsed,
      allSegments: parsed.segments,
      totalSeconds: parsed.segments.reduce((sum, seg) => sum + seg.duration, 0),
    };
    const next = [...hlsSources.value];
    if (index >= 0) next[index] = source;
    else next.push(source);
    hlsSources.value = next;

    const keptIds = new Set(hlsSegments.value.map((seg) => seg.id));
    hlsAllSegments.value = next.flatMap((item) => item.allSegments);
    // 新解析 / 刷新的源默认全部保留，其它源沿用已有删除结果。
    // 广告分片不在此过滤：它们仍要显示在列表里（置灰），只是下载 / 播放时忽略。
    hlsSegments.value = hlsAllSegments.value.filter(
      (seg) => seg.sourceId === source.id || keptIds.has(seg.id),
    );
  }

  /** 移除一个源及其分片 */
  function removeHlsSource(sourceId: string) {
    hlsSources.value = hlsSources.value.filter((item) => item.id !== sourceId);
    hlsAllSegments.value = hlsAllSegments.value.filter(
      (seg) => seg.sourceId !== sourceId,
    );
    hlsSegments.value = hlsSegments.value.filter(
      (seg) => seg.sourceId !== sourceId,
    );
  }

  /** 调整源的合并顺序（delta：-1 上移 / +1 下移） */
  function moveHlsSource(sourceId: string, delta: number) {
    const list = [...hlsSources.value];
    const index = list.findIndex((item) => item.id === sourceId);
    const target = index + delta;
    if (index < 0 || target < 0 || target >= list.length) return;
    [list[index], list[target]] = [list[target], list[index]];
    hlsSources.value = list;
    // 分片顺序跟随源顺序，已有的删除结果按 id 保留
    const keptIds = new Set(hlsSegments.value.map((seg) => seg.id));
    hlsAllSegments.value = list.flatMap((item) => item.allSegments);
    hlsSegments.value = hlsAllSegments.value.filter((seg) =>
      keptIds.has(seg.id),
    );
  }

  /**
   * 更新某一行分片链接。一行里粘贴多个地址（空格 / 换行分隔）会自动拆成多行，
   * 且在最后一行输入后自动补出下一行空行——无需手动回车换行。
   */
  function updateHlsUrlRow(rowId: number, value: string) {
    const rows = [...hlsUrlRows.value];
    const index = rows.findIndex((row) => row.id === rowId);
    if (index < 0) return;

    const urls = value.split(/\s+/).filter((item) => item.length > 0);
    // 第一个地址沿用原行 id：正在输入的那一行不会被重建，光标不丢；
    // 清空时同样保留该行（值为空），由用户决定重新填写还是继续往下写
    const replacement: HlsUrlRow[] =
      urls.length > 0
        ? urls.map((url, i) =>
            i === 0 ? { ...rows[index], value: url } : createHlsUrlRow(url),
          )
        : [{ ...rows[index], value: '' }];
    rows.splice(index, 1, ...replacement);
    hlsUrlRows.value = normalizeHlsUrlRows(rows, rowId);
  }

  /** 清空全部输入行 */
  function clearHlsUrlRows() {
    hlsUrlRows.value = [createHlsUrlRow()];
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

  /**
   * 解析分片链接。输入框支持一次粘贴多个地址（空格 / 换行分隔）：
   * 已存在的地址原位刷新，新地址追加为后续片段——多个源按顺序合并播放 / 下载。
   */
  async function parseHls() {
    const raw = hlsURL.value.trim();
    if (!raw) {
      notifyNegative('请输入分片链接');
      return;
    }
    const urls = raw.split(/\s+/).filter((item) => item.length > 0);
    const invalid = urls.find((item) => !isHttpUrl(item));
    if (invalid) {
      notifyNegative('请输入有效的分片链接（http/https）');
      return;
    }

    hlsParsing.value = true;
    const failures: string[] = [];
    let added = 0;
    let addedSegments = 0;
    try {
      for (const url of urls) {
        try {
          const text = await fetchPlaylistText(url);
          if (!text.includes('#EXTM3U')) {
            throw new Error('内容不是有效的 m3u8 播放列表');
          }
          // 已有地址沿用原源 id，保证刷新后删除结果之外的顺序与分组不变
          const existed = hlsSources.value.find((item) => item.url === url);
          const sourceId = existed?.id ?? `src-${++hlsSourceIdSeq}`;
          const parsed = parsePlaylist(text, url, sourceId);
          if (parsed.segments.length === 0) {
            throw new Error('播放列表中没有可用分片');
          }
          upsertHlsSource(url, sourceId, parsed);
          added++;
          addedSegments += parsed.segments.length;
        } catch (e) {
          failures.push(`${url}：${(e as Error).message}`);
        }
      }
    } finally {
      hlsParsing.value = false;
    }

    if (added > 0) {
      // 解析成功即清空输入框，便于继续粘贴下一个地址
      hlsURL.value = '';
      $q.notify({
        type: 'positive',
        message:
          hlsSourceCount.value > 1
            ? `已添加 ${added} 个播放列表 / ${addedSegments} 个分片，当前共 ${hlsSourceCount.value} 个源将顺序合并`
            : `解析完成，共 ${addedSegments} 个分片`,
        position: 'top',
      });
    }
    if (failures.length > 0) {
      notifyNegative(`解析失败：${failures.join('；')}`);
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

  /** 播放剩余分片：有删除或多个源时用重建后的播放列表，否则直接播原始地址 */
  async function playHlsRemaining() {
    const sources = hlsSources.value;
    const segments = hlsUsableSegments.value;
    if (sources.length === 0 || segments.length === 0) {
      notifyNegative('没有可播放的分片');
      return;
    }
    const removed = hlsRemovedCount.value;
    const multi = sources.length > 1;
    if (removed > 0 && !sources.every((item) => item.parsed.hasEndList)) {
      $q.notify({
        type: 'warning',
        message: '该播放列表疑似直播流，删除分片可能导致时间轴异常',
        position: 'top',
      });
    }
    if (multi) {
      $q.notify({
        type: 'info',
        message: `按顺序合并播放 ${sources.length} 个播放列表`,
        position: 'top',
      });
    }

    destroyHls();
    revokeBlobUrl();
    releaseDownloadPlaybackUrl();

    const name = fileNameFromUrl(hlsPrimaryUrl.value || hlsURL.value);
    let source = hlsPrimaryUrl.value || hlsURL.value.trim();
    // 多源合并或删过分片时，分片序号 / 密钥行与原始播放列表已对不上，必须重建
    if (removed > 0 || multi) {
      const text = buildPlaylistText(sources, segments);
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
    // 浏览器直下进行中时优先展示直下进度
    const bp = browserDownloadProgress.value[item.id];
    if (bp) return `浏览器直下 ${bp.done}/${bp.total} 个分片`;
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

  /** 重启失败/已取消的下载任务：服务端复用落盘的播放列表重新入队 */
  async function restartHlsDownload(id: string) {
    try {
      const res = await HlsRestartAPI(id);
      if (res?.Code !== 200) {
        notifyNegative(res?.Message || '重启失败');
        return;
      }
      $q.notify({
        type: 'positive',
        message: '任务已重新启动',
        position: 'top',
        timeout: 2000,
      });
      await refreshHlsDownloads();
      syncDownloadPolling();
    } catch (e) {
      notifyNegative('重启任务失败：' + (e as Error).message);
    }
  }

  // ── 浏览器直下（前端备用下载，不经过服务端） ─────────────────────────────
  /** 浏览器直下的并发窗口与单分片重试次数 */
  const BROWSER_DOWNLOAD_CONCURRENCY = 6;
  const BROWSER_SEGMENT_RETRY = 2;

  /** 浏览器直下进度：任务 id → 已完成分片数 / 总分片数 */
  const browserDownloadProgress = ref<Record<string, { done: number; total: number }>>({});

  /** 该任务是否正在浏览器直下 */
  function browserDownloadInProgress(item: HlsDownloadItem): boolean {
    return browserDownloadProgress.value[item.id] != null;
  }

  /** 浏览器直下的进度条取值（0~1），null 表示当前没有浏览器直下 */
  function browserProgressRatio(item: HlsDownloadItem): number | null {
    const bp = browserDownloadProgress.value[item.id];
    if (!bp) return null;
    return bp.total > 0 ? bp.done / bp.total : 0;
  }

  /** 单分片拉取：带重试；#EXT-X-BYTERANGE 分片用 Range 头取子区间 */
  async function fetchBrowserSegment(
    url: string,
    range: { length: number; offset: number } | null,
  ): Promise<Uint8Array> {
    const headers: Record<string, string> = {};
    if (range) headers.Range = `bytes=${range.offset}-${range.offset + range.length - 1}`;
    let lastErr: Error | null = null;
    for (let attempt = 0; attempt <= BROWSER_SEGMENT_RETRY; attempt++) {
      try {
        const res = await fetch(url, { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return new Uint8Array(await res.arrayBuffer());
      } catch (e) {
        lastErr = e as Error;
      }
    }
    throw lastErr ?? new Error('分片下载失败');
  }

  /** 导入 AES-128 密钥（HLS 标准即 AES-128-CBC + PKCS7，WebCrypto 原生支持） */
  async function importBrowserKey(bytes: Uint8Array): Promise<CryptoKey> {
    return crypto.subtle.importKey('raw', bytes, { name: 'AES-CBC' }, false, ['decrypt']);
  }

  /** 取任务播放列表：优先服务端落盘副本，其次直接从源站拉（受跨域限制） */
  async function resolveBrowserPlaylist(item: HlsDownloadItem): Promise<string> {
    try {
      const res = await HlsPlaylistAPI(item.id);
      if (res?.Code === 200 && res.Data) return res.Data;
    } catch {
      /* 服务端副本不可用时尝试源站直取 */
    }
    if (!item.sourceUrl) throw new Error('播放列表不存在，且源地址缺失');
    const res = await fetch(item.sourceUrl);
    if (!res.ok) throw new Error(`从源站拉取播放列表失败：HTTP ${res.status}`);
    return res.text();
  }

  /**
   * 浏览器直下核心：给定播放列表文本，浏览器直接从源站拉取全部分片，
   * AES-128 用 WebCrypto 解密，按序合并后另存到本机，全程不经过服务端。
   * 注意：源站禁止跨域（CORS）时会失败，此时请改用服务端下载。
   */
  async function browserDownloadFromPlaylist(
    id: string,
    name: string,
    sourceUrl: string | undefined,
    text: string,
  ) {
    try {
      const parsed = parsePlaylist(text, sourceUrl || location.href, `browser-${id}`);
      if (parsed.segments.length === 0) throw new Error('播放列表中没有分片');
      const keys = resolveSegmentKeys(parsed);
      if (keys.some((k) => k?.method === 'SAMPLE-AES')) {
        throw new Error('该视频使用 SAMPLE-AES 加密，浏览器直下暂不支持');
      }

      const total = parsed.segments.length;
      browserDownloadProgress.value = {
        ...browserDownloadProgress.value,
        [id]: { done: 0, total },
      };

      // fMP4：初始化段（#EXT-X-MAP）必须排在所有分片之前
      const chunks: Uint8Array[] = [];
      const mapLine = parsed.header.find((line) => line.startsWith('#EXT-X-MAP'));
      const mapUri = mapLine ? /URI="([^"]*)"/.exec(mapLine)?.[1] : undefined;
      if (mapUri) chunks.push(await fetchBrowserSegment(mapUri, null));

      // 乱序完成后按序落位，保证拼接顺序与播放列表一致
      const parts: (Uint8Array | null)[] = new Array(total).fill(null);
      let cursor = 0;
      let next = 0;
      let done = 0;
      const keyCache = new Map<string, CryptoKey>();

      const worker = async () => {
        for (;;) {
          const i = cursor++;
          if (i >= total) return;
          const seg = parsed.segments[i];
          const key = keys[i];
          let data = await fetchBrowserSegment(seg.url, seg.byteRange);
          if (key && key.method === 'AES-128') {
            let ck = keyCache.get(key.url);
            if (!ck) {
              ck = await importBrowserKey(await fetchBrowserSegment(key.url, null));
              keyCache.set(key.url, ck);
            }
            // 缺省 IV 按分片序号推导（与 HLS 规范一致）
            const iv = key.iv ?? sequenceIV(parsed.mediaSeq + seg.index - 1);
            data = new Uint8Array(
              await crypto.subtle.decrypt({ name: 'AES-CBC', iv }, ck, data),
            );
          }
          parts[i] = data;
          done++;
          browserDownloadProgress.value = {
            ...browserDownloadProgress.value,
            [id]: { done, total },
          };
          while (next < total && parts[next]) {
            chunks.push(parts[next] as Uint8Array);
            parts[next] = null;
            next++;
          }
        }
      };

      await Promise.all(
        Array.from({ length: Math.min(BROWSER_DOWNLOAD_CONCURRENCY, total) }, () => worker()),
      );

      const blob = new Blob(chunks as BlobPart[]);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = name || 'video';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      // 大文件触发保存后保留 object URL 一段时间，避免浏览器尚未开始写盘就被回收
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
      $q.notify({
        type: 'positive',
        message: `浏览器直下完成：${name}`,
        position: 'top',
        timeout: 2500,
      });
    } catch (e) {
      notifyBrowserDownloadError(e);
    } finally {
      const rest = { ...browserDownloadProgress.value };
      delete rest[id];
      browserDownloadProgress.value = rest;
    }
  }

  /**
   * 前端备用下载（下载列表条目）：优先服务端落盘的播放列表副本，
   * 其次从源站直接拉取，全程不经过服务端。
   */
  async function downloadHlsInBrowser(item: HlsDownloadItem) {
    if (browserDownloadInProgress(item)) {
      notifyNegative('该任务的浏览器直下正在进行中');
      return;
    }
    try {
      const text = await resolveBrowserPlaylist(item);
      await browserDownloadFromPlaylist(item.id, item.name, item.sourceUrl, text);
    } catch (e) {
      notifyBrowserDownloadError(e);
    }
  }

  /** 浏览器直下失败的统一提示：跨域类错误给出改用服务端下载的建议 */
  function notifyBrowserDownloadError(e: unknown) {
    const msg = (e as Error).message || String(e);
    notifyNegative(
      /Failed to fetch|NetworkError|CORS|load failed/i.test(msg)
        ? `浏览器直下失败（源站可能禁止跨域）：${msg}，建议使用服务端下载`
        : `浏览器直下失败：${msg}`,
    );
  }

  /** 本地面板浏览器直下固定使用的进度 id */
  const BROWSER_NOW_ID = 'local';

  /** 本地面板的浏览器直下是否进行中 */
  function browserNowInProgress(): boolean {
    return browserDownloadProgress.value[BROWSER_NOW_ID] != null;
  }

  /** 本地面板浏览器直下的进度（0~1），null 表示未进行 */
  function browserNowProgressRatio(): number | null {
    const bp = browserDownloadProgress.value[BROWSER_NOW_ID];
    if (!bp) return null;
    return bp.total > 0 ? bp.done / bp.total : 0;
  }

  /**
   * 浏览器直下当前面板已解析的分片：不创建服务端任务，
   * 直接在浏览器内拉取当前分片列表、解密合并后另存到本机。
   */
  async function downloadHlsInBrowserNow() {
    const sources = hlsSources.value;
    const segments = hlsUsableSegments.value;
    if (sources.length === 0 || segments.length === 0) {
      notifyNegative('没有可下载的分片');
      return;
    }
    // SAMPLE-AES 无法在浏览器内解密：按源分别解出每个分片生效的密钥再校验
    const keysBySource = new Map(
      sources.map((item) => [item.id, resolveSegmentKeys(item.parsed)]),
    );
    const unsupported = segments.some(
      (seg) =>
        keysBySource.get(seg.sourceId)?.[seg.index - 1]?.method === 'SAMPLE-AES',
    );
    if (unsupported) {
      notifyNegative('该视频使用 SAMPLE-AES 加密，浏览器直下暂不支持');
      return;
    }
    const fileName = resolveDownloadName(
      hlsDownloadName.value,
      hlsDefaultDownloadName.value,
      hlsDownloadExt.value,
    );
    // 与服务端下载一致：重建播放列表（含分片删除与多源合并结果）
    const playlist = buildPlaylistText(sources, segments);
    const sourceUrl = hlsPrimaryUrl.value || hlsURL.value.trim();
    await browserDownloadFromPlaylist(
      BROWSER_NOW_ID,
      fileName,
      sourceUrl || undefined,
      playlist,
    );
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
    const sources = hlsSources.value;
    const segments = hlsUsableSegments.value;
    if (sources.length === 0 || segments.length === 0) {
      notifyNegative('没有可下载的分片');
      return;
    }

    // SAMPLE-AES 无法在服务端解密：按源分别解出每个分片生效的密钥再校验
    const keysBySource = new Map(
      sources.map((item) => [item.id, resolveSegmentKeys(item.parsed)]),
    );
    const unsupported = segments.some(
      (seg) =>
        keysBySource.get(seg.sourceId)?.[seg.index - 1]?.method ===
        'SAMPLE-AES',
    );
    if (unsupported) {
      notifyNegative('该视频使用 SAMPLE-AES 加密，暂不支持下载');
      return;
    }

    // 用户填了名字就用它（自动补扩展名），留空则回退到默认名
    const fileName = resolveDownloadName(
      hlsDownloadName.value,
      hlsDefaultDownloadName.value,
      hlsDownloadExt.value,
    );
    const sourceUrl = hlsPrimaryUrl.value || hlsURL.value.trim();
    // 重建播放列表（含分片删除结果与多源合并），交给服务端按序拉取
    const playlist = buildPlaylistText(sources, segments);
    const total = segments.length;

    hlsLoading.value = true;
    try {
      const res = await HlsDownloadAPI({
        playlist,
        sourceUrl,
        fileName,
        dir: hlsDownloadDir.value,
        xcode: hlsDownloadXcode.value || undefined,
      });
      if (res?.Code !== 200) {
        notifyNegative(res?.Message || '创建下载任务失败');
        return;
      }
      $q.notify({
        type: 'positive',
        message: `已提交服务端下载 · ${total} 个分片`,
        caption: '下载在服务端继续，分片列表已清空',
        position: 'top',
      });
      // 任务已交给服务端，本地这批解析结果随之作废：清空分片列表与输入框里的链接列表，
      // 方便直接粘贴下一组地址。下载列表保留——它是服务端任务的镜像，与本地状态无关。
      resetHlsParse();
      clearHlsUrlRows();
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
    // 分片链接多行输入（每行一个地址，带排序号，自动续行）
    hlsUrlRows,
    updateHlsUrlRow,
    clearHlsUrlRows,
    hlsLoading,
    hlsParsing,
    linkActionLabel,
    linkActionIcon,
    linkActionTooltip,
    linkActionLoading,
    // 分片列表（支持多个 m3u8 源按顺序合并）
    hlsParsed,
    hlsSegments,
    hlsSegmentGroups,
    hlsSourceList,
    hlsSourceCount,
    hlsTotalCount,
    hlsKeptCount,
    hlsRemovedCount,
    hlsKeptDuration,
    // 下载（服务端任务）
    hlsDownloadName,
    hlsDefaultDownloadName,
    hlsDownloadDir,
    hlsDownloadDirOptions,
    // 下载完成后自动转码（'' 不转 / copy / h264 / h265）
    hlsDownloadXcode,
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
    // 广告黑名单
    hlsAdBlacklist,
    hlsAdCount,
    isHlsAdSegment,
    markHlsAdSegments,
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
    refreshHlsDownloads,
    playHlsDownload,
    clearHlsDownloads,
    destroyHls,
    cleanup,
  };
}
