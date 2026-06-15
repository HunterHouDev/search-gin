import { defineStore } from 'pinia'
import { FileModel } from 'components/model/File'
import recordWrapper from 'components/model/RecordWrapper'
import { defaultVideoOffset, defaultVideoWidth } from 'components/utils'

// 播放器相关状态
// 原 System.ts 中 videoOptions / playerMemory / PlayingMovie 相关

export const usePlayerStore = defineStore({
  id: 'player',
  persist: {
    enabled: true,
    strategies: [{ key: 'playerProperty', storage: localStorage }],
  },
  state: () => ({
    videoOptions: {
      autoPlay: true,
      volume: 0.6,
      playerMode: 'contain' as 'contain' | 'cover' | 'fill',
      widescreen: true,
      arrowForwardTime: 60,
      brightness: 100,
      rotate: 0,
      scaleX: 0,
    },
    playerMemory: recordWrapper,
    playerRunning: false,
    playerReLocation: true,
    PlayingMovie: new FileModel(),
    videoPlayTimes: {} as Record<string, number>,
    pictureInPictureVideoOffset: defaultVideoOffset,
    pictureInPictureVideoOffsetFullBefore: defaultVideoOffset,
    pictureInPictureVideoWidth: defaultVideoWidth,
    pictureInPictureVideoWidthFullBefore: defaultVideoWidth,
  }),
  actions: {
    addPlayerLocation(key: string, value: number) {
      this.playerMemory.add(key, value)
    },
    getPlayerLocation(key: string) {
      return this.playerMemory.get(key)
    },
    savePlayTime(id: string) {
      this.videoPlayTimes[id] = Date.now()
    },
    getPlayTime(id: string) {
      return this.videoPlayTimes[id] || null
    },
    setVolume(v: number) { this.videoOptions.volume = v },
    setPlayerMode(mode: 'contain' | 'cover' | 'fill') { this.videoOptions.playerMode = mode },
  },
})
