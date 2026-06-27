import { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('layouts/MainLayout.vue'),
    children: [
      { path: '/', component: () => import('pages/file/SearchPage.vue') },
      { path: '/search', component: () => import('pages/file/SearchPage.vue') },
      {
        path: '/picture',
        component: () => import('pages/picture/PicturePage.vue'),
      },
      {
        path: '/data',
        component: () => import('pages/IndexPage.vue'),
      },
      {
        path: '/setting',
        component: () => import('pages/setting/SettingPage.vue'),
      },

      {
        path: '/system',
        component: () => import('pages/system/SystemPage.vue'),
      },
    ],
  },
  {
    path: '/immersive',
    component: () => import('pages/immersive/ImmersivePlayer.vue'),
  },
  {
    path: '/playing/:id',
    component: () => import('src/pages/playing/PlayingPage.vue'),
  },
  // Always leave this as last one,
  // but you can also remove it
  {
    path: '/:catchAll(.*)*',
    component: () => import('pages/ErrorNotFound.vue'),
  },
  {
    path: '/login',
    component: () import('pages/LoginPage.vue'),
  },
  {
    path: '/init',
    component: () => import('pages/InitPage.vue'),
  }
];

export default routes;
