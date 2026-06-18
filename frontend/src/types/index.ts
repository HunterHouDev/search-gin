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
