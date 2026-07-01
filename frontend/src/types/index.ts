// ── 搜索参数 ──────────────────────────────────────────────────────

export interface SearchParams {
  Keyword: string
  MovieType: string
  Author: string
  SortField: string
  SortType: string
  Page: number
  PageSize: number
  OnlyRepeat: boolean
}

// ── 文件条目 ──────────────────────────────────────────────────────

export interface FileItem {
  Id: string
  Name: string
  Title: string
  Path: string
  DirPath: string
  BaseDir: string
  Code: string
  Author: string
  Studio: string
  MovieType: string
  FileType: string
  Tags: string[]
  Size: number
  SizeStr: string
  MTime: string
  Jpg: string
  Png: string
  Gif: string
  StreamUrl: string
  PngUrl: string
  JpgUrl: string
  NodeHost: string
  NodeName: string
  PageNo: number
}

// ── 操作结果 ──────────────────────────────────────────────────────

export interface ApiResult {
  Code: number
  Message: string
  Data?: unknown
}

export interface PageResult<T = unknown> {
  TotalCnt: number
  TotalSize: string
  PageSize: number
  Data: T[]
  CurCnt: number
  CurSize: string
}

// ── 设置 ──────────────────────────────────────────────────────────

export interface SettingInfo {
  AdminPassword?: string
  Dirs: string[]
  Tags: string[]
  TagsLib: string[]
  VideoTypes: string[]
  ImageTypes: string[]
  DocsTypes: string[]
  Types: string[]
  MovieTypes: string[]
  ControllerHost: string
  FileHost: string
  EnableLanDiscovery: boolean | null
  NodeName: string
  DiscoveryPeers: string[]
  HardwareAcceleration: boolean
  HardwareAccelMode: string
  EnableTimeScan: boolean
  SystemPlayer: string
  SystemPlayerVolumn: string
  SystemPlayerWidth: string
  CutThenDelete: boolean
}

// ── SSE / WebSocket 事件枚举 ───────────────────────────────────────

export enum SSEEventType {
  ScanStart = 'scan_start',
  ScanComplete = 'scan_complete',
  ScanOneDone = 'scan_one_done',
  RenameStart = 'rename_start',
  ScanError = 'scan_error',
  FileChanged = 'file_changed',
  IndexUpdate = 'index_update',
  IndexHealth = 'index_health',
  TaskLog = 'task_log',
}

export enum WSMessageType {
  Chat = 'chat',
  Online = 'online',
  System = 'system',
  Signal = 'signal',
  SignalAll = 'signal-all',
}

export enum SignalAction {
  Join = 'join',
  Leave = 'leave',
  Offer = 'offer',
  Answer = 'answer',
  Ice = 'ice',
}

// ── 节点信息 ──────────────────────────────────────────────────────

export interface PeerInfo {
  ID: string
  IP: string
  Port: string
  Name: string
  FilePort: string
  LastSeen: number
  Disabled?: boolean
}
