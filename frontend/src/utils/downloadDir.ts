// 记住下载目录（基于 File System Access API），让「另存为」不必每次重新选目录。
//
// 原理：showDirectoryPicker() 返回的 FileSystemDirectoryHandle 可以结构化克隆，
// 存进 IndexedDB 后下次仍能取出来用；只要该目录还持有 readwrite 权限，就能
// getFileHandle(name, { create: true }) + createWritable() 直接落盘，全程不弹对话框。
//
// 浏览器安全策略带来的边界（绕不过去，只能优雅降级）：
// 1. 仅 Chromium 桌面版 + 安全上下文（https / localhost）可用，其余环境回退浏览器下载；
// 2. 目录授权默认随会话失效，重启浏览器后首次使用需要点一次「允许」——
//    那是一次权限确认，不是重新浏览目录，比每次重选目录轻得多。

/** 文件系统选择器（lib.dom 尚未声明，按用到的字段最小声明） */
interface FsAccessWindow {
  showDirectoryPicker?: (options?: {
    id?: string;
    mode?: 'read' | 'readwrite';
    startIn?: string | FileSystemHandle;
  }) => Promise<FileSystemDirectoryHandle>;
  showSaveFilePicker?: (options?: {
    id?: string;
    suggestedName?: string;
    startIn?: string | FileSystemHandle;
  }) => Promise<FileSystemFileHandle>;
}

/** 句柄权限的查询与申请（lib.dom 尚未声明） */
interface PermissionAwareHandle {
  queryPermission: (desc: {
    mode: 'read' | 'readwrite';
  }) => Promise<PermissionState>;
  requestPermission: (desc: {
    mode: 'read' | 'readwrite';
  }) => Promise<PermissionState>;
}

const DB_NAME = 'search-gin-fs';
const DB_VERSION = 1;
const STORE_NAME = 'handles';
const DIR_KEY = 'download-dir';

/** 选择器 id：Chrome 会据此记住上次使用的目录（即使没存 IndexedDB 也有效果） */
const PICKER_ID = 'search-gin-download';

/** 同名文件的退避上限，超出后改用时间戳 */
const MAX_DUPLICATE_INDEX = 99;

function pickerWindow(): FsAccessWindow {
  return window as unknown as FsAccessWindow;
}

/** 环境是否支持目录选择（决定是否展示「选择目录」入口） */
export const fsDirSupported =
  typeof window !== 'undefined' &&
  typeof pickerWindow().showDirectoryPicker === 'function';

/** 环境是否支持「另存为」对话框 */
export const fsSaveSupported =
  typeof window !== 'undefined' &&
  typeof pickerWindow().showSaveFilePicker === 'function';

// ── IndexedDB：存放目录句柄（句柄不能序列化进 localStorage，只能存 IndexedDB） ──

let dbPromise: Promise<IDBDatabase> | null = null;

function createDb(): Promise<IDBDatabase> {
  return new Promise<IDBDatabase>((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME);
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () =>
      reject(request.error ?? new Error('打开 IndexedDB 失败'));
  });
}

function openDb(): Promise<IDBDatabase> {
  if (typeof indexedDB === 'undefined') {
    return Promise.reject(new Error('当前环境不支持 IndexedDB'));
  }
  if (dbPromise) return dbPromise;
  const pending = createDb().catch((e: unknown) => {
    // 打开失败后允许下次重试，避免一次失败永久不可用
    dbPromise = null;
    throw e;
  });
  dbPromise = pending;
  return pending;
}

function runRequest<T>(
  mode: IDBTransactionMode,
  run: (store: IDBObjectStore) => IDBRequest,
): Promise<T> {
  return openDb().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const tx = db.transaction(STORE_NAME, mode);
        const request = run(tx.objectStore(STORE_NAME));
        request.onsuccess = () => resolve(request.result as T);
        request.onerror = () =>
          reject(request.error ?? new Error('IndexedDB 请求失败'));
        tx.onabort = () =>
          reject(tx.error ?? new Error('IndexedDB 事务被中止'));
      }),
  );
}

async function readStoredDir(): Promise<FileSystemDirectoryHandle | null> {
  const value = await runRequest<FileSystemDirectoryHandle | undefined>(
    'readonly',
    (store) => store.get(DIR_KEY),
  );
  // 旧版本或异常数据可能不是目录句柄，按未保存处理
  return value && typeof value.getFileHandle === 'function' ? value : null;
}

async function writeStoredDir(dir: FileSystemDirectoryHandle): Promise<void> {
  await runRequest('readwrite', (store) => store.put(dir, DIR_KEY));
}

async function deleteStoredDir(): Promise<void> {
  await runRequest('readwrite', (store) => store.delete(DIR_KEY));
}

// ── 对外能力 ──────────────────────────────────────────────────────────────────

/** 读取记住的下载目录；不可用或读取失败时返回 null */
export async function loadSavedDir(): Promise<FileSystemDirectoryHandle | null> {
  if (!fsDirSupported) return null;
  try {
    return await readStoredDir();
  } catch {
    return null;
  }
}

