// 开窗收口 —— 应用内页面与外部地址走不同的开窗策略。
//
// 两条铁律：
// 1. 应用内页面（openAppWindow）必须携带本窗口登录态，新窗口才能免登录；
// 2. 外部地址（openExternalWindow）绝不携带登录态，避免把 token 泄漏给第三方站点。
//
// 浏览器与 Electron 的行为差异：Electron 下 window.open 无法设置主进程窗口参数，
// 统一改走 preload 暴露的 window.electron 通道（见 src-electron/windows/index.ts）。

import { buildSessionUrl } from 'src/utils/authStorage';

/** Electron 主进程窗口参数（与 src-electron/windows/index.ts 的 SonWindowParam 对齐） */
interface ElectronWindowParam {
  router: string;
  width?: number;
  height?: number;
  titleBarStyle?: string;
}

/** preload 暴露的开窗通道（见 src-electron/electron-preload.ts） */
interface ElectronBridge {
  createWindow: (param: ElectronWindowParam) => void;
  openBySystem: (param: { Path: string }) => void;
}

/** 开窗选项：结构化字段优先，features 为 window.open 原始语法 */
export interface WindowOpenOptions {
  /** window.open 的 target，Electron 下忽略 */
  target?: string;
  /** window.open 的 features 原始串，形如 "width=1280,height=720,titleBarStyle=" */
  features?: string;
  width?: number;
  height?: number;
  titleBarStyle?: string;
}

const HTTP_URL_RE = /^https?:\/\//i;

function getElectronBridge(): ElectronBridge | undefined {
  return (window as unknown as { electron?: ElectronBridge }).electron;
}

/** 解析 features 串中 Electron 关心的窗口参数 */
function parseFeatures(features?: string): WindowOpenOptions {
  const parsed: WindowOpenOptions = {};
  if (!features) return parsed;
  for (const pair of features.split(',')) {
    const [rawKey, ...rest] = pair.split('=');
    const key = rawKey.trim();
    const value = rest.join('=').trim();
    if (key === 'width') {
      const size = Number(value);
      if (Number.isFinite(size) && size > 0) parsed.width = size;
    } else if (key === 'height') {
      const size = Number(value);
      if (Number.isFinite(size) && size > 0) parsed.height = size;
    } else if (key === 'titleBarStyle') {
      parsed.titleBarStyle = value;
    }
  }
  return parsed;
}

/** 合并为 window.open 的 features 串（结构化字段覆盖同名的原始项） */
function buildFeatures(options: WindowOpenOptions): string {
  const pairs = (options.features ?? '')
    .split(',')
    .map((pair) => pair.trim())
    .filter(Boolean)
    .filter((pair) => {
      const key = pair.split('=')[0].trim();
      return !(
        (key === 'width' && options.width !== undefined) ||
        (key === 'height' && options.height !== undefined) ||
        (key === 'titleBarStyle' && options.titleBarStyle !== undefined)
      );
    });
  if (options.width !== undefined) pairs.push(`width=${options.width}`);
  if (options.height !== undefined) pairs.push(`height=${options.height}`);
  if (options.titleBarStyle !== undefined) pairs.push(`titleBarStyle=${options.titleBarStyle}`);
  return pairs.join(',');
}

function toElectronParam(url: string, options: WindowOpenOptions): ElectronWindowParam {
  const parsed = parseFeatures(options.features);
  const width = options.width ?? parsed.width;
  const height = options.height ?? parsed.height;
  const titleBarStyle = options.titleBarStyle ?? parsed.titleBarStyle;
  const param: ElectronWindowParam = { router: url };
  if (width !== undefined) param.width = width;
  if (height !== undefined) param.height = height;
  if (titleBarStyle !== undefined) param.titleBarStyle = titleBarStyle;
  return param;
}

/** 打开应用内页面窗口（播放页 / 作者搜索页等），携带一次性登录态快照 */
export function openAppWindow(url: string, options: WindowOpenOptions = {}): Window | null {
  const target = buildSessionUrl(url);
  const electron = getElectronBridge();
  if (electron) {
    electron.createWindow(toElectronParam(target, options));
    return null;
  }
  return window.open(target, options.target ?? '_blank', buildFeatures(options));
}

/**
 * 打开外部地址窗口，绝不携带登录态。
 * Electron 下本地路径（非 http）交给系统默认程序，避免被当成应用内路由加载。
 */
export function openExternalWindow(url: string, options: WindowOpenOptions = {}): Window | null {
  const electron = getElectronBridge();
  if (electron) {
    if (!HTTP_URL_RE.test(url)) {
      electron.openBySystem({ Path: url });
      return null;
    }
    electron.createWindow(toElectronParam(url, options));
    return null;
  }
  return window.open(url, options.target ?? '_blank', buildFeatures(options));
}
