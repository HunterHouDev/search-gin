import { boot } from 'quasar/wrappers';
import axios, { AxiosInstance, type AxiosError } from 'axios';
import { useQuasar } from 'quasar';
import { isElectron } from './platform';
import { consumeSessionFromUrl, getAuthToken, logout } from 'src/utils/authStorage';

declare module '@vue/runtime-core' {
  interface ComponentCustomProperties {
    $axios: AxiosInstance;
    $api: AxiosInstance;
  }
}

// Electron 下直连后端，浏览器下用相对路径走 devServer proxy
const api = axios.create({
  baseURL: isElectron() ? 'http://localhost:10081' : '',
  timeout: 30000,
});

/** 统一错误类型：附加超时标记、可直接展示的提示语，以及「已提示过」标记 */
export type ApiError = AxiosError & {
  isTimeout?: boolean;
  friendlyMessage?: string;
  /** 拦截器已弹过提示：调用方未 catch 时不再重复提示，也不污染控制台 */
  __notified?: boolean;
};

/**
 * 拦截器已提示过的错误走这里 reject。
 * 调用方无需再 try/catch，未捕获的 rejection 由 unhandledrejection 兜底静默。
 */
function rejectNotified(error: unknown) {
  (error as ApiError).__notified = true;
  return Promise.reject(error);
}

/** 超时判定：浏览器侧 XHR 超时为 ECONNABORTED，axios 的超时错误信息形如 timeout of xxxms exceeded */
function isTimeoutError(error: unknown): boolean {
  const e = error as AxiosError;
  return (
    e?.code === 'ECONNABORTED' ||
    e?.code === 'ETIMEDOUT' ||
    /timeout of \d+\s*ms exceeded/i.test(e?.message || '')
  );
}

/**
 * 取接口错误的展示文案：超时固定为「请求超时」，
 * 其余优先用后端返回的 Message，最后兜底 axios 的 message。
 */
export function axiosErrorMessage(error: unknown, fallback = '请求失败'): string {
  const e = error as ApiError;
  if (e?.friendlyMessage) return e.friendlyMessage;
  const data = e?.response?.data as
    | { Message?: string; message?: string; msg?: string }
    | undefined;
  return data?.Message || data?.message || data?.msg || e?.message || fallback;
}

// 请求拦截器：自动添加 token（新窗口经 URL 桥接把登录态写入本窗口 sessionStorage）
api.interceptors.request.use(
  (config) => {
    const token = getAuthToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 导出 $q 供 response 拦截器使用（需在 boot 函数内注入）
let $q: ReturnType<typeof useQuasar>;

export default boot(({ app, router }) => {
  $q = useQuasar();

  // 新窗口 / 新标签页：从 URL 继承登录态，并立即抹掉地址栏里的凭据
  consumeSessionFromUrl();

  // 响应拦截器：统一错误处理 + Token 过期跳转
  api.interceptors.response.use(
    (response) => response,
    (error) => {
      // 超时最先判定：没有响应状态码，且提示必须明确为「请求超时」
      if (isTimeoutError(error)) {
        (error as ApiError).isTimeout = true;
        (error as ApiError).friendlyMessage = '请求超时';
      }
      const status = error?.response?.status;
      const data = error?.response?.data;
      const msg = data?.Message || data?.message || data?.msg || '';

      if (status === 412) {
        // 系统未初始化 → 跳转初始化页
        if (router) {
          router.push('/init');
        }
        return rejectNotified(error);
      }

      if (status === 401) {
        // Token 过期：静默清理本窗口登录态并跳转登录页
        logout(router);
        return rejectNotified(error);
      }

      const notify = (opts: Parameters<typeof $q.notify>[0]) => {
        if ($q) {
          $q.notify(opts);
        }
      };

      if (status === 403) {
        notify({
          type: 'warning',
          message: msg || '无权限执行此操作',
          position: 'top',
          timeout: 3000,
        });
        return rejectNotified(error);
      }

      if (status && status >= 400 && status < 500) {
        notify({
          type: 'negative',
          message: msg || `请求错误 (${status})`,
          position: 'top',
          timeout: 3000,
        });
        return rejectNotified(error);
      }

      if (status && status >= 500) {
        notify({
          type: 'negative',
          message: msg || `服务器错误 (${status})，请稍后重试`,
          position: 'top',
          timeout: 4000,
        });
        return rejectNotified(error);
      }

      // 超时 / 网络断开
      if ((error as ApiError).isTimeout) {
        notify({
          type: 'warning',
          message: '请求超时，请稍后重试',
          position: 'top',
          timeout: 3000,
        });
      } else if (!status) {
        notify({
          type: 'negative',
          message: '网络连接失败，请检查服务器状态',
          position: 'top',
          timeout: 4000,
        });
      }

      return rejectNotified(error);
    }
  );

  // 兜底：调用方未 catch 的请求错误统一在这里收口。
  // 拦截器已提示过的直接静默（避免控制台刷未捕获异常），
  // 漏网的请求错误补一次提示；非请求类异常交还给控制台。
  window.addEventListener('unhandledrejection', (event) => {
    const err = event.reason as ApiError | undefined;
    const isRequestError =
      !!err && (err.isAxiosError === true || !!err.config || !!err.response);
    if (!isRequestError) return;
    event.preventDefault();
    if (err.__notified || err.isTimeout) return;
    $q?.notify({
      type: 'negative',
      message: axiosErrorMessage(err),
      position: 'top',
      timeout: 3000,
    });
  });

  // Vue Options API 全局注入
  app.config.globalProperties.$axios = axios;
  app.config.globalProperties.$api = api;
});

/** 获取通用 axios 实例 */
const commonAxios = () => api;

export { api, axios, commonAxios };
