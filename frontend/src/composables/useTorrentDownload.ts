import { ref } from 'vue'
import axios from 'axios'
import type { QVueGlobals } from 'quasar'

// 磁力链 / BT 下载逻辑
// 提取自 ImmersivePlayer.vue，减少组件代码约 220 行

export interface TorrentFile {
  path: string
  name: string
  size: number
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

  // 提交磁力链
  async function submitMagnet() {
    const uri = magnetURI.value.trim()
    if (!uri.startsWith('magnet:')) {
      $q.notify({ type: 'negative', message: '请输入有效的磁力链', position: 'top' })
      return
    }
    torrentLoading.value = true
    torrentProgress.value = 0
    torrentState.value = '正在解析磁力链...'
    torrentName.value = '获取种子信息中...'
    try {
      const res = await axios.post('/api/torrent/add', { magnetURI: uri })
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
    } catch (err: any) {
      $q.notify({
        type: 'negative',
        message: err.response?.data?.message ?? err.response?.data?.Message ?? '请求失败: ' + err.message,
        position: 'top',
      })
      torrentLoading.value = false
    }
  }

  function selectTorrentFile(file: TorrentFile) {
    selectedTorrentFile.value = file.path
  }

  async function playSelectedTorrentFile() {
    if (!selectedTorrentFile.value || !currentInfoHash.value) return
    torrentLoading.value = true
    showTorrentFiles.value = false
    torrentState.value = '正在开始下载...'
    torrentProgress.value = 0
    const fileName = torrentFiles.value.find((f) => f.path === selectedTorrentFile.value)?.name || '未知文件'
    try {
      const response = await axios.post('/api/torrent/startDownload', {
        infoHash: currentInfoHash.value,
        filePath: selectedTorrentFile.value,
      })
      const result = response.data?.data ?? response.data?.Data
      const newTask: DownloadTask = {
        infoHash: currentInfoHash.value,
        name: torrentName.value,
        fileName,
        filePath: selectedTorrentFile.value,
        progress: result?.skipped ? 100 : 0,
        state: result?.skipped ? '已下载' : '准备下载',
        peers: 0,
      }
      activeDownloads.value.push(newTask)
      if (!result?.skipped) startPolling(currentInfoHash.value, newTask, selectedTorrentFile.value)
      const streamUrl = `/api/torrent/stream/${currentInfoHash.value}?file=${encodeURIComponent(selectedTorrentFile.value)}`
      onVideoReady(streamUrl, fileName)
      if (result?.skipped) {
        $q.notify({ type: 'positive', message: '文件已存在，无需下载', position: 'top', timeout: 2000 })
      }
    } catch (err: any) {
      $q.notify({
        type: 'negative',
        message: '启动下载失败: ' + ((err.response?.data?.message ?? err.response?.data?.Message) || err.message),
        position: 'top',
      })
    }
    torrentLoading.value = false
    selectedTorrentFile.value = null
  }

  function startPolling(infoHash: string, task: DownloadTask, filePath: string) {
    stopPolling()
    const pollStart = Date.now()
    torrentPollTimer = setInterval(async () => {
      if (Date.now() - pollStart > 5 * 60 * 1000) {
        stopPolling()
        $q.notify({ type: 'warning', message: '下载超时', position: 'top' })
        return
      }
      try {
        const res = await axios.get(`/api/torrent/status/${infoHash}`)
        const d = res.data?.data ?? res.data?.Data
        if (res.data?.code === 200 && d) {
          torrentName.value = d.name
          torrentProgress.value = d.progress
          torrentState.value = d.state
          torrentPeers.value = d.peers
          if (task) { task.progress = d.progress; task.state = d.state; task.peers = d.peers }
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
    stopPolling()
    if (currentInfoHash.value) {
      try { await axios.delete(`/api/torrent/${currentInfoHash.value}`) } catch { /* ignore */ }
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
    axios.delete(`/api/torrent/${task.infoHash}`).catch(() => {})
    activeDownloads.value = activeDownloads.value.filter((t) => t.infoHash !== task.infoHash)
    if (currentInfoHash.value === task.infoHash) {
      currentInfoHash.value = ''
      torrentLoading.value = false
    }
  }

  function cleanup() {
    stopPolling()
    if (currentInfoHash.value) {
      axios.delete(`/api/torrent/${currentInfoHash.value}`).catch(() => {})
    }
  }

  return {
    // state
    magnetURI, magnetFocused, torrentLoading, torrentName, torrentProgress,
    torrentState, torrentPeers, currentInfoHash, torrentFiles, showTorrentFiles,
    selectedTorrentFile, showDownloadManager, activeDownloads,
    // actions
    submitMagnet, selectTorrentFile, playSelectedTorrentFile,
    cancelTorrent, playDownloadTask, openDownloadFolder, removeDownloadTask,
    cleanup,
  }
}
