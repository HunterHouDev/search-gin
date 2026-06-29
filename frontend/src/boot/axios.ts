import { boot } from 'quasar/wrappers';
import axios, { AxiosInstance } from 'axios';
import { useQuasar } from 'quasar';
import { isElectron } from './platform';

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

// 请求拦截器：自动添加 token
api.interceptors.request.use(
  (config) => {
    const token = sessionStorage.getItem('authToken');
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

  // 响应拦截器：统一错误处理 + Token 过期跳转
  api.interceptors.response.use(
    (response) => response,
    (error) => {
      const status = error?.response?.status;
      const data = error?.response?.data;
      const msg = data?.Message || data?.message || data?.msg || '';

      if (status === 401) {
        // Token 过期 → 静默清理并跳转登录
        sessionStorage.removeItem('authToken');
        sessionStorage.removeItem('isAuthenticated');
        sessionStorage.removeItem('userRole');
        sessionStorage.removeItem('username');
        sessionStorage.removeItem('userPermissions');
        if (router) {
          router.push('/login');
        }
        return Promise.reject(error);
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
        return Promise.reject(error);
      }

      if (status && status >= 400 && status < 500) {
        notify({
          type: 'negative',
          message: msg || `请求错误 (${status})`,
          position: 'top',
          timeout: 3000,
        });
        return Promise.reject(error);
      }

      if (status && status >= 500) {
        notify({
          type: 'negative',
          message: msg || `服务器错误 (${status})，请稍后重试`,
          position: 'top',
          timeout: 4000,
        });
        return Promise.reject(error);
      }

      // 网络断开 / 超时
      if (error.code === 'ECONNABORTED') {
        notify({
          type: 'warning',
          message: '请求超时，请检查网络连接',
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

      return Promise.reject(error);
    }
  );

  // Vue Options API 全局注入
  app.config.globalProperties.$axios = axios;
  app.config.globalProperties.$api = api;
});

/** 获取通用 axios 实例 */
const commonAxios = () => api;

export { api, axios, commonAxios };
