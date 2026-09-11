// 登录态存储 —— 按窗口隔离（sessionStorage），新窗口经 URL 一次性桥接继承。
//
// 为什么不用 localStorage：
// 同一台机器可能同时开多个窗口用不同账号（主窗口 admin、子窗口普通用户），
// 共享存储会让后登录的窗口顶掉先登录的窗口，因此每个窗口各持一份副本。
//
// 新窗口如何免登录：openAppWindow() 把登录态快照编码进 URL（buildSessionUrl），
// 新窗口启动时由 consumeSessionFromUrl() 解析并写入本窗口 sessionStorage，
// 随即用 replaceState 抹掉地址栏与历史记录中的凭据。

const TOKEN_KEY = 'authToken';
const ROLE_KEY = 'authRole';
const USERNAME_KEY = 'authUsername';
const PERMISSIONS_KEY = 'authPermissions';
const EXPIRE_AT_KEY = 'authExpireAt';
const EXPIRE_IN_KEY = 'authExpireIn';

/** 新窗口继承登录态所用的 URL 参数名 */
export const SESSION_URL_PARAM = 'auth';

/** 默认有效期（秒）—— 与后端 auth_service.go issueToken 的 4 小时保持一致 */
const DEFAULT_EXPIRE_IN = 4 * 3600;

/** 登录态快照 */
export interface AuthSession {
  token: string;
  role: string;
  username: string;
  permissions: string[];
  /** 过期时间戳（毫秒） */
  expireAt: number;
}

/** 写入登录态的入参 */
export interface SaveAuthSessionParam {
  token: string;
  role: string;
  username: string;
  permissions?: string[];
  /** 服务端返回的有效期（秒），缺省 4 小时 */
  expireIn?: number;
}

/** URL 桥接载荷（字段逐个校验，避免脏数据污染存储） */
interface RawSessionPayload {
  token?: unknown;
  role?: unknown;
  username?: unknown;
  permissions?: unknown;
  expireAt?: unknown;
}

// ── 底层读写（sessionStorage 在隐私模式 / 沙箱下可能抛异常，统一兜底） ──

function read(key: string): string {
  try {
    return sessionStorage.getItem(key) ?? '';
  } catch {
    return '';
  }
}

function write(key: string, value: string): void {
  try {
    sessionStorage.setItem(key, value);
  } catch {
    // 存储不可用时本窗口内存态仍在，接口 401 会兜底
  }
}

function remove(key: string): void {
  try {
    sessionStorage.removeItem(key);
  } catch {
    // ignore
  }
}

// ── 登录态读写 ────────────────────────────────────────────────────

/** 写入本窗口登录态（登录成功 / URL 桥接继承时调用） */
export function saveAuthSession({
  token,
  role,
  username,
  permissions = [],
  expireIn,
}: SaveAuthSessionParam): void {
  if (!token) {
    clearAuthSession();
    return;
  }
  const seconds = expireIn && expireIn > 0 ? expireIn : DEFAULT_EXPIRE_IN;
  write(TOKEN_KEY, token);
  write(ROLE_KEY, role);
  write(USERNAME_KEY, username);
  write(PERMISSIONS_KEY, JSON.stringify(permissions));
  write(EXPIRE_AT_KEY, String(Date.now() + seconds * 1000));
  write(EXPIRE_IN_KEY, String(seconds));
}

/** 清空本窗口登录态 */
export function clearAuthSession(): void {
  remove(TOKEN_KEY);
  remove(ROLE_KEY);
  remove(USERNAME_KEY);
  remove(PERMISSIONS_KEY);
  remove(EXPIRE_AT_KEY);
  remove(EXPIRE_IN_KEY);
}

/** 登出：先通知服务端吊销 token，再清空本窗口登录态并回到登录页 */
export function logout(router?: { push: (to: string) => unknown }): void {
  revokeServerToken(getAuthToken());
  clearAuthSession();
  if (router) {
    void router.push('/login');
    return;
  }
  window.location.href = '/#/login';
}

/** 本窗口是否持有未过期的登录态 */
export function isLoggedIn(): boolean {
  if (!getAuthToken()) return false;
  const expireAt = getAuthExpireAt();
  return expireAt === 0 || expireAt > Date.now();
}

/** 当前 token（未登录返回空串，可直接用于请求头判断） */
export function getAuthToken(): string {
  return read(TOKEN_KEY);
}

/** 当前用户名 */
export function getAuthUsername(): string {
  return read(USERNAME_KEY);
}

/** 当前角色 */
export function getAuthRole(): string {
  return read(ROLE_KEY);
}

/** 当前权限列表（解析失败返回空数组） */
export function getAuthPermissions(): string[] {
  const raw = read(PERMISSIONS_KEY);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    return Array.isArray(parsed) ? parsed.filter((p): p is string => typeof p === 'string') : [];
  } catch {
    return [];
  }
}

/** 过期时间戳（毫秒）；0 表示无到期信息 */
export function getAuthExpireAt(): number {
  const value = Number(read(EXPIRE_AT_KEY));
  return Number.isFinite(value) && value > 0 ? value : 0;
}

