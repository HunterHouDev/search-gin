import { ref } from 'vue'
import { api } from 'src/boot/axios'
import type { QVueGlobals } from 'quasar'
import type { TorrentTransferTask } from 'src/types'

// 磁力链 / BT 下载逻辑
// 提取自 ImmersivePlayer.vue，减少组件代码约 220 行
//
// 下载任务列表已合并到后端统一任务列表（/api/transferTasks，Type=磁力下载）：
// 关闭播放页 / 刷新页面不影响下载，进度由服务端每 5 秒同步回任务列表。
// 完成后不触发索引扫描（有意设计：磁力下载文件不自动入库）。

export interface TorrentFile {
  path: string
  name: string
  /** 后端字段名为 length（字节） */
  length: number
}

export interface DownloadTask {
  infoHash: string
  name: string
  fileName: string
  filePath: string
  progress: number
  state: string
  peers: number
}

/** 统一任务列表中磁力下载任务的类型标识（后端 TaskTypeTorrent） */
const TORRENT_TASK_TYPE = '磁力下载'
/** 下载管理器任务列表轮询间隔（毫秒） */
const TASK_LIST_POLL_MS = 3000

/** 把统一任务列表中的磁力下载任务映射为下载管理器条目 */
function toDownloadTask(t: TorrentTransferTask): DownloadTask {
  const filePath = t.TorrentFile || ''
  const fileName = filePath ? (filePath.split('/').pop() || filePath) : ''
  return {
    infoHash: t.InfoHash,
    name: t.TorrentName || t.Name,
    fileName,
    filePath,
    progress: t.Progress ?? 0,
    state: t.Status,
    peers: 0,
  }
}

