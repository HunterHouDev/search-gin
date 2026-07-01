import { type Directive, watch } from 'vue'
import { usePermissionStore } from 'src/stores/permission'

// v-permission="'op:edit'" — 无权限时隐藏元素
// v-permission.not ="'op:edit'" — 有权限时隐藏元素
export const vPermission: Directive<HTMLElement, string | string[]> = {
  mounted(el, binding) {
    const store = usePermissionStore()
    store.loadFromSession()

    const update = () => {
      const perms = Array.isArray(binding.value) ? binding.value : [binding.value]
      const has = store.hasAnyPermission(perms)
      const modifierNot = binding.modifiers.not
      el.style.display = ((!modifierNot && !has) || (modifierNot && has)) ? 'none' : ''
    }

    update()
    const unwatch = watch(() => [store.role, store.permissions], update)
    // 组件卸载时清理 watcher
    const observer = new MutationObserver(() => {
      if (!document.contains(el)) {
        observer.disconnect()
        unwatch()
      }
    })
    observer.observe(document.body, { childList: true, subtree: true })
  },
}
