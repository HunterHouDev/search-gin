import { defineStore } from 'pinia';
import { FileModel, FileQuery } from 'components/model/File';
import { SettingInfo } from 'components/model/Setting';
import { defaultVideoOffset, defaultVideoWidth } from 'components/utils';
import recordWrapper from 'components/model/RecordWrapper';

export const useSystemProperty = defineStore({
  id: 'system',
  persist: {
    enabled: true,
    // 自定义持久化参数
    strategies: [
      {
        // 自定义key
        key: 'systemProperty',
        // 自定义存储方式，默认sessionStorage
        storage: localStorage,
        // 指定要持久化的数据，默认所有 state 都会进行缓存，可以通过 paths 指定要持久化的字段，其他的则不会进行持久化。
        // paths: [
        //   'SearchSuggestions',
        // ],
      },
    ],
  },
  state: () => ({
    singleWindow:{
      width:1280,
      height:720,
    },
    showStyle:'lg', // lg md sm
    showImage:'poster', // post cover
    theme:'natural', // post cover
    expireTime:null,
    lastAuthor:'',
    lastAuthores:[],
    searchPageAutoPullData: false,
    pictureInPictureVideoOffset: defaultVideoOffset,
    pictureInPictureVideoOffsetFullBefore: defaultVideoOffset,
    pictureInPictureVideoWidth: defaultVideoWidth,
    pictureInPictureVideoWidthFullBefore: defaultVideoWidth,
    isFullscreen: false,
    isElectron: false,
    addPlayingTagGoNext: true,
    goAuthorNewWidow: false,
    goSearchNewWidow: false,
    submitTagFromData: true,
    submitMutiTag: true,
    fileEditAutoCode: true,
    fileEditAutoJpg: true,
    fileEditAutoNext: true,
    fileEditAutoRefresh: true,
    tagSizeMap: [],
    shutdownLeftSecond: null,
    shutdownTimer: null,
    videoOptions: {
      autoPlay: true,
      volume: 0.6,
      playerMode: 'contain',
      widescreen: true,
      arrowForwardTime: 60,
      brightness: 100,
      rotate: 0,
      scaleX: 0,
    },
    SearchWords: {} as Record<string, number>,
    SearchRecords: [] as Array<FileQuery>,
    playerMemory: recordWrapper,
    playerRunning: false,
    playerReLocation: true,
    PlayingMovie: new FileModel(),
    FileSearchParam: {
      Page: 1,
      PageSize: 10,
      MovieType: '',
      SortField: 'MTime',
      SortType: 'desc',
      Keyword: '',
      OnlyRepeat: false,
      showStyle: 'post',
    } as FileQuery,
    SettingInfo: {
      ControllerHost: ':10081',
    } as SettingInfo,
    SearchSuggestions: [] as Array<string>,
    videoPlayTimes: {} as Record<string, number>,
  }),
  getters: {
    // 全局主题样式
    themeStyle(): Record<string, string> {
      return {
        color: 'var(--q-text-primary)',
        backgroundColor: 'var(--q-bg-card)',
      };
    },
    getSettingInfo(this) {
      return this.SettingInfo;
    },
    getControllerHost(this) {
      return this.SettingInfo?.ControllerHost;
    },
    getSuggestions(this) {
      if (!this.SearchSuggestions || this.SearchSuggestions.length == 0) {
        this.SearchSuggestions = JSON.parse(
          localStorage.getItem('SearchSuggestions') || '[]'
        );
      }
      return this.SearchSuggestions;
    },
    getSearchParam(this) {
      return this.FileSearchParam;
    },
  },
  actions: {
    syncSearchParam(param: FileQuery) {
      const {
        Page,
        PageSize,
        MovieType,
        SortField,
        SortType,
        Keyword,
        showStyle,
      } = param;
      this.FileSearchParam.Page = Page;
      this.FileSearchParam.PageSize = PageSize;
      this.FileSearchParam.MovieType = MovieType;
      this.FileSearchParam.SortField = SortField;
      this.FileSearchParam.SortType = SortType;
      this.FileSearchParam.Keyword = Keyword;
      this.FileSearchParam.showStyle = showStyle;
      if (param.Keyword) {
        this.addSuggestions(param.Keyword);
      }
      if (Page == 1) {
        this.keywordCount(param);
      }
      this.addRecords(param);
    },
    addRecords(param: FileQuery) {
      if (!this.SearchRecords) {
        this.SearchRecords = [];
      }

      const exist = this.SearchRecords.find(
        (x) =>
          x.Keyword == param.Keyword &&
          x.MovieType == param.MovieType &&
          x.Page == param.Page &&
          x.PageSize == param.PageSize &&
          x.SortField == param.SortField &&
          x.SortType == param.SortType
      );
      if (exist) {
        exist.createdAt = new Date();
        return;
      }
      const rec = new FileQuery().fromObject(param);
      rec.createdAt = new Date();
      this.SearchRecords.unshift(rec);
      if (this.SearchRecords.length > 50) {
        for (let i = 0; i < 10; i++) {
          this.SearchRecords.pop();
        }
      }
    },
    keywordCount(param: FileQuery) {
      const { Keyword } = param;
      if (!Keyword || !isNaN(Number(Keyword))) {
        return;
      }
      // 修改代码以解决类型错误，使用合适的索引类型
      if (this.SearchWords[Keyword]) {
        this.SearchWords[Keyword] = this.SearchWords[Keyword] + 1;
      } else {
        this.SearchWords[Keyword] = 1;
      }
    },
    setSettingInfo(settingInfo: SettingInfo) {
      this.SettingInfo = settingInfo;
      // 同步图片/文件流基础 URL（FileHost 默认值由后端 init() 保证）
      const port = settingInfo.FileHost?.split(':').pop();
      import('components/utils/images').then(({ setFileBaseUrl }) => {
        setFileBaseUrl(`${window.location.protocol}//${window.location.hostname}:${port}`);
      });
    },

    setControllerHost(url: string) {
      this.SettingInfo.ControllerHost = url;
    },

    setPage(page: number) {
      this.FileSearchParam.Page = page;
    },
    setPageSize(pageSize: number) {
      this.FileSearchParam.PageSize = pageSize;
    },
    setMovieType(MovieType: string) {
      this.FileSearchParam.MovieType = MovieType;
    },
    setKeyword(Keyword: string) {
      this.FileSearchParam.Keyword = Keyword;
    },
    setSortField(SortField: string) {
      this.FileSearchParam.SortField = SortField;
    },
    setSortType(SortType: string) {
      this.FileSearchParam.SortType = SortType;
    },
    setOnlyRepeat(OnlyRepeat: boolean) {
      this.FileSearchParam.OnlyRepeat = OnlyRepeat;
    },
    addPlayerLocation(key: string, value: number) {
      this.playerMemory.add(key, value);
    },
    getPlayerLocation(key: string) {
      return this.playerMemory.get(key);
    },

    addSuggestions(queryParam: string) {
      if (!queryParam) {
        return;
      }
      if (!this.SearchSuggestions) {
        this.SearchSuggestions = [];
      }
      const idx = this.SearchSuggestions.indexOf(queryParam);
      if (idx >= 0) {
        this.SearchSuggestions.splice(idx, 1);
      }
      this.SearchSuggestions.unshift(queryParam);
      if (this.SearchSuggestions.length > 100) {
        this.SearchSuggestions.pop();
      }
      localStorage.setItem(
        'SearchSuggestions',
        JSON.stringify(this.SearchSuggestions)
      );
    },
    savePlayTime(id: string) {
      this.videoPlayTimes[id] = Date.now();
    },
    getPlayTime(id: string) {
      return this.videoPlayTimes[id] || null;
    },
  },
});