export function useTorrentDownload(
  $q: QVueGlobals,
  onVideoReady: (src: string, name: string) => void,
) {
  // 状态
  const magnetURI = ref('')
  const magnetFocused = ref(false)
  const torrentLoading = ref(false)
  const torrentName = ref('')
  const torrentProgress = ref(0)
  const torrentState = ref('')
  const torrentPeers = ref(0)
  const currentInfoHash = ref('')
  const torrentFiles = ref<TorrentFile[]>([])
  const showTorrentFiles = ref(false)
  const selectedTorrentFile = ref<string | null>(null)
  const showDownloadManager = ref(false)
  const activeDownloads = ref<DownloadTask[]>([])
  let torrentPollTimer: ReturnType<typeof setInterval> | null = null
  let taskListTimer: ReturnType<typeof setInterval> | null = null

  // ── 统一任务列表同步（服务端驱动） ────────────────────────────────────────────
  async function refreshDownloadTasks() {
    try {
      const res = await api.get('/api/transferTasks')
      const body = res.data
      const tasks: TorrentTransferTask[] = body?.data?.tasks ?? body?.Data?.tasks ?? []
      activeDownloads.value = tasks
        .filter((t) => t.Type === TORRENT_TASK_TYPE)
        .map((t) => toDownloadTask(t))
      // 无进行中的任务时停止轮询（下载完成即定格最终状态）
      if (!hasRunningTask()) stopTaskListPolling()
    } catch {
      /* 任务列表刷新失败不影响播放 */
    }
  }

  /** 是否仍有等待/执行中的磁力下载任务 */
  function hasRunningTask() {
    return activeDownloads.value.some((t) => t.state === '执行中' || t.state === '等待')
  }

  function startTaskListPolling() {
    if (taskListTimer) return
    taskListTimer = setInterval(refreshDownloadTasks, TASK_LIST_POLL_MS)
  }

  function stopTaskListPolling() {
    if (taskListTimer) {
      clearInterval(taskListTimer)
      taskListTimer = null
    }
  }

  // 进入页面时恢复服务端任务列表（下载不随页面关闭而中断）
  refreshDownloadTasks()

  // 解析请求序号：取消或重新提交后，旧请求的响应直接丢弃，
  // 避免迟到的响应把弹窗意外打开
  let magnetSeq = 0

  // 提交磁力链
  async function submitMagnet() {
    const uri = magnetURI.value.trim()
    if (!uri.startsWith('magnet:')) {
      $q.notify({ type: 'negative', message: '请输入有效的磁力链', position: 'top' })
      return
    }
    const seq = ++magnetSeq
    torrentLoading.value = true
    torrentProgress.value = 0
    torrentState.value = '正在解析磁力链，等待 DHT 网络返回种子信息（最长 60 秒）'
    torrentName.value = '获取种子信息中...'
    try {
      // 后端要等 DHT 网络返回 metadata，最长 60s；axios 默认 30s 会先断开，
      // 表现为「解析结果出不来」，这里单独放宽到 90s
      const res = await api.post('/api/torrent/add', { magnetURI: uri }, { timeout: 90000 })
      if (seq !== magnetSeq) return
      const code = res.data?.code ?? res.data?.Code
      const data = res.data?.data ?? res.data?.Data
      if (code === 200 && data) {
        currentInfoHash.value = data.infoHash
        torrentName.value = data.name || '未知种子'
        torrentFiles.value = data.files || []
        if (torrentFiles.value.length > 0) {
          showTorrentFiles.value = true
          torrentLoading.value = false
        } else {
          $q.notify({ type: 'warning', message: '未解析到文件', position: 'top' })
          torrentLoading.value = false
        }
      } else {
        $q.notify({
          type: 'negative',
          message: res.data?.message ?? res.data?.Message ?? '添加磁力链失败',
          position: 'top',
        })
        torrentLoading.value = false
      }
    } catch (err: unknown) {
      if (seq !== magnetSeq) return
      const axiosErr = err as { response?: { data?: { message?: string; Message?: string } }; message?: string }
      $q.notify({
        type: 'negative',
        message: axiosErr.response?.data?.message ?? axiosErr.response?.data?.Message ?? '请求失败: ' + axiosErr.message,
        position: 'top',
      })
      torrentLoading.value = false
    }
  }

  function selectTorrentFile(file: TorrentFile) {
    selectedTorrentFile.value = file.path
  }

  /**
   * 启动选中文件的下载。
   * play=true：加入下载列表并立即起播（边下边播）；play=false：只下载，不动播放器。
   */
  async function startSelectedTorrentFile(play: boolean) {
    const infoHash = currentInfoHash.value
    const filePath = selectedTorrentFile.value
    if (!filePath || !infoHash) return
    const fileName = torrentFiles.value.find((f) => f.path === filePath)?.name || '未知文件'

    showTorrentFiles.value = false
    if (play) {
      torrentLoading.value = true
      torrentState.value = '正在开始下载...'
      torrentProgress.value = 0
    }

    try {
      const response = await api.post('/api/torrent/startDownload', {
        infoHash,
        filePath,
      })
      const result = response.data?.data ?? response.data?.Data
      // 任务列表由服务端统一维护，这里刷新一次并启动轮询跟踪进度
      await refreshDownloadTasks()
      if (!result?.skipped) startTaskListPolling()
      // 边下边播：继续轮询单种子进度（更新缓冲状态显示 + 3% 缓冲就绪后起播）
      if (play && !result?.skipped) startPolling(infoHash, filePath)

      if (play) {
        // 立即用流地址起播，后台继续下载
        const streamUrl = `/api/torrent/stream/${infoHash}?file=${encodeURIComponent(filePath)}`
        onVideoReady(streamUrl, fileName)
        if (result?.skipped) {
          $q.notify({ type: 'positive', message: '文件已存在，无需下载', position: 'top', timeout: 2000 })
        }
      } else {
        $q.notify({
          type: 'positive',
          message: result?.skipped ? '文件已存在，无需下载' : `已开始下载：${fileName}`,
          position: 'top',
          timeout: 2000,
        })
      }
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { message?: string; Message?: string } }; message?: string }
      $q.notify({
        type: 'negative',
        message: '启动下载失败: ' + ((axiosErr.response?.data?.message ?? axiosErr.response?.data?.Message) || axiosErr.message),
        position: 'top',
      })
      // 请求失败时还原选择弹窗，避免用户以为已经在下
      showTorrentFiles.value = true
    }
    torrentLoading.value = false
    selectedTorrentFile.value = null
  }

  function playSelectedTorrentFile() {
    return startSelectedTorrentFile(true)
  }

  function downloadSelectedTorrentFile() {
    return startSelectedTorrentFile(false)
  }

  /** 边下边播：轮询单种子进度，缓冲就绪（≥3%）后起播。任务进度已由统一任务列表轮询负责 */
  function startPolling(infoHash: string, filePath: string) {
    stopPolling()
    const pollStart = Date.now()
    torrentPollTimer = setInterval(async () => {
      if (Date.now() - pollStart > 5 * 60 * 1000) {
        stopPolling()
        $q.notify({ type: 'warning', message: '缓冲超时，请稍后从下载管理器手动播放', position: 'top' })
        return
      }
      try {
        const res = await api.get(`/api/torrent/status/${infoHash}`)
        const d = res.data?.data ?? res.data?.Data
        if (res.data?.code === 200 && d) {
          torrentName.value = d.name
          torrentProgress.value = d.progress
          torrentState.value = d.state
          torrentPeers.value = d.peers
          if (d.progress >= 3) {
            torrentState.value = '缓冲就绪，开始播放'
            const streamUrl = `/api/torrent/stream/${infoHash}?file=${encodeURIComponent(filePath)}`
            onVideoReady(streamUrl, d.videoFile || d.name)
            stopPolling()
          }
        }
      } catch { /* poll errors are non-critical */ }
    }, 2000)
  }

  function stopPolling() {
    if (torrentPollTimer) { clearInterval(torrentPollTimer); torrentPollTimer = null }
  }

  async function cancelTorrent() {
    // 使在途的解析请求作废，避免响应回来后又被弹出文件选择框
    magnetSeq++
    stopPolling()
    if (currentInfoHash.value) {
      const infoHash = currentInfoHash.value
      try { await api.delete(`/api/torrent/${infoHash}`) } catch { /* ignore */ }
      activeDownloads.value = activeDownloads.value.filter((t) => t.infoHash !== infoHash)
      // 服务端会连带清理该种子的任务条目，同步刷新一次
      refreshDownloadTasks()
    }
    torrentLoading.value = false
    torrentProgress.value = 0
    torrentState.value = ''
    torrentName.value = ''
    currentInfoHash.value = ''
    torrentFiles.value = []
    showTorrentFiles.value = false
    selectedTorrentFile.value = null
  }

  function playDownloadTask(task: DownloadTask) {
    const streamUrl = `/api/torrent/stream/${task.infoHash}?file=${encodeURIComponent(task.filePath)}`
    currentInfoHash.value = task.infoHash
    onVideoReady(streamUrl, task.fileName)
  }

  function openDownloadFolder(task: DownloadTask) {
    api
      .post('/api/torrent/openFolder', {
        infoHash: task.infoHash,
        filePath: task.filePath,
      })
      .catch((err: unknown) => {
        const axiosErr = err as {
          response?: { data?: { message?: string; Message?: string } }
          message?: string
        }
        $q.notify({
          type: 'negative',
          message: '打开下载目录失败: ' + ((axiosErr.response?.data?.message ?? axiosErr.response?.data?.Message) || axiosErr.message),
          position: 'top',
        })
      })
  }

  /** 删除任务 = 取消下载（服务端会停止种子并清理同种子全部任务条目） */
  async function removeDownloadTask(task: DownloadTask) {
    try { await api.delete(`/api/torrent/${task.infoHash}`) } catch { /* ignore */ }
    activeDownloads.value = activeDownloads.value.filter((t) => t.infoHash !== task.infoHash)
    if (currentInfoHash.value === task.infoHash) {
      currentInfoHash.value = ''
      torrentLoading.value = false
    }
    await refreshDownloadTasks()
  }

  function cleanup() {
    stopPolling()
    stopTaskListPolling()
    // 任务已落在服务端统一任务列表中，离开播放页不取消下载
  }

  return {
    // state
    magnetURI, magnetFocused, torrentLoading, torrentName, torrentProgress,
    torrentState, torrentPeers, currentInfoHash, torrentFiles, showTorrentFiles,
    selectedTorrentFile, showDownloadManager, activeDownloads,
    // actions
    submitMagnet, selectTorrentFile, playSelectedTorrentFile, downloadSelectedTorrentFile,
    cancelTorrent, playDownloadTask, openDownloadFolder, removeDownloadTask,
    refreshDownloadTasks,
    cleanup,
  }
}
