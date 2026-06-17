import { defineStore } from 'pinia'
import { FileQuery } from 'components/model/File'

// 搜索参数 / 历史记录 / 建议词
// 原 System.ts 中 FileSearchParam / SearchRecords / SearchWords / SearchSuggestions 相关

export const useSearchStore = defineStore({
  id: 'search',
  persist: {
    enabled: true,
    strategies: [{ key: 'searchProperty', storage: localStorage }],
  },
  state: () => ({
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
    SearchRecords: [] as Array<FileQuery>,
    SearchWords: {} as Record<string, number>,
    SearchSuggestions: [] as Array<string>,
    lastAuthor: '',
    lastAuthores: [] as string[],
  }),
  getters: {
    suggestions: (state) => state.SearchSuggestions,
    searchParam: (state) => state.FileSearchParam,
  },
  actions: {
    syncSearchParam(param: FileQuery) {
      const { Page, PageSize, MovieType, SortField, SortType, Keyword, showStyle } = param
      Object.assign(this.FileSearchParam, { Page, PageSize, MovieType, SortField, SortType, Keyword, showStyle })
      if (param.Keyword) this.addSuggestions(param.Keyword)
      if (Page == 1) this.keywordCount(param)
      this.addRecords(param)
    },
    addRecords(param: FileQuery) {
      if (!this.SearchRecords) this.SearchRecords = []
      const exist = this.SearchRecords.find(
        (x) => x.Keyword == param.Keyword && x.MovieType == param.MovieType &&
          x.Page == param.Page && x.PageSize == param.PageSize &&
          x.SortField == param.SortField && x.SortType == param.SortType,
      )
      if (exist) { exist.createdAt = new Date(); return }
      const rec = new FileQuery().fromObject(param)
      rec.createdAt = new Date()
      this.SearchRecords.unshift(rec)
      if (this.SearchRecords.length > 50) {
        for (let i = 0; i < 10; i++) this.SearchRecords.pop()
      }
    },
    keywordCount(param: FileQuery) {
      const { Keyword } = param
      if (!Keyword || !isNaN(Number(Keyword))) return
      this.SearchWords[Keyword] = (this.SearchWords[Keyword] || 0) + 1
    },
    addSuggestions(queryParam: string) {
      if (!queryParam) return
      if (!this.SearchSuggestions) this.SearchSuggestions = []
      const idx = this.SearchSuggestions.indexOf(queryParam)
      if (idx >= 0) this.SearchSuggestions.splice(idx, 1)
      this.SearchSuggestions.unshift(queryParam)
      if (this.SearchSuggestions.length > 100) this.SearchSuggestions.pop()
      localStorage.setItem('SearchSuggestions', JSON.stringify(this.SearchSuggestions))
    },
    setPage(page: number) { this.FileSearchParam.Page = page },
    setPageSize(size: number) { this.FileSearchParam.PageSize = size },
    setMovieType(type: string) { this.FileSearchParam.MovieType = type },
    setKeyword(kw: string) { this.FileSearchParam.Keyword = kw },
    setSortField(field: string) { this.FileSearchParam.SortField = field },
    setSortType(type: string) { this.FileSearchParam.SortType = type },
    setOnlyRepeat(v: boolean) { this.FileSearchParam.OnlyRepeat = v },
  },
})