/** 用户有操作时续期本窗口倒计时（服务端 token 有效期不随之延长，401 会兜底） */
export function refreshAuthActivity(): void {
  if (!getAuthToken()) return;
  const stored = Number(read(EXPIRE_IN_KEY));
  const seconds = Number.isFinite(stored) && stored > 0 ? stored : DEFAULT_EXPIRE_IN;
  write(EXPIRE_AT_KEY, String(Date.now() + seconds * 1000));
}

/** 读取完整快照；未登录返回 null */
export function snapshotAuthSession(): AuthSession | null {
  if (!isLoggedIn()) return null;
  return {
    token: getAuthToken(),
    role: getAuthRole(),
    username: getAuthUsername(),
    permissions: getAuthPermissions(),
    expireAt: getAuthExpireAt(),
  };
}

// ── 新窗口桥接（URL 一次性携带登录态） ────────────────────────────

/** 把当前登录态快照附加到 url 上（未登录时原样返回） */
export function buildSessionUrl(url: string): string {
  const session = snapshotAuthSession();
  if (!session) return url;
  const param = `${SESSION_URL_PARAM}=${encodeSession(session)}`;
  const hashIndex = url.indexOf('#');
  if (hashIndex < 0) {
    return url + (url.includes('?') ? '&' : '?') + param;
  }
  // hash 路由：参数必须落在 hash 段内部，否则 vue-router 读不到
  const hash = url.slice(hashIndex);
  return `${url.slice(0, hashIndex)}${hash}${hash.includes('?') ? '&' : '?'}${param}`;
}

/** 从当前 URL 继承登录态，并立即抹掉地址栏与历史记录中的凭据（启动时调用一次） */
export function consumeSessionFromUrl(): void {
  const search = window.location.search.replace(/^\?/, '');
  const hash = window.location.hash.replace(/^#/, '');
  const queryIndex = hash.indexOf('?');
  const hashPath = queryIndex >= 0 ? hash.slice(0, queryIndex) : hash;
  const hashQuery = queryIndex >= 0 ? hash.slice(queryIndex + 1) : '';

  const fromSearch = takeSessionParam(search);
  const fromHash = takeSessionParam(hashQuery);
  const carrier = fromHash.raw ? fromHash : fromSearch;
  if (!carrier.raw) return;

  const session = decodeSession(carrier.raw);
  if (session && session.expireAt > Date.now()) {
    saveAuthSession({
      token: session.token,
      role: session.role,
      username: session.username,
      permissions: session.permissions,
      expireIn: Math.max(1, Math.floor((session.expireAt - Date.now()) / 1000)),
    });
  }

  const nextSearch = fromSearch.raw ? fromSearch.rest : search;
  const nextHashQuery = fromHash.raw ? fromHash.rest : hashQuery;
  const nextHash = hashPath + (nextHashQuery ? `?${nextHashQuery}` : '');
  const nextUrl = `${window.location.pathname}${nextSearch ? `?${nextSearch}` : ''}${
    nextHash ? `#${nextHash}` : ''
  }`;
  window.history.replaceState(window.history.state, '', nextUrl);
}

// ── 内部工具 ──────────────────────────────────────────────────────

/**
 * 通知服务端立即吊销 token（尽力而为）。
 * 用原生 fetch 而非项目 axios 实例：登出链路不能反过来被 401 拦截器影响，
 * 否则会形成 拦截器 → logout → 接口 → 401 → 拦截器 的递归。
 * keepalive 保证跳转登录页时请求不被中断；失败也不阻塞本地登出。
 */
function revokeServerToken(token: string): void {
  if (!token) return;
  try {
    void fetch('/api/logout', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      keepalive: true,
    }).catch(() => {
      // 网络不可用时忽略：本地登出照常完成，服务端 token 到期后自然失效
    });
  } catch {
    // fetch 不可用（极端沙箱环境）时忽略
  }
}

/** 取出并移除 query 中的登录态参数 */
function takeSessionParam(query: string): { raw: string | null; rest: string } {
  if (!query) return { raw: null, rest: '' };
  const params = new URLSearchParams(query);
  const raw = params.get(SESSION_URL_PARAM);
  if (!raw) return { raw: null, rest: '' };
  params.delete(SESSION_URL_PARAM);
  return { raw, rest: params.toString() };
}

/** base64url（UTF-8 安全，可安全放进 URL 参数） */
function encodeSession(session: AuthSession): string {
  const bytes = new TextEncoder().encode(JSON.stringify(session));
  let binary = '';
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function decodeSession(raw: string): AuthSession | null {
  try {
    const binary = atob(raw.replace(/-/g, '+').replace(/_/g, '/'));
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
    const parsed = JSON.parse(new TextDecoder().decode(bytes)) as RawSessionPayload;
    if (typeof parsed.token !== 'string' || !parsed.token) return null;
    return {
      token: parsed.token,
      role: typeof parsed.role === 'string' ? parsed.role : '',
      username: typeof parsed.username === 'string' ? parsed.username : '',
      permissions: Array.isArray(parsed.permissions)
        ? parsed.permissions.filter((p): p is string => typeof p === 'string')
        : [],
      expireAt: typeof parsed.expireAt === 'number' ? parsed.expireAt : 0,
    };
  } catch {
    return null;
  }
}
