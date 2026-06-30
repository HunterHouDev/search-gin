import { route } from 'quasar/wrappers';
import {
  createMemoryHistory,
  createRouter,
  createWebHashHistory,
  createWebHistory,
} from 'vue-router';
import { commonAxios } from 'src/boot/axios';
import { usePermissionStore } from 'src/stores/permission';


import routes from './routes';

/*
 * If not building with SSR mode, you can
 * directly export the Router instantiation;
 *
 * The function below can be async too; either use
 * async/await or return a Promise which resolves
 * with the Router instance.
 */

export default route(function (/* { store, ssrContext } */) {
  const createHistory = process.env.SERVER
    ? createMemoryHistory
    : process.env.VUE_ROUTER_MODE === 'history'
    ? createWebHistory
    : createWebHashHistory;

  const Router = createRouter({
    scrollBehavior: () => ({ left: 0, top: 0 }),
    routes,
    // Leave this as is and make changes in quasar.conf.js instead!
    // quasar.conf.js -> build -> vueRouterMode
    // quasar.conf.js -> build -> publicPath
    history: createHistory(process.env.VUE_ROUTER_BASE),
  });

  Router.beforeEach(async (to, from, next) => {
    // 初始化页面不检查
    if (to.path === '/init') {
      next();
      return;
    }

    // 检查是否已完成初始化
    try {
      const res = await commonAxios().get('/api/init');
      if (!res.data?.Data) {
        next('/init');
        return;
      }
    } catch {
      // 网络错误时放行（可能后端未启动）
    }

    if (to.path === '/login') {
      sessionStorage.removeItem('authToken');
      sessionStorage.removeItem('isAuthenticated');
      sessionStorage.removeItem('userRole');
      sessionStorage.removeItem('username');
      next();
      return;
    }

    const isAuthenticated = sessionStorage.getItem('isAuthenticated');
    const token = sessionStorage.getItem('authToken');

    if (!isAuthenticated || !token) {
      next('/login');
      return;
    }

    const permRoutes: Record<string, string> = {
      '/setting': 'menu:setting',
      '/system': 'menu:system',
    };
    const requiredPerm = permRoutes[to.path];
    if (requiredPerm) {
      const permStore = usePermissionStore()
      permStore.loadFromSession()
      if (!permStore.hasPermission(requiredPerm)) {
        next('/');
        return;
      }
    }

    next();
  });

  return Router;
});
