import { useQuasar, type QNotifyCreateOptions } from 'quasar'

// 统一异步操作结果处理
// 替代 SearchPage/EditVideoTag/TagPop/ListEditDialog 中重复的 commonExec

interface ExecResult {
  Code?: number
  Message?: string
  Data?: unknown
}

interface CommonExecOptions {
  /** 通知位置 */
  position?: QNotifyCreateOptions['position']
  /** 成功时是否通知（默认 false，仅错误通知） */
  notifyOnSuccess?: boolean
  /** 延迟执行（毫秒） */
  delay?: number
  /** 执行前回调 */
  onBefore?: () => void
}

/**
 * 统一的操作结果处理器
 * @example
 * const { exec } = useCommonExec()
 * const data = await exec(() => api.renameFile(params))
 */
export function useCommonExec(options: CommonExecOptions = {}) {
  const $q = useQuasar()
  const { position = 'bottom-left', notifyOnSuccess = false, delay = 0, onBefore } = options

  async function exec<T = unknown>(fn: () => Promise<ExecResult>): Promise<T | undefined> {
    onBefore?.()

    if (delay > 0) {
      await new Promise((resolve) => setTimeout(resolve, delay))
    }

    const { Code, Message, Data } = (await fn()) || {}
    if (Code !== 200 || notifyOnSuccess) {
      $q.notify({ message: Message || (Code === 200 ? '操作成功' : '操作失败'), position })
    }
    return Data as T
  }

  return { exec }
}
