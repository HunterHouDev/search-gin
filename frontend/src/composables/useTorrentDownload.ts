import { ref } from 'vue'
import { api } from 'src/boot/axios'
import type { QVueGlobals } from 'quasar'

// 磁力链 / BT 下载逻辑
// 提取自 ImmersivePlayer.vue，减少组件代码约 220 行

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
      const newTask: DownloadTask = {
        infoHash,
        name: torrentName.value,
        fileName,
        filePath,
        progress: result?.skipped ? 100 : 0,
        state: result?.skipped ? '已下载' : '准备下载',
        peers: 0,
      }
      activeDownloads.value.push(newTask)
      if (!result?.skipped) startPolling(infoHash, newTask, filePath, play)

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
          message: result?.skipped ? '文件已存在，已加入下载列表' : `已开始下载：${fileName}`,
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

  /** autoPlay=false 时只跟踪进度，不触发播放 */
  function startPolling(infoHash: string, task: DownloadTask, filePath: string, autoPlay = true) {
    stopPolling()
    const pollStart = Date.now()
    torrentPollTimer = setInterval(async () => {
      if (Date.now() - pollStart > 5 * 60 * 1000) {
        stopPolling()
        $q.notify({ type: 'warning', message: '下载超时', position: 'top' })
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
          if (task) { task.progress = d.progress; task.state = d.state; task.peers = d.peers }
          if (autoPlay && d.progress >= 3) {
            torrentState.value = '缓冲就绪，开始播放'
            const streamUrl = `/api/torrent/stream/${infoHash}?file=${encodeURIComponent(filePath)}`
            onVideoReady(streamUrl, d.videoFile || d.name)
            stopPolling()
            return
          }
          if (!autoPlay && d.progress >= 100) {
            task.state = '已下载'
            stopPolling()
            $q.notify({ type: 'positive', message: `下载完成：${task.fileName}`, position: 'top' })
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
      try { await api.delete(`/api/torrent/${currentInfoHash.value}`) } catch { /* ignore */ }
      activeDownloads.value = activeDownloads.value.filter((t) => t.infoHash !== currentInfoHash.value)
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
    window.open(`/api/openFolder/${task.infoHash}`, '_blank')
  }

  function removeDownloadTask(task: DownloadTask) {
    api.delete(`/api/torrent/${task.infoHash}`).catch(() => { /* ignore */ })
    activeDownloads.value = activeDownloads.value.filter((t) => t.infoHash !== task.infoHash)
    if (currentInfoHash.value === task.infoHash) {
      currentInfoHash.value = ''
      torrentLoading.value = false
    }
  }

  function cleanup() {
    stopPolling()
    if (currentInfoHash.value) {
      api.delete(`/api/torrent/${currentInfoHash.value}`).catch(() => { /* ignore */ })
    }
  }

  return {
    // state
    magnetURI, magnetFocused, torrentLoading, torrentName, torrentProgress,
    torrentState, torrentPeers, currentInfoHash, torrentFiles, showTorrentFiles,
    selectedTorrentFile, showDownloadManager, activeDownloads,
    // actions
    submitMagnet, selectTorrentFile, playSelectedTorrentFile, downloadSelectedTorrentFile,
    cancelTorrent, playDownloadTask, openDownloadFolder, removeDownloadTask,
    cleanup,
  }
}
