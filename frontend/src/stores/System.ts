import { ref, reactive, computed } from 'vue';
import { defineStore } from 'pinia';
import { FileModel, FileQuery } from 'components/model/File';
import type { SettingInfo } from 'components/model/Setting';
import { defaultVideoOffset, defaultVideoWidth } from 'components/utils';
import recordWrapper from 'components/model/RecordWrapper';

export const useSystemProperty = defineStore('system', () => {
  // ── state ──
  const singleWindow = ref({ width: 1280, height: 720 });
  const showStyle = ref('lg'); // lg md sm
  const showImage = ref('poster'); // post cover
  const theme = ref('natural');
  const expireTime = ref<number | null>(null);
  const lastAuthor = ref('');
  const lastAuthores = ref([] as string[]);
  const searchPageAutoPullData = ref(false);
  const pictureInPictureVideoOffset = ref(defaultVideoOffset);
  const pictureInPictureVideoOffsetFullBefore = ref(defaultVideoOffset);
  const pictureInPictureVideoWidth = ref(defaultVideoWidth);
  const pictureInPictureVideoWidthFullBefore = ref(defaultVideoWidth);
  const isFullscreen = ref(false);
  const isElectron = ref(false);
  const addPlayingTagGoNext = ref(true);
  const goAuthorNewWidow = ref(false);
  const goSearchNewWidow = ref(false);
  const submitTagFromData = ref(true);
  const submitMutiTag = ref(true);
  const fileEditAutoCode = ref(true);
  const fileEditAutoJpg = ref(true);
  const fileEditAutoNext = ref(true);
  const fileEditAutoRefresh = ref(true);
  const tagSizeMap = ref([] as any[]);
  const shutdownLeftSecond = ref<number | null>(null);
  const shutdownTimer = ref<ReturnType<typeof setInterval> | null>(null);
  const videoOptions = reactive({
    autoPlay: true,
    volume: 0.6,
    playerMode: 'contain',
    widescreen: true,
    arrowForwardTime: 60,
    brightness: 100,
    rotate: 0,
    scaleX: 0,
  });
  const SearchWords = ref({} as Record<string, number>);
  const SearchRecords = ref([] as Array<FileQuery>);
  const playerMemory = ref(recordWrapper);
  const playerRunning = ref(false);
  const playerReLocation = ref(true);
  const PlayingMovie = ref(new FileModel());
  const FileSearchParam = reactive<FileQuery>({
    Page: 1,
    PageSize: 10,
    MovieType: '',
    SortField: 'MTime',
    SortType: 'desc',
    Keyword: '',
    OnlyRepeat: false,
    showStyle: 'post',
  } as FileQuery);
  const SettingInfo = ref({
    ControllerHost: ':10081',
  } as SettingInfo);
  const SearchSuggestions = ref([] as string[]);
  const videoPlayTimes = ref({} as Record<string, number>);

  // ── getters ──
  const themeStyle = computed(() => ({
    color: 'var(--q-text-primary)',
    backgroundColor: 'var(--q-bg-card)',
  }));

  const getSettingInfo = computed(() => SettingInfo.value);
  const getControllerHost = computed(() => SettingInfo.value?.ControllerHost);
  const getSuggestions = computed(() => SearchSuggestions.value);
  const getSearchParam = computed(() => FileSearchParam);

  // ── actions ──
  function syncSearchParam(param: FileQuery) {
    const {
      Page,
      PageSize,
      MovieType,
      SortField,
      SortType,
      Keyword,
      showStyle: ss,
    } = param;
    FileSearchParam.Page = Page;
    FileSearchParam.PageSize = PageSize;
    FileSearchParam.MovieType = MovieType;
    FileSearchParam.SortField = SortField;
    FileSearchParam.SortType = SortType;
    FileSearchParam.Keyword = Keyword;
    FileSearchParam.showStyle = ss;
    if (param.Keyword) {
      addSuggestions(param.Keyword);
    }
    if (Page == 1) {
      keywordCount(param);
    }
    addRecords(param);
  }

  function addRecords(param: FileQuery) {
    if (!SearchRecords.value) {
      SearchRecords.value = [];
    }

    const exist = SearchRecords.value.find(
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
    SearchRecords.value.unshift(rec);
    if (SearchRecords.value.length > 50) {
      for (let i = 0; i < 10; i++) {
        SearchRecords.value.pop();
      }
    }
  }

  function keywordCount(param: FileQuery) {
    const { Keyword } = param;
    if (!Keyword || !isNaN(Number(Keyword))) {
      return;
    }
    if (SearchWords.value[Keyword]) {
      SearchWords.value[Keyword] = SearchWords.value[Keyword] + 1;
    } else {
      SearchWords.value[Keyword] = 1;
    }
    // 限制 SearchWords 大小，保留最近 200 个关键词
    const keys = Object.keys(SearchWords.value);
    if (keys.length > 200) {
      const sorted = keys.sort((a, b) => SearchWords.value[a] - SearchWords.value[b]);
      for (let i = 0; i < 50; i++) {
        delete SearchWords.value[sorted[i]];
      }
    }
  }

  function setSettingInfo(settingInfo: SettingInfo) {
    SettingInfo.value = settingInfo;
    // 同步图片/文件流基础 URL（FileHost 默认值由后端 init() 保证）
    const port = settingInfo.FileHost?.split(':').pop();
    import('components/utils/images').then(({ setFileBaseUrl }) => {
      setFileBaseUrl(`${window.location.protocol}//${window.location.hostname}:${port}`);
    });
  }

  function setControllerHost(url: string) {
    SettingInfo.value.ControllerHost = url;
  }

  function setPage(page: number) {
    FileSearchParam.Page = page;
  }

  function setPageSize(pageSize: number) {
    FileSearchParam.PageSize = pageSize;
  }

  function setMovieType(movieType: string) {
    FileSearchParam.MovieType = movieType;
  }

  function setKeyword(keyword: string) {
    FileSearchParam.Keyword = keyword;
  }

  function setSortField(sortField: string) {
    FileSearchParam.SortField = sortField;
  }

  function setSortType(sortType: string) {
    FileSearchParam.SortType = sortType;
  }

  function setOnlyRepeat(onlyRepeat: boolean) {
    FileSearchParam.OnlyRepeat = onlyRepeat;
  }

  function addPlayerLocation(key: string, value: number) {
    playerMemory.value.add(key, value);
  }

  function getPlayerLocation(key: string) {
    return playerMemory.value.get(key);
  }

  function addSuggestions(queryParam: string) {
    if (!queryParam) {
      return;
    }
    if (!SearchSuggestions.value) {
      SearchSuggestions.value = [];
    }
    const idx = SearchSuggestions.value.indexOf(queryParam);
    if (idx >= 0) {
      SearchSuggestions.value.splice(idx, 1);
    }
    SearchSuggestions.value.unshift(queryParam);
    if (SearchSuggestions.value.length > 100) {
      SearchSuggestions.value.pop();
    }
  }

  function savePlayTime(id: string) {
    videoPlayTimes.value[id] = Date.now();
    // 限制 videoPlayTimes 大小，保留最近 500 条
    const keys = Object.keys(videoPlayTimes.value);
    if (keys.length > 500) {
      const sorted = keys.sort((a, b) => videoPlayTimes.value[a] - videoPlayTimes.value[b]);
      for (let i = 0; i < 100; i++) {
        delete videoPlayTimes.value[sorted[i]];
      }
    }
  }

  function getPlayTime(id: string) {
    return videoPlayTimes.value[id] || null;
  }

  return {
    // state
    singleWindow, showStyle, showImage, theme,
    expireTime, lastAuthor, lastAuthores, searchPageAutoPullData,
    pictureInPictureVideoOffset, pictureInPictureVideoOffsetFullBefore,
    pictureInPictureVideoWidth, pictureInPictureVideoWidthFullBefore,
    isFullscreen, isElectron,
    addPlayingTagGoNext, goAuthorNewWidow, goSearchNewWidow,
    submitTagFromData, submitMutiTag,
    fileEditAutoCode, fileEditAutoJpg, fileEditAutoNext, fileEditAutoRefresh,
    tagSizeMap, shutdownLeftSecond, shutdownTimer,
    videoOptions,
    SearchWords, SearchRecords,
    playerMemory, playerRunning, playerReLocation,
    PlayingMovie, FileSearchParam, SettingInfo,
    SearchSuggestions, videoPlayTimes,
    // getters
    themeStyle, getSettingInfo, getControllerHost, getSuggestions, getSearchParam,
    // actions
    syncSearchParam, addRecords, keywordCount,
    setSettingInfo, setControllerHost,
    setPage, setPageSize, setMovieType, setKeyword, setSortField, setSortType, setOnlyRepeat,
    addPlayerLocation, getPlayerLocation,
    addSuggestions, savePlayTime, getPlayTime,
  };
}, {
  persist: {
    key: 'systemProperty',
    storage: localStorage,
    pick: [
      'singleWindow', 'showStyle', 'showImage', 'theme',
      'expireTime', 'lastAuthor', 'lastAuthores',
      'searchPageAutoPullData',
      'pictureInPictureVideoOffset', 'pictureInPictureVideoOffsetFullBefore',
      'pictureInPictureVideoWidth', 'pictureInPictureVideoWidthFullBefore',
      'isFullscreen', 'isElectron',
      'addPlayingTagGoNext', 'goAuthorNewWidow', 'goSearchNewWidow',
      'submitTagFromData', 'submitMutiTag',
      'fileEditAutoCode', 'fileEditAutoJpg', 'fileEditAutoNext', 'fileEditAutoRefresh',
      'tagSizeMap', 'shutdownLeftSecond',
      'videoOptions', 'SearchWords', 'SearchRecords',
      'playerMemory', 'playerRunning', 'playerReLocation',
      'PlayingMovie', 'FileSearchParam', 'SettingInfo',
      'SearchSuggestions', 'videoPlayTimes',
    ],
  },
});
