import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import type { QVueGlobals } from 'quasar'

// Mock src/boot/axios — useTorrentDownload imports { api } from this module
const mockApi = {
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}

vi.mock('src/boot/axios', () => ({
  api: mockApi,
}))

describe('useTorrentDownload', () => {
  // 模拟 $q notify
  const mockNotify = vi.fn()
  const mockOnVideoReady = vi.fn()
  const $q = { notify: mockNotify } as unknown as QVueGlobals

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('submitMagnet', () => {
    it('should reject invalid magnet URI', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { magnetURI, submitMagnet } = useTorrentDownload($q, mockOnVideoReady)

      magnetURI.value = 'not-a-magnet'
      await submitMagnet()

      expect(mockNotify).toHaveBeenCalledWith({
        type: 'negative',
        message: '请输入有效的磁力链',
        position: 'top',
      })
    })

    it('should handle successful magnet submission', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { magnetURI, submitMagnet, torrentLoading } = useTorrentDownload($q, mockOnVideoReady)

      const mockData = {
        infoHash: 'abc123',
        name: 'test-torrent',
        files: [{ path: '/file1.mp4', name: 'file1.mp4', size: 1024 }],
      }

      mockApi.post.mockResolvedValue({
        data: { code: 200, data: mockData },
      })

      magnetURI.value = 'magnet:?xt=urn:btih:abc123'
      await submitMagnet()

      expect(torrentLoading.value).toBe(false)
      expect(mockNotify).not.toHaveBeenCalled()
    })

    it('should handle API error', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { magnetURI, submitMagnet } = useTorrentDownload($q, mockOnVideoReady)

      mockApi.post.mockRejectedValue({
        response: { data: { message: 'Server error' } },
      })

      magnetURI.value = 'magnet:?xt=urn:btih:abc123'
      await submitMagnet()

      expect(mockNotify).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'negative' }),
      )
    })
  })

  describe('selectTorrentFile', () => {
    it('should update selectedTorrentFile', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { selectTorrentFile, selectedTorrentFile } = useTorrentDownload($q, mockOnVideoReady)

      const file = { path: '/test.mp4', name: 'test.mp4', length: 1024 }
      selectTorrentFile(file)

      expect(selectedTorrentFile.value).toBe('/test.mp4')
    })
  })

  describe('cancelTorrent', () => {
    it('should clear all torrent state', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const {
        cancelTorrent,
        currentInfoHash,
        torrentLoading,
        torrentProgress,
        torrentState,
        torrentName,
        torrentFiles,
      } = useTorrentDownload($q, mockOnVideoReady)

      currentInfoHash.value = 'abc123'
      torrentLoading.value = true
      torrentProgress.value = 50
      torrentState.value = 'downloading'
      torrentName.value = 'test'
      torrentFiles.value = [{ path: '/file', name: 'file', length: 100 }]

      mockApi.delete.mockResolvedValue({})
      mockApi.get.mockResolvedValue({ data: { Code: 200, Data: { tasks: [] } } })

      await cancelTorrent()

      expect(currentInfoHash.value).toBe('')
      expect(torrentLoading.value).toBe(false)
      expect(torrentProgress.value).toBe(0)
      expect(torrentState.value).toBe('')
      expect(torrentName.value).toBe('')
      expect(torrentFiles.value).toEqual([])
    })
  })

  describe('playDownloadTask', () => {
    it('should call onVideoReady with correct URL', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { playDownloadTask, currentInfoHash } = useTorrentDownload($q, mockOnVideoReady)

      const task = {
        infoHash: 'hash123',
        name: 'Test',
        fileName: 'video.mp4',
        filePath: '/path/video.mp4',
        progress: 100,
        state: '已下载',
        peers: 0,
      }

      playDownloadTask(task)

      expect(currentInfoHash.value).toBe('hash123')
      expect(mockOnVideoReady).toHaveBeenCalledWith(
        '/api/torrent/stream/hash123?file=%2Fpath%2Fvideo.mp4',
        'video.mp4',
      )
    })
  })

  describe('removeDownloadTask', () => {
    it('should remove task from activeDownloads and refresh from server', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { removeDownloadTask, activeDownloads } = useTorrentDownload($q, mockOnVideoReady)

      activeDownloads.value = [
        { infoHash: 'hash1', name: 't1', fileName: 'f1', filePath: '/p1', progress: 100, state: 'done', peers: 0 },
        { infoHash: 'hash2', name: 't2', fileName: 'f2', filePath: '/p2', progress: 50, state: 'downloading', peers: 5 },
      ]

      mockApi.delete.mockResolvedValue({})
      // 服务端删除 hash1 后，任务列表只剩 hash2（服务端已连带清理同种子任务）
      mockApi.get.mockResolvedValue({
        data: {
          Code: 200,
          Data: {
            tasks: [
              {
                ID: 'task2',
                Name: 'f2',
                Type: '磁力下载',
                Status: '执行中',
                Progress: 50,
                InfoHash: 'hash2',
                TorrentName: 't2',
                TorrentFile: '/p2',
                Path: '/dest/p2',
                CreateTime: '2026-09-15T00:00:00Z',
              },
            ],
          },
        },
      })

      const taskToRemove = activeDownloads.value[0]
      await removeDownloadTask(taskToRemove)

      expect(mockApi.delete).toHaveBeenCalledWith('/api/torrent/hash1')
      expect(activeDownloads.value.length).toBe(1)
      expect(activeDownloads.value[0].infoHash).toBe('hash2')
    })
  })

  describe('cleanup', () => {
    it('should stop polling without cancelling server-side downloads', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { cleanup, currentInfoHash } = useTorrentDownload($q, mockOnVideoReady)

      currentInfoHash.value = 'abc123'
      mockApi.delete.mockResolvedValue({})

      cleanup()

      // 任务已落在服务端统一任务列表，离开页面不取消下载
      expect(mockApi.delete).not.toHaveBeenCalled()
    })
  })

  describe('refreshDownloadTasks', () => {
    it('should map server torrent tasks to download tasks', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { activeDownloads, refreshDownloadTasks } = useTorrentDownload($q, mockOnVideoReady)

      mockApi.get.mockResolvedValue({
        data: {
          Code: 200,
          Data: {
            tasks: [
              {
                ID: 'task1',
                Name: '转码任务',
                Type: '转码',
                Status: '执行中',
                Progress: 0,
                InfoHash: '',
                TorrentName: '',
                TorrentFile: '',
                Path: '',
                CreateTime: '2026-09-15T00:00:00Z',
              },
              {
                ID: 'task2',
                Name: 'video.mp4',
                Type: '磁力下载',
                Status: '执行中',
                Progress: 42,
                InfoHash: 'hash9',
                TorrentName: 'my-torrent',
                TorrentFile: 'dir/video.mp4',
                Path: '/dest/dir/video.mp4',
                CreateTime: '2026-09-15T00:00:00Z',
              },
            ],
          },
        },
      })

      await refreshDownloadTasks()

      // 只保留磁力下载任务，且字段正确映射
      expect(activeDownloads.value.length).toBe(1)
      expect(activeDownloads.value[0]).toEqual({
        infoHash: 'hash9',
        name: 'my-torrent',
        fileName: 'video.mp4',
        filePath: 'dir/video.mp4',
        progress: 42,
        state: '执行中',
        peers: 0,
      })
    })
  })
})
