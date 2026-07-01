import { boot } from 'quasar/wrappers'
import { vPermission } from 'src/directives/permission'

// 注册 v-permission 全局指令
export default boot(({ app }) => {
  app.directive('permission', vPermission)
})
