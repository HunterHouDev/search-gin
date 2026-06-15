import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref } from 'vue'
import axios from 'axios'

// Mock axios
vi.mock('axios')

describe('useTorrentDownload', () => {
  // 模拟 $q notify
  const mockNotify = vi.fn()
  const mockOnVideoReady = vi.fn()
  const $q = { notify: mockNotify } as any

  beforeEach(() => {
    vi.clearAllMocks()
    vi.resetAllMocks()
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

      ;(axios.post as any).mockResolvedValue({
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

      ;(axios.post as any).mockRejectedValue({
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

      const file = { path: '/test.mp4', name: 'test.mp4', size: 1024 }
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
      torrentFiles.value = [{ path: '/file', name: 'file', size: 100 }]

      ;(axios.delete as any).mockResolvedValue({})

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
    it('should remove task from activeDownloads', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { removeDownloadTask, activeDownloads } = useTorrentDownload($q, mockOnVideoReady)

      activeDownloads.value = [
        { infoHash: 'hash1', name: 't1', fileName: 'f1', filePath: '/p1', progress: 100, state: 'done', peers: 0 },
        { infoHash: 'hash2', name: 't2', fileName: 'f2', filePath: '/p2', progress: 50, state: 'downloading', peers: 5 },
      ]

      ;(axios.delete as any).mockResolvedValue({})

      const taskToRemove = activeDownloads.value[0]
      removeDownloadTask(taskToRemove)

      expect(activeDownloads.value.length).toBe(1)
      expect(activeDownloads.value[0].infoHash).toBe('hash2')
    })
  })

  describe('cleanup', () => {
    it('should delete torrent and clear state', async () => {
      const { useTorrentDownload } = await import('./useTorrentDownload')
      const { cleanup, currentInfoHash } = useTorrentDownload($q, mockOnVideoReady)

      currentInfoHash.value = 'abc123'
      ;(axios.delete as any).mockResolvedValue({})

      cleanup()

      expect(axios.delete).toHaveBeenCalledWith('/api/torrent/abc123')
      expect(currentInfoHash.value).toBe('')
    })
  })
})
