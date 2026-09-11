/* eslint-disable */

declare namespace NodeJS {
  interface ProcessEnv {
    NODE_ENV: string;
    VUE_ROUTER_MODE: 'hash' | 'history' | 'abstract' | undefined;
    VUE_ROUTER_BASE: string | undefined;
  }
}

/**
 * preload 通过 contextBridge.exposeInMainWorld('electron', ...) 暴露的桥接 API。
 * 方法名与 src-electron/windows/func.ts 的 ipcMain 监听一一对应，改动需同步三处。
 */
interface ElectronBridge {
  createWindow: (param: { router: string; width?: number; height?: number; titleBarStyle?: string }) => void;
  openBySystem: (param: { Path: string }) => void;
  maxMainWindow: () => void;
  hideMainWindow: () => void;
  resizeMainWindow: () => void;
  showInFolder: (path: string) => void;
}

interface Window {
  /** 仅 Electron 环境下由 preload 注入，浏览器环境为 undefined（调用前需判 $q.platform.is.electron） */
  electron: ElectronBridge;
}