/** 记住下载目录（失败只影响下次记忆，不影响本次下载） */
export async function saveDir(dir: FileSystemDirectoryHandle): Promise<void> {
  try {
    await writeStoredDir(dir);
  } catch {
    /* 隐私模式等场景下忽略 */
  }
}

/** 忘记下载目录（句柄失效或用户更改目录时调用） */
export async function forgetDir(): Promise<void> {
  try {
    await deleteStoredDir();
  } catch {
    /* 忽略清除失败 */
  }
}

/**
 * 弹出目录选择器并记住选择结果。
 * 返回 null 表示环境不支持；用户取消会抛出 AbortError，由调用方忽略。
 */
export async function pickDir(): Promise<FileSystemDirectoryHandle | null> {
  const picker = pickerWindow().showDirectoryPicker;
  if (typeof picker !== 'function') return null;
  const dir = await picker.call(window, {
    id: PICKER_ID,
    mode: 'readwrite',
    startIn: 'downloads',
  });
  await saveDir(dir);
  return dir;
}

/**
 * 确认目录可写：已授权直接返回 true；未授权时申请一次。
 * requestPermission 需要用户手势，必须在点击事件的处理链上调用。
 */
export async function ensureDirWritable(
  dir: FileSystemDirectoryHandle,
): Promise<boolean> {
  const handle = dir as FileSystemDirectoryHandle &
    Partial<PermissionAwareHandle>;
  if (typeof handle.queryPermission !== 'function') return false;
  try {
    const state = await handle.queryPermission({ mode: 'readwrite' });
    if (state === 'granted') return true;
    if (state === 'denied') return false;
    if (typeof handle.requestPermission !== 'function') return false;
    const granted = await handle.requestPermission({ mode: 'readwrite' });
    return granted === 'granted';
  } catch {
    return false;
  }
}

/** 目录里是否已存在同名条目（目录也算占用，避免 getFileHandle 抛类型错误） */
async function entryExists(
  dir: FileSystemDirectoryHandle,
  name: string,
): Promise<boolean> {
  try {
    await dir.getFileHandle(name);
    return true;
  } catch (e) {
    // 只有 NotFoundError 能确定是可用的新名字
    return (e as DOMException)?.name !== 'NotFoundError';
  }
}

/** 同名时追加 (1)(2)…，避免直接覆盖用户已有文件 */
async function findAvailableName(
  dir: FileSystemDirectoryHandle,
  fileName: string,
): Promise<string> {
  const dot = fileName.lastIndexOf('.');
  const stem = dot > 0 ? fileName.slice(0, dot) : fileName;
  const ext = dot > 0 ? fileName.slice(dot) : '';
  for (let i = 0; i <= MAX_DUPLICATE_INDEX; i++) {
    const candidate = i === 0 ? fileName : `${stem} (${i})${ext}`;
    if (!(await entryExists(dir, candidate))) return candidate;
  }
  return `${stem} (${Date.now()})${ext}`;
}

export interface DirWritable {
  stream: FileSystemWritableFileStream;
  /** 实际落盘的文件名（同名冲突时带序号） */
  fileName: string;
}

/** 在已授权目录里开一个可写文件流 */
export async function openWritableInDir(
  dir: FileSystemDirectoryHandle,
  fileName: string,
): Promise<DirWritable> {
  const actual = await findAvailableName(dir, fileName);
  const handle = await dir.getFileHandle(actual, { create: true });
  const stream = await handle.createWritable();
  return { stream, fileName: actual };
}

export type SavePickResult =
  | { status: 'ok'; handle: FileSystemFileHandle }
  | { status: 'cancelled' }
  | { status: 'failed' };

/**
 * 弹出「另存为」对话框。
 * 传入 startIn 时优先定位到该目录；句柄失效会被浏览器拒绝，此时退回默认位置再弹一次。
 */
export async function pickSaveFile(
  fileName: string,
  startIn?: FileSystemHandle | string,
): Promise<SavePickResult> {
  const picker = pickerWindow().showSaveFilePicker;
  if (typeof picker !== 'function') return { status: 'failed' };

  try {
    const handle = await picker.call(window, {
      id: PICKER_ID,
      suggestedName: fileName,
      ...(startIn ? { startIn } : {}),
    });
    return { status: 'ok', handle };
  } catch (e) {
    if ((e as DOMException)?.name === 'AbortError') return { status: 'cancelled' };
    if (!startIn) return { status: 'failed' };
  }

  // startIn 相关的失败（句柄被移动 / 权限失效）退化为普通另存为
  try {
    const handle = await picker.call(window, {
      id: PICKER_ID,
      suggestedName: fileName,
    });
    return { status: 'ok', handle };
  } catch (e) {
    return (e as DOMException)?.name === 'AbortError'
      ? { status: 'cancelled' }
      : { status: 'failed' };
  }
}
