import { commonAxios } from 'src/boot/axios'

/** Torrent / 磁力链接 API — 原在 ImmersivePlayer.vue 中直接调 axios */

export interface TorrentAddParams {
  magnetURI: string
}

export interface TorrentDownloadParams {
  infoHash: string
  fileIndex?: number
}

export interface TorrentFile {
  name: string
  size: number
}

export interface TorrentStatus {
  infoHash: string
  name: string
  progress: number
  downloadSpeed: number
  state: string
  files: TorrentFile[]
}

export async function addTorrent(params: TorrentAddParams) {
  const { data } = await commonAxios().post('/api/torrent/add', params)
  return data
}

export async function startTorrentDownload(params: TorrentDownloadParams) {
  const { data } = await commonAxios().post('/api/torrent/startDownload', params)
  return data
}

export async function getTorrentStatus(infoHash: string): Promise<TorrentStatus> {
  const { data } = await commonAxios().get(`/api/torrent/status/${infoHash}`)
  return data
}
