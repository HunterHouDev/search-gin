// 应用内链接的登录态继承 —— 让菜单链接"在新窗口中打开"也能免登录。
//
// 背景：登录态按窗口隔离（sessionStorage），新窗口只能靠 URL 快照继承（buildSessionUrl）。
// 浏览器原生的"在新窗口中打开链接"、Ctrl+点击、鼠标中键这类手势无法被 JS 拦截
// （拦截会破坏浏览器行为），因此只能把快照挂在 href 上；同窗口左键仍走 SPA 路由，
// 避免凭据留在地址栏与历史记录里。
//
// 快照在指针进入时重算：窗口活跃会续期登录态（refreshAuthActivity），而右键菜单取的是
// 按下瞬间的 href，重算才能保证带出去的是最新快照，而不是初始化时的那一份。

import { ref } from 'vue';
import { useRouter } from 'vue-router';
import type { Ref } from 'vue';
import { buildSessionUrl } from 'src/utils/authStorage';

export interface SessionLink {
  /** 挂到 <a href> 上：携带一次性登录态快照，供浏览器原生开新窗口使用 */
  href: Ref<string>;
  /** 指针进入时刷新快照 */
  refresh: () => void;
  /** 同窗口左键走 SPA 路由；修饰键 / 中键保留浏览器原生开新窗口行为 */
  navigate: (event: MouseEvent) => void;
}

/**
 * 生成带登录态快照的应用内链接地址。
 * @param target 应用内路由（如 '/search'、'/'），静态菜单项传入即可
 */
export function useSessionLink(target: string): SessionLink {
  const router = useRouter();

  // 干净地址：不带凭据，SPA 导航与快照都以它为基准
  const resolveCleanHref = () => router.resolve(target).href;

  const href = ref(buildSessionUrl(resolveCleanHref()));

  const refresh = () => {
    href.value = buildSessionUrl(resolveCleanHref());
  };

  const navigate = (event: MouseEvent) => {
    // 修饰键 / 中键：不拦截，让浏览器带着 href 里的快照开新窗口或新标签页
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
      return;
    }
    event.preventDefault();
    void router.push(target);
  };

  return { href, refresh, navigate };
}
