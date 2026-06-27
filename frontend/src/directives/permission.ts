import { type Directive } from 'vue'
import { usePermissionStore } from 'src/stores/permission'

// v-permission="'op:edit'" — 无权限时隐藏元素
// v-permission.not ="'op:edit'" — 有权限时隐藏元素
export const vPermission: Directive<HTMLElement, string | string[]> = {
  mounted(el, binding) {
    const store = usePermissionStore()
    const perms = Array.isArray(binding.value) ? binding.value : [binding.value]
    const has = store.hasAnyPermission(perms)
    const modifierNot = binding.modifiers.not
    if ((!modifierNot && !has) || (modifierNot && has)) {
      el.style.display = 'none'
    }
  },
}

// v-permission-btn="'op:edit'" — 无权限时禁用按钮
export const vPermissionBtn: Directive<HTMLElement, string | string[]> = {
  mounted(el, binding) {
    const store = usePermissionStore()
    const perms = Array.isArray(binding.value) ? binding.value : [binding.value]
    const has = store.hasAnyPermission(perms)
    if (!has) {
      el.setAttribute('disabled', 'disabled')
      el.classList.add('disabled')
    }
  },
}
