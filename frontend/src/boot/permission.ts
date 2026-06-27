import { boot } from 'quasar/wrappers'
import { vPermission, vPermissionBtn } from 'src/directives/permission'

// 注册 v-permission 和 v-permission-btn 全局指令
export default boot(({ app }) => {
  app.directive('permission', vPermission)
  app.directive('permission-btn', vPermissionBtn)
})
