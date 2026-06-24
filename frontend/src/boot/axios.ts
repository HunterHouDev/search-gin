import { boot } from 'quasar/wrappers';
import axios, { AxiosInstance } from 'axios';
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

// 请求拦截器：自动添加token
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

export default boot(({ app, router }) => {
  // 响应拦截器：处理token过期 - 放在boot函数内，以便使用router实例
  api.interceptors.response.use(
    (response) => {
      return response;
    },
    (error) => {
      if (error.response && error.response.status === 401) {
        sessionStorage.removeItem('authToken');
        sessionStorage.removeItem('isAuthenticated');
        sessionStorage.removeItem('userRole');
        sessionStorage.removeItem('username');
        if (router) {
          router.push('/login');
        }
      }
      return Promise.reject(error);
    }
  );

  // for use inside Vue files (Options API) through this.$axios and this.$api

  app.config.globalProperties.$axios = axios;
  // ^ ^ ^ this will allow you to use this.$axios (for Vue Options API form)
  //       so you won't necessarily have to import axios in each vue file

  app.config.globalProperties.$api = api;
  // ^ ^ ^ this will allow you to use this.$api (for Vue Options API form)
  //       so you can easily perform requests against your app's API
});

const commonAxios = () => {
  return api;
};

export { api, axios, commonAxios };
