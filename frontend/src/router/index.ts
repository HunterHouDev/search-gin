import { route } from 'quasar/wrappers';
import {
  createMemoryHistory,
  createRouter,
  createWebHashHistory,
  createWebHistory,
} from 'vue-router';
import { usePermissionStore } from 'src/stores/permission';
import { clearAuthSession, isLoggedIn } from 'src/utils/authStorage';

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
  const createHistory = import.meta.env.QUASAR_SERVER
    ? createMemoryHistory
    : import.meta.env.QUASAR_VUE_ROUTER_MODE === 'history'
    ? createWebHistory
    : createWebHashHistory;

  const Router = createRouter({
    scrollBehavior: () => ({ left: 0, top: 0 }),
    routes,
    // Leave this as is and make changes in quasar.conf.js instead!
    // quasar.conf.js -> build -> vueRouterMode
    // quasar.conf.js -> build -> publicPath
    history: createHistory(import.meta.env.QUASAR_VUE_ROUTER_BASE),
  });

  Router.beforeEach(async (to, from, next) => {
    // 初始化页面不检查
    if (to.path === '/init') {
      next();
      return;
    }

    if (to.path === '/login') {
      // 登录态按窗口隔离，进入登录页只清理本窗口副本
      clearAuthSession();
      next();
      return;
    }

    // 登录态按窗口隔离：新窗口由 URL 桥接继承，未继承到则回登录页
    if (!isLoggedIn()) {
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
      if (!permStore.hasPermission(requiredPerm)) {
        next('/');
        return;
      }
    }

    next();
  });

  return Router;
});
