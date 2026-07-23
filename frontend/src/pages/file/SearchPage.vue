<template>
  <div>
    <q-layout view="lHh lpr lFf" container style="height: 93vh" class="shadow-2 rounded-borders"
      :class="{ 'theme-natural': systemProperty.theme === 'natural' }" :style="themeStyle">
      <!-- 头部 -->
      <q-header
        :style="[themeStyle, { backdropFilter: 'blur(10px)', boxShadow: '0 4px 12px rgba(0,0,0,0.15)', borderBottom: '1px solid var(--q-border)' }]"
        elevated class="q-gutter-sm flex justify-center">
        <!-- 索引按钮 -->
        <IndexButton v-permission="'op:scan'" ref="indexButton" @refresh-done="onIndexRefresh" glossy dense
          :size="btnSize('head')" />
        <!-- 用户行为偏好 -->
        <AppPreference />
        <!-- 重命名中指示 -->
        <q-btn v-if="pendingRenames > 0" color="red" text-color="white" size="md"
          icon="drive_file_rename_outline">
          {{ pendingRenames }}
          <q-tooltip>改名中 {{ pendingRenames }} 个文件</q-tooltip>
        </q-btn>

        <!-- 排序字段选择 -->
        <q-btn-dropdown glossy :size="btnSize('head')" class="w-5" :label="getLabelByValue(currentSort, sortOptions)">
          <q-list>
            <q-item v-for="item in sortOptions" :key="item.label" clickable v-close-popup @click="
              currentSort = item.value;
            fetchSearch();
            ">
              <q-item-section :class="{ 'text-blue': currentSort === item.value }">
                <q-item-label>{{ item.label }}</q-item-label>
              </q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>

        <!-- 电影类型选择   style="width: 26rem" -->
        <q-btn-toggle v-if="!isSmall" glossy push ripple stack :size="btnSize('head')" stretch toggleTextColor="red"
          toggleColor="blue" v-model="view.queryParam.MovieType" @update:model-value="fetchSearch()"
          :options="MovieTypeSelects" />

        <!-- 移动端电影类型选择 -->
        <q-btn-dropdown v-if="isSmall" glossy push ripple color="primary"
          :label="getLabelByValue(view.queryParam.MovieType, MovieTypeSelects)">
          <q-list>
            <q-item v-for="item in MovieTypeSelects" :key="item.label" clickable v-close-popup @click="
              view.queryParam.MovieType = item.value;
            fetchSearch();
            ">
              <q-item-section>
                <q-item-label>{{ item.label }}</q-item-label>
              </q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>

        <!-- 高级过滤按钮 -->
        <q-btn icon="ti-filter" dense :size="btnSize('head')" flat :color="hasAdvancedFilters ? 'orange' : 'grey'"
          @click="view.showAdvancedFilter = !view.showAdvancedFilter">
          <q-tooltip class="bg-white text-primary">高级过滤</q-tooltip>
        </q-btn>

        <!-- 搜索框 -->
        <q-input dense type="search" style="
          max-width: 400px;
          border-radius: 12px;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
          transition: all 0.3s ease;
        " outlined glossy :debounce="1000" id="searchBtn" v-model="view.queryParam.Keyword" filled clearable
          @clear="keywordChange" @update:model-value="keywordChange" class="search-input">
          <template v-slot:prepend>
            <q-icon name="ti-list" class="cursor-pointer">
              <q-popup-proxy>
                <div style="width: 200px; max-height: 50vh">
                  <q-list>
                    <q-item clickable v-ripple v-for="word in suggestions" :key="word" @click="
                      view.queryParam.Keyword = word;
                    fetchSearch();
                    ">
                      <q-item-section>
                        <q-item-label>{{ word }}</q-item-label>
                      </q-item-section>
                    </q-item>
                  </q-list>
                </div>
              </q-popup-proxy>
            </q-icon>
          </template>
          <template v-slot:append>
            <q-icon name="ti-search" title="搜" glossy class="cursor-pointer" @click="fetchSearch">
            </q-icon>
          </template>
        </q-input>
        <q-btn icon="ti-bar-chart-alt" class="cursor-pointer" dense :size="btnSize('head')" color="red" flat>
          <DataPop url />
        </q-btn>
        <!-- 仅重复项选择 -->
        <q-checkbox v-model="view.queryParam.OnlyRepeat" color="red" :keepColor="true" checked-icon="task_alt"
          unchecked-icon="ti-help" :val="true" flat dense @update:model-value="fetchSearch">
          <q-tooltip class="bg-white text-primary"> 去重 </q-tooltip>
        </q-checkbox>


        <q-btn v-if="isLarge || isMedium" flat dense no-caps style="align-items: center; align-content: center"
          @click="view.showNodeDialog = true">
          <q-icon :name="view.searchNodeDisplay === '本机' ? 'computer' : 'dns'" size="sm" class="q-mr-xs" />
          {{ view.searchNodeDisplay }}
          <q-icon name="arrow_drop_down" size="sm" />
        </q-btn>

        <!-- 设置按钮 -->
        <!-- Q-FAB 固定悬浮按钮 -->
        <q-btn icon="ti-pencil-alt" color="orange" glossy round :style="fabStyle" @click="openBatchEdit" />
      </q-header>

      <!-- 高级过滤面板 -->
      <q-slide-transition>
        <div v-show="view.showAdvancedFilter" class="advanced-filter-panel"
          :style="[themeStyle, { padding: '8px 12px' }]" @mouseleave="view.showAdvancedFilter = false">

          <!-- 作者聚合（已选中的隐藏，移到最底部） -->
          <div v-if="unselectedAuthors && unselectedAuthors.length > 0" class="row no-wrap q-mb-sm">
            <div class="text-caption text-grey-7" style="width: 48px; line-height: 28px; flex-shrink: 0;">作者</div>
            <div style="display: flex; flex-wrap: wrap; gap: 4px;">
              <q-chip v-for="item in (unselectedAuthors || [])" :key="item.name" color="indigo-1" text-color="indigo-7"
                size="md" dense clickable @click="toggleAggFilter('filterAuthor', item.name)">
                {{ item.name }} ({{ item.cnt }})
              </q-chip>
            </div>
          </div>

          <!-- 标签聚合（已选中的隐藏，移到最底部） -->
          <div v-if="unselectedTags && unselectedTags.length > 0" class="row no-wrap q-mb-sm">
            <div class="text-caption text-grey-7" style="width: 48px; line-height: 28px; flex-shrink: 0;">标签</div>
            <div style="display: flex; flex-wrap: wrap; gap: 4px;">
              <q-chip v-for="item in (unselectedTags || [])" :key="item.name" color="orange-1" text-color="orange-8"
                size="md" dense clickable @click="toggleAggFilter('filterTag', item.name)">
                {{ item.name }} ({{ item.cnt }})
              </q-chip>
            </div>
          </div>

          <!-- 系列聚合（已选中的隐藏，移到最底部） -->
          <div v-if="unselectedSeries && unselectedSeries.length > 0" class="row no-wrap q-mb-sm">
            <div class="text-caption text-grey-7" style="width: 48px; line-height: 28px; flex-shrink: 0;">系列</div>
            <div style="display: flex; flex-wrap: wrap; gap: 4px;">
              <q-chip v-for="item in (unselectedSeries || [])" :key="item.name" color="green-1" text-color="green-7"
                size="md" dense clickable @click="toggleAggFilter('filterSeries', item.name)">
                {{ item.name }} ({{ item.cnt }})
              </q-chip>
            </div>
          </div>

          <!-- 动态大小快捷（根据搜索结果 min/max 生成） -->
          <div v-if="sizePresets && sizePresets.length > 0" class="row no-wrap q-mb-md" style="padding: 4px 0;">
            <div class="text-caption text-grey-7" style="width: 48px; line-height: 28px; flex-shrink: 0;">大小</div>
            <div style="display: flex; flex-wrap: wrap; gap: 6px;">
              <q-btn v-for="p in (sizePresets || [])" :key="p.label" dense flat size="md"
                :color="p.label.startsWith('<') ? 'deep-orange-6' : 'indigo-6'" :label="p.label"
                @click="applySizePreset(p.min, p.max)" />
            </div>
          </div>

          <q-separator class="q-my-sm" />

          <!-- 动态日期快捷 -->
          <div v-if="datePresets && datePresets.length > 0" class="row no-wrap q-mb-sm">
            <div class="text-caption text-grey-7" style="width: 48px; line-height: 28px; flex-shrink: 0;">日期</div>
            <div style="display: flex; flex-wrap: wrap; gap: 4px;">
              <q-btn v-for="p in (datePresets || [])" :key="p.label" dense flat size="md" color="primary"
                :label="p.label" :outline="!!p.outline" @click="applyDatePreset(p.days)" />
            </div>
          </div>

          <!-- 动态扩展名快捷 -->
          <div v-if="extPresets && extPresets.length > 0" class="row no-wrap q-mb-sm">
            <div class="text-caption text-grey-7" style="width: 48px; line-height: 28px; flex-shrink: 0;">扩展名</div>
            <div style="display: flex; flex-wrap: wrap; gap: 4px;">
              <q-chip v-for="e in (extPresets || [])" :key="e.ext" :color="isExtSelected(e.ext) ? 'primary' : 'grey-3'"
                :text-color="isExtSelected(e.ext) ? 'white' : 'grey-8'" size="md" dense clickable
                @click="toggleExt(e.ext)" :removable="isExtSelected(e.ext)" @remove="toggleExt(e.ext)">
                {{ e.ext }} ({{ e.cnt }})
              </q-chip>
            </div>
          </div>

          <q-separator class="q-my-sm" />

          <!-- 已选中的过滤条件 + 重置 -->
          <div class="row justify-between items-center">
            <div class="row q-gutter-xs items-center" v-if="selectedFilterChips.length > 0">
              <span class="text-caption text-grey-7 q-mr-xs">已选</span>
              <template v-for="chip in selectedFilterChips" :key="chip.key">
                <q-chip v-if="chip.label" dense size="md" :color="chip.color" text-color="white" removable
                  @remove="chip.onRemove()">
                  {{ chip.label }}
                </q-chip>
              </template>
            </div>
            <div style="flex:1;"></div>
            <q-btn dense flat size="md" color="red" icon="ti-close" label="重置" @click="clearAdvancedFilters" />
          </div>
        </div>
      </q-slide-transition>

      <!-- 底部 -->
      <q-footer elevated :style="themeStyle" class="glossy">
        <div class="flex flex-center">
          <!-- 分页器 -->
          <q-pagination v-model="view.queryParam.Page" @update:model-value="gotoPageNo" color="deep-orange"
            :ellipses="true" :max="view.resultData.TotalPage || 0" :max-pages="isSmall ? 5 : 10" boundary-numbers
            direction-links></q-pagination>
          <!-- 每页大小 -->
          <q-select size="sm" dense borderless dark class="q-ml-sm" @update:model-value="currentPageSizeChange"
            v-model="view.queryParam.PageSize" :options="pageOptions" style="min-width: 60px" />
          <!-- 页码直达 -->
          <q-input v-model="gotoPage" dense dark borderless class="q-ml-sm" style="width: 56px; text-align: center"
            placeholder="#" @update:model-value="pageNoGoto" />

        </div>
        <div style="position: fixed; right: 10px; bottom: 40px; z-index: 10">
          <q-btn icon="history" color="blue" glossy>
            <q-popup-proxy v-model="view.showHistory" class="history-popup">
              <q-card flat bordered class="no-shadow" style="width: 380px">
                <q-card-section class="q-pa-sm row items-center justify-between">
                  <q-btn flat dense size="sm" color="red" icon="delete_sweep" label="清空"
                    @click="systemProperty.SearchRecords = []; systemProperty.SearchWords = {}" />
                </q-card-section>

                <q-tabs v-model="view.historyTab" dense no-caps class="text-grey-7 q-mx-sm" active-color="primary"
                  indicator-color="primary">
                  <q-tab name="records" label="记录" />
                  <q-tab name="keywords" label="关键词" />
                </q-tabs>

                <q-tab-panels v-model="view.historyTab" animated class="bg-transparent">
                  <!-- 关键词面板 -->
                  <q-tab-panel name="keywords" class="q-pa-sm">
                    <div v-if="Object.keys(systemProperty.SearchWords).length" class="row q-gutter-xs"
                      style="max-height: 40vh; overflow-y: auto">
                      <q-chip v-for="(count, word) in systemProperty.SearchWords" :key="word" clickable size="sm"
                        color="red" text-color="white" @click="searchByKeyword(word)">
                        {{ word }}
                        <q-badge v-if="count > 1" color="red" floating>{{ count }}</q-badge>
                      </q-chip>
                    </div>
                    <div v-else class="text-caption text-grey-5 text-center q-py-md">
                      暂无关键词记录
                    </div>
                  </q-tab-panel>

                  <!-- 记录面板 -->
                  <q-tab-panel name="records" class="q-pa-sm">
                    <q-list v-if="sortedSearchRecords.length" dense separator
                      style="max-height: 40vh; overflow-y: auto">
                      <q-item v-for="(his, idx) in sortedSearchRecords" :key="idx" clickable v-ripple v-close-popup
                        @click="redirectUrl(his)" class="q-px-sm q-py-xs">
                        <q-item-section>
                          <q-item-label class="text-caption text-blue row justify-between">
                            <span> {{ his.Keyword || '无' }} &mdash;
                              {{ his.Page }}/{{ his.PageSize }}
                              {{ getLabelByValue(his.MovieType, MovieTypeOptions) || '全部' }}</span>
                            <span>{{ date.formatDate(his.createdAt, 'MM-DD HH:mm') }}</span>

                          </q-item-label>
                        </q-item-section>
                      </q-item>
                    </q-list>
                    <div v-else class="text-caption text-grey-5 text-center q-py-md">
                      暂无搜索记录
                    </div>
                  </q-tab-panel>
                </q-tab-panels>
              </q-card>
            </q-popup-proxy>
          </q-btn>
        </div>
      </q-footer>
      <!-- 页面内容 -->
      <q-page-container class="scrollRef">
        <q-page>
          <div class="row q-gutter-sm justify-start q-pl-sm"
            v-if="view.resultData.Data && view.resultData.Data.length > 0">
            <!-- 卡片列表 -->
            <q-card v-for="item in view.resultData.Data" :key="item.Id" :id="item.Id" v-bind:class="{
              'large-result': isLarge,
              'medium-result': isMedium,
              'small-result': isSmall,
            }" class="search-result-card" :style="{
              transition: isFetching ? 'none' : 'all 0.3s ease-out',
              backgroundColor:
                item.Id == view.currentDataInPlayer.Id
                  ? 'rgba(99, 102, 241, 0.2)'
                  : item.Id == view.currentDataInEditor.Id
                    ? 'rgba(234, 179, 8, 0.2)'
                    : 'var(--q-bg-card)',
            }">
              <div class="card-top-tag" style="width: 80%">
                <!-- 种草按钮 -->
                <q-btn text-color="white" color="red" :size="btnSize('top')" dense class="glossy"
                  style="max-width: 5rem" @contextmenu="
                    (e) => {
                      refreshDebounceFn(item);
                      e.returnValue = false;
                    }
                  ">
                  <q-popup-proxy style="background-color: rgba(250, 250, 250, 0.9)">
                    <TagPop :currentData="item" :current-tag="item.Tags" :delay="10" />
                  </q-popup-proxy>
                  <span>{{ `种草/${item.PageNo}` }}</span>
                </q-btn>

                <!-- 标签列表 -->

                <q-chip square dense v-for="tag in item.Tags" :key="tag" :size="btnSize('top')" class="chip-tag">
                  <span @click="searchKeyword(tag)">{{
                    tag?.substring(0, 4)
                    }}</span>
                </q-chip>
              </div>
              <div class="card-top-type">
                <!-- 电影类型选择按钮 -->
                <q-btn dense :size="btnSize('top')" class="glossy" color="primary"
                  :label="`${item.MovieType === '无' ? `分类 ` : item.MovieType}`">
                  <q-menu>
                    <q-list class="menu-min-w-sm">
                      <q-item v-for="mt in MovieTypeOptions" :key="mt.value" clickable v-close-popup>
                        <q-item-section @click="
                          setMovieType(item.Id, mt.value);
                        item.btnMovieType = false;
                        ">{{ mt.label }}</q-item-section>
                      </q-item>
                      <q-item clickable v-close-popup>
                        <q-item-section class="tag-blue" @click="refreshDebounceFn(item)">刷新</q-item-section>
                      </q-item>
                    </q-list>
                  </q-menu>
                </q-btn>
                <q-btn dense glossy color="grey" size="sm" class="mt-1" v-if="formatSeries(item.Code)">
                  <span @click="searchKeyword(formatSeries(item.Code))">{{
                    formatSeries(item.Code).substring(0, 4)
                    }}</span>
                </q-btn>
                <q-btn dense flat text-color="green" size="sm" class="mt-1" v-if="systemProperty.getPlayTime(item.Id)">
                  <span>{{ formatPlayTime(systemProperty.getPlayTime(item.Id)) }}</span>
                </q-btn>
                <!-- 文件类型标签 -->
                <q-chip square v-if="item.FileType != 'mp4'" :size="btnSize('top')" dense color="orange">
                  <span @click="searchKeyword(item.FileType)">
                    {{ item.FileType }}</span>
                </q-chip>
              </div>
              <!-- 图片 -->
              <q-img fit="fill" :lazy="true" :class="{
                'large-result-image': isLarge,
                'medium-result-image': isMedium,
                'small-result-image': isSmall,
                'card-img': true,
              }" :src="getImage(item)" @contextmenu="(e) => pictureRightClick(item, e)" @click="openFileInfoRef(item)">
                <template v-slot:loading>
                  <q-spinner-ios color="white" size="2em">Loading...</q-spinner-ios>
                </template>
                <template v-slot:error>
                  <!-- 图片加载失败时显示的占位图 -->
                  <div class="text-subtitle1 text-white">
                    <q-icon name="image_not_supported" size="2em"></q-icon>
                  </div>
                </template>
                <q-inner-loading :showing="item.Id == view.currentDataInEditor.Id">
                  <q-spinner-gears size="80px" color="primary" label="编辑中" />
                </q-inner-loading>
                <q-inner-loading :showing="item.Id == view.currentDataInPlayer.Id" label="播放中" label-class="text-teal"
                  label-style="font-size: 1.1em" />
              </q-img>
              <div class="absolute-bottom float-btn card-btn-bar">
                <div>
                  <div class="btn-row">
                    <!-- 播放按钮 -->
                    <q-btn round ripple flat glossy color="white" :size="btnSize('footer')" icon="play_circle_outline"
                      @click="playBySystem(item)" title="播放" v-if="!isSmall" />
                    <!-- 单页播放按钮 -->
                    <q-btn round flat ripple glossy color="yellow" :size="btnSize('footer')" icon="fullscreen"
                      title="单页播放" @click="playByPage(item)" />
                    <!-- 小播放按钮 -->
                    <q-btn round flat ripple glossy color="blue" :size="btnSize('footer')" icon="tv"
                      @click="openFileInfoRef(item, true)" title="小播放" />
                    <!-- 画中画按钮 -->
                    <q-btn round flat ripple glossy color="white" :size="btnSize('footer')" icon="picture_in_picture"
                      @click="picInPic(item)" @contextmenu="
                        (e) => {
                          picInPic(item, true);
                          e.returnValue = false;
                        }
                      " title="画中画" />
                  </div>
                  <div class="btn-row btn-row-responsive">
                    <!-- 编辑按钮 -->
                    <q-btn round ripple glossy :size="btnSize('footer')" color="grey-8" icon="edit" @click="
                      view.currentDataInEditor = item;
                    fileEditRef.open(item);
                    " title="编辑" style="box-shadow: 0 2px 6px rgba(128, 128, 128, 0.2)" />
                    <!-- 文件夹按钮 -->
                    <q-btn round ripple glossy :size="btnSize('footer')" color="primary" icon="open_in_new"
                      @click="openFolder(item)" v-if="!isSmall" title="文件夹" />
                    <!-- 网搜按钮 -->
                    <q-btn round ripple glossy :size="btnSize('footer')" color="brown-5" icon="ti-search" title="网搜"
                      @click="searchCode(item)" />
                    <!-- 截图按钮 -->
                    <q-btn round ripple glossy :size="btnSize('footer')" color="black" @click="
                      () => {
                        view.currentDataInEditor = item;
                        fileCutImageRef.open(item);
                      }
                    " icon="ti-cut" title="截图" />
                    <!-- 删除按钮 -->
                    <q-btn round ripple glossy :size="btnSize('footer')" color="negative" icon="delete" title="删除"
                      @click="confirmDelete(item)" />
                    <!-- 扫码按钮 -->
                    <q-btn round ripple glossy :size="btnSize('footer')" color="teal" icon="qr_code_scanner" title="扫码"
                      v-if="!isSmall" @click="openQrDownload(item)" />
                    <!-- 更多按钮 -->
                    <q-btn round ripple glossy :size="btnSize('footer')" color="grey-7" icon="more_vert" title="更多">
                      <q-menu anchor="top left" self="bottom left" transition-show="jump-down"
                        transition-hide="jump-up">
                        <q-list style="min-width: 120px">
                          <q-item clickable v-close-popup @click="
                            view.currentDataInEditor = item;
                          fileEditRef.open(item);
                          ">
                            <q-item-section avatar><q-icon name="edit" color="grey-8" /></q-item-section>
                            <q-item-section>编辑</q-item-section>
                          </q-item>
                          <q-item clickable v-close-popup @click="openFolder(item)" v-if="!isSmall">
                            <q-item-section avatar><q-icon name="open_in_new" color="primary" /></q-item-section>
                            <q-item-section>文件夹</q-item-section>
                          </q-item>
                          <q-item clickable v-close-popup @click="searchCode(item)">
                            <q-item-section avatar><q-icon name="ti-search" color="brown-5" /></q-item-section>
                            <q-item-section>网搜</q-item-section>
                          </q-item>
                          <q-item clickable v-close-popup @click="
                            view.currentDataInEditor = item;
                          fileCutImageRef.open(item);
                          ">
                            <q-item-section avatar><q-icon name="ti-cut" color="black" /></q-item-section>
                            <q-item-section>截图</q-item-section>
                          </q-item>
                          <q-item clickable v-close-popup @click="confirmDelete(item)">
                            <q-item-section avatar><q-icon name="delete" color="negative" /></q-item-section>
                            <q-item-section>删除</q-item-section>
                          </q-item>
                          <q-item clickable v-close-popup @click="openQrDownload(item)" v-if="!isSmall">
                            <q-item-section avatar><q-icon name="qr_code_scanner" color="teal" /></q-item-section>
                            <q-item-section>扫码</q-item-section>
                          </q-item>
                        </q-list>
                      </q-menu>
                    </q-btn>
                  </div>
                </div>

                <div class="content-row" :style="{
                  ...themeStyle, height: isLarge ? '54px' : '38px',
                  fontSize: isLarge ? '14px' : '14px',
                }">
                  <span style="
                    color: green;
                    margin-right: 1px;
                  " class="cursor-pointer">{{ getTimeAgo(item.MTime) }}
                    <q-popup-proxy>
                      <div style="
                        width: 400px;
                        padding: 10px;
                        background-color: rgba(0, 0, 0, 0.1);
                      ">
                        <div>
                          <span style="
                            color: rgb(161, 100, 19);
                            background-color: rgba(0, 0, 0, 0.1);
                            margin-right: 1px;
                          " class="cursor-pointer" @click="copyText(item.Author)">{{ item.Author }}</span>
                        </div>
                        <div>
                          <span class="tag-red cursor-pointer" @click="copyText(item.Code)">{{
                            item.Code }}</span>
                        </div>
                        <div>
                          {{ formatTitle(item.Title) }}
                        </div>
                        <div class="tag-green cursor-pointer" @click="searchKeyword(item.BaseDir)">
                          {{ item.BaseDir }}
                        </div>
                        <div class="tag-gray">
                          {{ item.Path }}
                        </div>
                      </div>
                    </q-popup-proxy>
                  </span>
                  <span @click="copyText(item.Title)" class="cursor-pointer" style="

                    margin-right: 1px;
                  ">
                    {{ humanStorageSize(item.Size) }}
                  </span>
                  <span style="
                    color: green;
                    margin-right: 1px;
                  " class="cursor-pointer" @click="goAuthor(item.Author)">{{ item.Author }}</span>

                  <span style="
                    color: orange;
                    margin-right: 4px;
                  " class="cursor-pointer" @click="copyText(item.Code)">{{ item.title ? item.Code?.substring(0,
                    12) : item.Code }}
                    <q-tooltip class="bg-white text-primary">{{
                      item.Code
                      }}</q-tooltip>
                  </span>

                  {{ formatTitle(item.Title) }}
                </div>
              </div>
            </q-card>
          </div>

          <!-- 扫码下载弹窗 -->
          <QrDownloadDialog v-model="qrDownloadVisible" :item="qrDownloadItem" />

          <!-- 页面滚动器 -->
          <q-page-scroller position="bottom-right" :scroll-offset="150" :offset="[18, 100]">
            <q-btn fab icon="keyboard_arrow_up" color="accent" />
          </q-page-scroller>

          <div v-if="view.queryParam.Page < view.resultData.TotalPage">
            <div style="height: 8vh">
              <q-btn v-show="systemProperty.searchPageAutoPullData && isMoreLoading" color="primary"
                label="加载中..."></q-btn>
            </div>

            <div v-intersection="onIntersection" style="height: 8vh; color: #9e089e" @click="
              () => {
                pullNextPage();
              }
            ">
              点击可加载更多数据
            </div>
          </div>
        </q-page>
        <!-- 上一页按钮 -->
        <q-page-sticky style="z-index: 9" position="bottom-left" v-if="view.queryParam.Page > 1"
          :offset="[6, isSmall ? 200 : 366]">
          <q-btn glossy class="page-sticky" flat text-color="blue" :label="`P${view.queryParam.Page - 1}`"
            @click="gotoPageNo(view.queryParam.Page - 1)"></q-btn>
        </q-page-sticky>

        <!-- 下一页按钮 -->
        <q-page-sticky style="z-index: 9" position="bottom-right" :offset="[10, isSmall ? 200 : 366]">
          <q-btn v-if="view.queryParam.Page < view.resultData.TotalPage" flat glossy class="page-sticky"
            text-color="blue" :label="`P${view.queryParam.Page + 1}`"
            @click="gotoPageNo(view.queryParam.Page + 1)"></q-btn>
        </q-page-sticky>
      </q-page-container>
    </q-layout>

    <!-- 节点选择弹窗 -->
    <q-dialog v-model="view.showNodeDialog">
      <q-card style="min-width: 400px" class="theme-card">
        <q-card-section class="q-pa-sm">
          <div class="text-subtitle2 q-mb-sm">选择搜索节点</div>
          <q-list bordered separator dense>
            <q-item clickable v-close-popup :active="!view.queryParam.SearchNode" @click="selectNode('')"
              class="q-py-xs">
              <q-item-section avatar>
                <q-icon name="computer" color="primary" />
              </q-item-section>
              <q-item-section>
                <q-item-label class="text-caption">本机</q-item-label>
                <q-item-label caption class="text-caption">{{ view.localNodeName }}</q-item-label>
              </q-item-section>
              <q-item-section side v-if="!view.queryParam.SearchNode">
                <q-icon name="check" color="positive" />
              </q-item-section>
            </q-item>
            <q-item clickable v-close-popup v-for="peer in view.nodeList" :key="peer.id"
              :active="view.queryParam.SearchNode === peer.id" @click="selectNode(peer.id)" class="q-py-xs">
              <q-item-section avatar>
                <q-icon name="dns" color="purple" />
              </q-item-section>
              <q-item-section>
                <q-item-label class="text-caption">{{ peer.name || peer.id }}</q-item-label>
                <q-item-label caption class="text-caption">
                  {{ peer.totalCnt }} 文件 · {{ peer.totalSize }}
                </q-item-label>
              </q-item-section>
              <q-item-section side v-if="view.queryParam.SearchNode === peer.id">
                <q-icon name="check" color="positive" />
              </q-item-section>
            </q-item>
          </q-list>
          <div v-if="view.nodeList.length === 0" class="text-center text-grey text-caption q-py-sm">
            暂无在线节点
          </div>
        </q-card-section>
        <q-card-actions align="right" class="q-pa-sm q-pt-none">
          <q-btn flat dense label="关闭" color="grey" v-close-popup />
          <q-btn flat dense icon="refresh" color="primary" @click="fetchNodeList">刷新</q-btn>
        </q-card-actions>
      </q-card>
    </q-dialog>

    <!-- 视频播放器 -->

    <InnerVideoPlayer ref="videoRef" @next-one="viewNextOne('play')" @prev-one="viewPrevOne('play')"
       @choose-data="
        (d) => {
          saveParam();
          view.currentDataInPlayer = d;
        }
      " @close="view.currentDataInPlayer = {}" />

    <!-- 文件编辑对话框 -->
    <FileEdit ref="fileEditRef" @next-one="viewNextOne('info')" @prev-one="viewPrevOne('info')"
       @hide="view.currentDataInEditor = {}" />
    <!-- 文件信息对话框 -->
    <FileInfo ref="fileInfoRef" @next-one="viewNextOne('edit')" @prev-one="viewPrevOne('edit')"
      @hide="view.currentDataInEditor = {}" />

    <!-- 批量操作 / 任务列表（已合并） -->
    <BatchEdit ref="batchEditRef" />
    <!-- 截图对话框 -->
    <Screenshot ref="fileCutImageRef" @next-one="viewNextOne('cut')" @prev-one="viewPrevOne('cut')"
      @hide="view.currentDataInEditor = {}" @close="
        () => {
          fetchSearch();
        }
      " />
    <q-dialog v-model="moveView.targetPathDialog" title="移动文件">
      <q-card style="min-width: 350px; width: 600px">
        <q-card-section>
          <q-input bg-color="green" label="原始地址" outlined :readonly="true" stack-label filled autogrow
            v-model="moveView.originPath" @click="moveView.targetPath = moveView.originPath">
          </q-input>
        </q-card-section>

        <q-card-section class="q-pt-none">
          <q-input label="文件夹" stack-label filled autogrow v-model="moveView.targetPath">
          </q-input>
        </q-card-section>
        <q-card-section class="q-pt-none">
          <q-input label="文件名" stack-label filled autogrow v-model="moveView.targetName" />
        </q-card-section>

        <q-card-actions align="right" class="text-primary">
          <q-btn flat label="取消" v-close-popup />
          <q-btn flat label="移动" @click="moveThis" v-close-popup />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </div>
</template>

<script setup>
import { date, format, useQuasar } from 'quasar';
const { humanStorageSize } = format;

// 卡片大小样式（基于 System store showStyle）
const { fromStyle } = useBreakpoint()
const { isSmall, isMedium, isLarge } = fromStyle(() => systemProperty.showStyle)

import {
  DeleteFile,
  MoveFile,
  OpenFileFolder,
  PlayMovie,
  ResetMovieType,
  SearchAPI,
} from 'components/api/searchAPI';
import { computed, onMounted, onUnmounted, provide, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { GetSettingInfo, GetLanPeersWithStats } from 'components/api/settingAPI';
import {
  formatSeries,
  formatTitle,
  getLabelByValue,
  MovieTypeOptions,
  MovieTypeSelects,
} from 'components/utils';
import { SSEEventType } from 'src/types';


const getImage = (item) => {
  if (systemProperty.showImage === 'poster') {
    return item.PngUrl;
  }
  return item.JpgUrl;
};

import DataPop from 'components/DataPop.vue';
import IndexButton from 'components/IndexButton.vue';
import TagPop from 'components/TagPop.vue';
import AppPreference from 'components/AppPreference.vue';
import { useSystemProperty } from 'stores/System';
import FileEdit from './components/FileEditDialog.vue';
import FileInfo from './components/FileInfoDialog.vue';
import BatchEdit from './components/BatchEditDialog.vue';
import Screenshot from './components/ScreenshotDialog.vue';
import InnerVideoPlayer from './components/VideoPlayerInPicture.vue';
import QrDownloadDialog from 'components/QrDownloadDialog.vue';

import { onKeyStroke, useClipboard, useDebounceFn } from '@vueuse/core';
import { getTimeAgoShort as getTimeAgo } from 'src/utils/date';
import { useSortOptions } from 'src/composables/useSortOptions';
import { useCommonExec } from 'src/composables/useCommonExec';
import { useBreakpoint } from 'src/composables/useBreakpoint';
import { useSSE } from 'src/composables/useSSE';

// SSE 实时更新
let sseDebounceTimer = null
const debouncedFetchSearch = () => {
  clearTimeout(sseDebounceTimer)
  sseDebounceTimer = setTimeout(() => fetchSearch(), 2000)
}
const pendingRenames = ref(0);

const handleSSEEvent = (event) => {
  if (event.Type === SSEEventType.RenameStart) {
    pendingRenames.value = event.Data?.count || 0;
  }
  if (event.Type === SSEEventType.FileChanged) {
    debouncedFetchSearch()
    const action = event.Data?.action;
    const path = event.Data?.path || event.Data?.new || '';
    if (action === 'delete') {
      $q.notify({ type: 'info', message: `已删除: ${path}`, position: 'bottom-left', timeout: 2000 });
    } else if (action === 'rename' || action === 'move') {
      $q.notify({ type: 'info', message: `文件已移动/重命名`, position: 'bottom-left', timeout: 2000 });
    }
  }
  if (event.Type === SSEEventType.ScanStart) {
    indexButton.value?.queryHealth();
    const total = event.Data?.totalDirs || '';
    $q.notify({ type: 'info', message: `开始扫描 ${total} 个目录...`, position: 'bottom-left', timeout: 2000 });
  }
  if (event.Type === SSEEventType.ScanOneDone) {
    indexButton.value?.queryHealth();
  }
  if (event.Type === SSEEventType.ScanComplete) {
    indexButton.value?.queryHealth();
    debouncedFetchSearch();
    const cnt = event.Data?.fileCount || '';
    $q.notify({ type: 'positive', message: `扫描完成，共 ${cnt} 个文件`, position: 'bottom-left', timeout: 3000 });
  }
  if (event.Type === SSEEventType.ScanError) {
    const dir = event.Data?.dir || '';
    $q.notify({ type: 'negative', message: `扫描 "${dir}" 失败`, position: 'bottom-left', timeout: 5000 });
  }
  if (event.Type === SSEEventType.IndexUpdate) {
    indexButton.value?.queryHealth();
  }
  if (event.Type === SSEEventType.IndexHealth && indexButton.value) {
    indexButton.value.updateHealth(event.Data);
  }
};
useSSE(handleSSEEvent);

// 变量声明
const $q = useQuasar();
const { exec: commonExec } = useCommonExec();
const fileEditRef = ref(null);
const fileInfoRef = ref(null);
const batchEditRef = ref(null);
const videoRef = ref(null);
const indexButton = ref(null);
const fileCutImageRef = ref(null);
const isMoreLoading = ref(false);
const isFetching = ref(false);
const pageOptions = ref([10, 12, 14, 20, 30, 50, 200]);
// AbortController 用于取消前一个搜索请求
const searchAbortController = ref(null);



const moveView = reactive({
  targetPath: '',
  targetPathDialog: false,
  targetId: '',
});

// 悬浮按钮位置
const fabPos = reactive({ x: 10, y: 150 });

const fabStyle = computed(() => ({
  position: 'fixed',
  right: `${fabPos.x}px`,
  top: `${fabPos.y}px`,
}));



const view = reactive({
  currentDataInPlayer: {},
  currentDataInEditor: {},
  settingInfo: {},
  allPageNo: 0,
  showAdvancedFilter: false,
  playBy: '',
  queryParam: {
    Keyword: '',
    MovieType: '',
    OnlyRepeat: false,
    Page: 1,
    PageSize: 20,
    SortField: 'MTime',
    SortType: 'desc',
    SearchNode: '',
    minSize: 0,
    maxSize: 0,
    dateFrom: '',
    dateTo: '',
    fileExts: [],
    filterAuthor: '',
    filterTag: '',
    filterSeries: '',
  },
  resultData: {},
  searchNodeDisplay: '本机',
  localNodeName: '',
  showNodeDialog: false,
  nodeList: [],
  showHistory: false,
  historyTab: 'records',
});

const sortOptions = useSortOptions('   ');

// ========== 高级过滤 ==========
const onFilterChange = () => {
  view.queryParam.Page = 1;
  fetchSearch();
};

const hasAdvancedFilters = computed(() => {
  return (view.queryParam.minSize > 0) ||
    (view.queryParam.maxSize > 0) ||
    (view.queryParam.dateFrom !== '' && view.queryParam.dateFrom != null) ||
    (view.queryParam.dateTo !== '' && view.queryParam.dateTo != null) ||
    (Array.isArray(view.queryParam.fileExts) && view.queryParam.fileExts.length > 0) ||
    (view.queryParam.filterAuthor !== '') ||
    (view.queryParam.filterTag !== '') ||
    (view.queryParam.filterSeries !== '');
});

const applySizePreset = (minBytes, maxBytes) => {
  view.queryParam.minSize = minBytes;
  view.queryParam.maxSize = maxBytes;
  onFilterChange();
};

const applyDatePreset = (daysAgo) => {
  const now = new Date();
  const from = new Date(now.getTime() - daysAgo * 86400000);
  view.queryParam.dateFrom = from.toISOString().slice(0, 10);
  view.queryParam.dateTo = now.toISOString().slice(0, 10);
  onFilterChange();
};

// 根据搜索结果大小范围生成快捷按钮
const sizePresets = computed(() => {
  const agg = aggregates.value;
  if (!agg || !agg.minSize || !agg.maxSize) return [];
  const min = agg.minSize;
  const max = agg.maxSize;
  if (min >= max) return [];

  const GB = 1073741824;
  const MB = 1048576;
  const presets = [];

  const greaterLabels = ['>100MB', '>500MB', '>1GB', '>5GB', '>10GB', '>50GB', '>100GB'];
  const greaterSizes = [MB * 100, MB * 500, GB, GB * 5, GB * 10, GB * 50, GB * 100];
  let firstIdx = -1;
  for (let i = 0; i < greaterSizes.length && presets.length < 5; i++) {
    if (greaterSizes[i] > min && greaterSizes[i] < max) {
      if (firstIdx === -1) firstIdx = i;
      presets.push({ label: greaterLabels[i], min: greaterSizes[i], max: 0 });
    }
  }

  // 添加 <min 按钮（小于最小区间阈值）
  if (firstIdx >= 0) {
    const lessLabel = greaterLabels[firstIdx].replace('>', '<');
    presets.unshift({ label: lessLabel, min: 0, max: greaterSizes[firstIdx] });
  }

  return presets;
});

// 根据搜索结果日期范围生成日期快捷按钮
const datePresets = computed(() => {
  const agg = aggregates.value;
  if (!agg || !agg.maxDate) return [];
  const now = Date.now() / 1000;
  const maxDate = agg.maxDate;
  const range = now - maxDate; // 最老文件距今秒数
  const presets = [];

  const DAY = 86400;

  const items = [
    { label: '近一周', days: 7 },
    { label: '近一月', days: 30 },
    { label: '近一年', days: 365 },
    { label: '近五年', days: 1825 },
    { label: '近十年', days: 3650 },
  ];

  for (const item of items) {
    const secs = item.days * DAY;
    presets.push({
      label: item.label,
      days: item.days,
      outline: secs < range, // 若范围大于预设值，灰色描边
    });
  }

  return presets;
});

// 从后台搜索结果提取扩展名聚合 + 按数量排序
const extPresets = computed(() => {
  const agg = aggregates.value;
  if (!agg || !agg.exts) return [];
  return Object.entries(agg.exts)
    .map(([ext, info]) => ({ ext, cnt: info.Cnt || 0 }))
    .sort((a, b) => b.cnt - a.cnt)
    .slice(0, 10);
});

const isExtSelected = (ext) => {
  return Array.isArray(view.queryParam.fileExts) && view.queryParam.fileExts.includes(ext);
};

const toggleExt = (ext) => {
  if (!Array.isArray(view.queryParam.fileExts)) {
    view.queryParam.fileExts = [];
  }
  const idx = view.queryParam.fileExts.indexOf(ext);
  if (idx >= 0) {
    view.queryParam.fileExts.splice(idx, 1);
  } else {
    view.queryParam.fileExts.push(ext);
  }
  view.queryParam.Page = 1;
  fetchSearch();
};

const clearAdvancedFilters = () => {
  view.queryParam.minSize = 0;
  view.queryParam.maxSize = 0;
  view.queryParam.dateFrom = '';
  view.queryParam.dateTo = '';
  view.queryParam.fileExts = [];
  view.queryParam.filterAuthor = '';
  view.queryParam.filterTag = '';
  view.queryParam.filterSeries = '';
  onFilterChange();
};

// 确保从旧版 localStorage/pinia 恢复的 queryParam 包含高级过滤默认字段
const ensureFilterDefaults = () => {
  if (view.queryParam.minSize == null) view.queryParam.minSize = 0;
  if (view.queryParam.maxSize == null) view.queryParam.maxSize = 0;
  if (view.queryParam.dateFrom == null) view.queryParam.dateFrom = '';
  if (view.queryParam.dateTo == null) view.queryParam.dateTo = '';
  if (!Array.isArray(view.queryParam.fileExts)) view.queryParam.fileExts = [];
  if (view.queryParam.filterAuthor == null) view.queryParam.filterAuthor = '';
  if (view.queryParam.filterTag == null) view.queryParam.filterTag = '';
  if (view.queryParam.filterSeries == null) view.queryParam.filterSeries = '';
};

// ========== 聚合数据（作者/标签/系列） ==========
// 从搜索结果中提取聚合数据
const aggregates = computed(() => {
  return view.resultData?.Aggregates || null;
});

// 将聚合 map 转为排序后的数组
const sortAggEntries = (mapData) => {
  if (!mapData) return [];
  return Object.entries(mapData)
    .map(([name, info]) => ({ name, cnt: info.Cnt || 0 }))
    .sort((a, b) => b.cnt - a.cnt)
    .slice(0, 20); // 最多显示前 20 个
};

const aggregatesAuthors = computed(() => sortAggEntries(aggregates.value?.authors));
const aggregatesTags = computed(() => sortAggEntries(aggregates.value?.tags));
const aggregatesSeries = computed(() => sortAggEntries(aggregates.value?.series));

// 未选中的（在聚合区域显示）
const unselectedAuthors = computed(() => {
  const list = aggregatesAuthors.value;
  if (!Array.isArray(list)) return [];
  return list.filter(i => view.queryParam.filterAuthor !== i.name);
});
const unselectedTags = computed(() => {
  const list = aggregatesTags.value;
  if (!Array.isArray(list)) return [];
  return list.filter(i => view.queryParam.filterTag !== i.name);
});
const unselectedSeries = computed(() => {
  const list = aggregatesSeries.value;
  if (!Array.isArray(list)) return [];
  return list.filter(i => view.queryParam.filterSeries !== i.name);
});

// 已选中的过滤条件（在重置行平铺显示）
const selectedFilterChips = computed(() => {
  const chips = [];

  // 聚合: 作者
  if (view.queryParam.filterAuthor && Array.isArray(aggregatesAuthors.value)) {
    const item = aggregatesAuthors.value.find(i => i.name === view.queryParam.filterAuthor);
    if (item) chips.push({ key: 'author', label: `作者: ${item.name} (${item.cnt})`, color: 'indigo-5', onRemove: () => toggleAggFilter('filterAuthor', '') });
  }
  // 聚合: 标签
  if (view.queryParam.filterTag && Array.isArray(aggregatesTags.value)) {
    const item = aggregatesTags.value.find(i => i.name === view.queryParam.filterTag);
    if (item) chips.push({ key: 'tag', label: `标签: ${item.name} (${item.cnt})`, color: 'orange-5', onRemove: () => toggleAggFilter('filterTag', '') });
  }
  // 聚合: 系列
  if (view.queryParam.filterSeries && Array.isArray(aggregatesSeries.value)) {
    const item = aggregatesSeries.value.find(i => i.name === view.queryParam.filterSeries);
    if (item) chips.push({ key: 'series', label: `系列: ${item.name} (${item.cnt})`, color: 'green-5', onRemove: () => toggleAggFilter('filterSeries', '') });
  }
  // 大小
  if (view.queryParam.minSize > 0 || view.queryParam.maxSize > 0) {
    const minStr = view.queryParam.minSize > 0 ? formatFilterSize(view.queryParam.minSize) : '';
    const maxStr = view.queryParam.maxSize > 0 ? formatFilterSize(view.queryParam.maxSize) : '';
    let label = '';
    if (minStr && maxStr) label = `${minStr}~${maxStr}`;
    else if (minStr) label = `>${minStr}`;
    else label = `<${maxStr}`;
    chips.push({ key: 'size', label: `大小: ${label}`, color: 'blue-grey-5', onRemove: () => { view.queryParam.minSize = 0; view.queryParam.maxSize = 0; onFilterChange(); } });
  }
  // 日期
  if (view.queryParam.dateFrom || view.queryParam.dateTo) {
    const from = view.queryParam.dateFrom || '...';
    const to = view.queryParam.dateTo || '...';
    chips.push({ key: 'date', label: `日期: ${from}~${to}`, color: 'blue-grey-5', onRemove: () => { view.queryParam.dateFrom = ''; view.queryParam.dateTo = ''; onFilterChange(); } });
  }
  // 扩展名
  if (Array.isArray(view.queryParam.fileExts) && view.queryParam.fileExts.length > 0) {
    const label = view.queryParam.fileExts.join(', ');
    chips.push({ key: 'exts', label: `扩展名: ${label}`, color: 'blue-grey-5', onRemove: () => { view.queryParam.fileExts = []; onFilterChange(); } });
  }
  return chips;
});

const formatFilterSize = (bytes) => {
  if (bytes >= GB) return (bytes / GB).toFixed(1) + 'GB';
  if (bytes >= MB) return (bytes / MB).toFixed(0) + 'MB';
  if (bytes >= 1024) return Math.round(bytes / 1024) + 'KB';
  return bytes + 'B';
};

// 切换聚合过滤：点击选中/取消
const toggleAggFilter = (field, value) => {
  if (view.queryParam[field] === value) {
    view.queryParam[field] = '';
  } else {
    view.queryParam[field] = value;
  }
  view.queryParam.Page = 1;
  fetchSearch();
};

const currentSort = computed({
  get: () => `${view.queryParam.SortField}_${view.queryParam.SortType}`,
  set: (val) => {
    const [field, type] = val.split('_');
    view.queryParam.SortField = field;
    view.queryParam.SortType = type;
  }
});

const { copy } = useClipboard({ source: ref('') });

const systemProperty = useSystemProperty();
const suggestions = computed(() => {
  return systemProperty.getSuggestions;
});

const reg = /\w+[-_]\d+/;

const scrollTop = () => {
  const target = document.getElementsByClassName('scroll');
  if (target && target[2]) {
    target[2].scrollTo(0, 0, 500);
  }
};

// 排序后的搜索记录（不可变副本，避免模板中 sort() 修改原数组）
const sortedSearchRecords = computed(() => {
  const records = systemProperty.SearchRecords;
  if (!records || records.length === 0) return [];
  return [...records].sort((a, b) => b.createdAt - a.createdAt);
});

const themeStyle = computed(() => systemProperty.themeStyle);

onKeyStroke(['Enter'], () => {
  fetchSearch();
});

const btnSize = (position) => {
  if (isLarge.value) return '14px';
  if (isMedium.value && position === 'head') return '13px';
  return '12px';
};

const formatPlayTime = (timestamp) => {
  if (!timestamp) return '';
  const now = Date.now();
  const diff = now - timestamp;
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  const months = Math.floor(days / 30);
  const years = Math.floor(days / 365);

  if (years >= 1) {
    return `${years}年前`;
  }
  if (months >= 1) {
    return `${months}月前`;
  }
  if (days >= 1) {
    return `${days}天前`;
  }
  if (hours >= 1) {
    return `${hours}小时前`;
  }
  if (minutes >= 1) {
    return `${minutes}分钟前`;
  }
  return `${seconds}秒前`;
};

const searchByKeyword = (word) => {
  view.queryParam.Keyword = word;
  view.queryParam.Page = 1;
  fetchSearch();
};

const redirectUrl = (item) => {
  view.queryParam = { ...item };
  fetchSearch();
  return;
};
const openBatchEdit = () => {
  batchEditRef.value?.open({
    queryParam: view.queryParam,
    settingInfo: view.settingInfo,
    cb: (data) => { if (data?.settingInfo) view.settingInfo = data.settingInfo; },
  });
};

const playByPage = (item) => {
  systemProperty.savePlayTime(item.Id);
  const url = `#/playing/${item.Id}?a=refresh`;
  view.playBy = 'fullscreen';
  if ($q.platform.is.electron) {
    window.electron.createWindow({ router: url });
  } else {
    const opts = `width=${systemProperty.singleWindow.width},height=${systemProperty.singleWindow.height},titleBarStyle=`;
    window.open(url, 'player', opts);
  }
};

const searchCode = (item) => {
  let vcode = item.Code;
  if (!vcode) {
    return;
  }
  vcode = vcode.replace(/[\r\n\t]+/g, '');
  vcode = vcode.replace(/&nbsp;/g, '');
  vcode = vcode.trimEnd();
  let itemCode = vcode.match(reg);
  if (!itemCode) {
    return;
  }
  if (itemCode.indexOf('-C') > 0) {
    itemCode = itemCode.substring(0, itemCode.indexOf('-C'));
  }
  if (itemCode.indexOf('-') === 0) {
    itemCode = itemCode.substring(1);
  }
  if (itemCode.indexOf('@') >= 0) {
    itemCode = itemCode.substring(0, itemCode.indexOf('@'));
  }
  const url = `${view.settingInfo.BaseUrl}${itemCode}`;
  if ($q.platform.is.electron) {
    window.electron.createWindow({
      router: url,
      width: 1280,
      height: 1000,
      titleBarStyle: '',
    });
  } else {
    if (systemProperty.goSearchNewWidow) {
      window.open(url, '', 'width=1080,height=800,titleBarStyle=');
    } else {
      window.open(url);
    }
  }
};

const openFolder = (item) => {
  if ($q.platform.is.electron) {
    window.electron.showInFolder(item.Path);
  } else {
    commonExec(() => OpenFileFolder(item));
  }
};

const playBySystem = (item) => {
  systemProperty.savePlayTime(item.Id);
  view.playBy = 'system';
  if ($q.platform.is.electron) {
    window.electron.playMovie(item.Id);
  } else {
    commonExec(() => PlayMovie(item.Id));
  }
};

const confirmDelete = (item) => {
  $q.dialog({
    title: item.Name,
    message: '确定删除吗?',
    cancel: true,
    persistent: true,
  }).onOk(() => {
    commonExec(() => DeleteFile(item)).then(() => {
      $q.notify({ message: `已删除`, position: 'bottom-left' });
    });
  });
};

const qrDownloadVisible = ref(false);
const qrDownloadItem = ref(null);

const openQrDownload = (item) => {
  qrDownloadItem.value = item;
  qrDownloadVisible.value = true;
};

const fetchGetSettingInfo = async () => {
  const data = await GetSettingInfo();
  view.settingInfo = data.data;
  systemProperty.setSettingInfo(data.data);
  if (view.settingInfo.Pages && view.settingInfo.Pages.length > 0) {
    pageOptions.value = view.settingInfo.Pages.map((item) => {
      return Number(item);
    });
  }
};

const copyText = async (str) => {
  if (str && str.startsWith('-')) {
    str = str.substring(1);
  }
  await copy(str);
  $q.notify({ message: `${str}`, position: 'bottom-left' });
};

const goAuthor = (Author) => {
  if (!systemProperty.goAuthorNewWidow) {
    view.queryParam.Keyword = Author;
    fetchSearch();
  } else {
    const { Page, PageSize, MovieType, SortField, SortType } = view.queryParam;
    const routeData = resolve({
      path: '/search',
      query: {
        Page,
        PageSize,
        MovieType,
        SortField,
        SortType,
        Keyword: Author,
      },
    });
    window.open(routeData.href, '_blank');
  }
};

const picInPic = async (item, webFullScreen) => {
  if (!item) {
    return;
  }
  view.playBy = 'picInPic'
  systemProperty.savePlayTime(item.Id);
  view.currentDataInPlayer = item;
  videoRef.value.openVideo({
    item,
    queryParam: view.queryParam,
    webFullScreen,
  });
  const targetElement = document.getElementById(item.Id);
  const idx = getcurrentIndex(item);
  if (
    targetElement &&
    ((view.queryParam.PageSize == 10 && idx > 10) || idx > 12)
  ) {
    targetElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }
};

// 移除类型注释，因为当前不是 TypeScript 文件
const getcurrentIndex = (currentFile) => {
  let currentIndex;
  for (let index = 0; index < view.resultData.Data.length; index++) {
    const element = view.resultData.Data[index];
    if (element.Path === currentFile.Path) {
      currentIndex = index;
      break;
    }
  }
  return currentIndex;
};

const getNextFile = async (item) => {
  let currentIndex = getcurrentIndex(item);
  const targetIndex = currentIndex + 1;
  if (targetIndex <= view.resultData.Data.length - 1) {
    return view.resultData.Data[targetIndex];
  }
  if (view.queryParam.Page >= view.resultData.TotalPage) {
    // 如果当前页数大于总页数，则返回
    $q.notify({ type: 'negative', message: '已经是最后一页了' });
    return null;
  }
  if (
    systemProperty.searchPageAutoPullData &&
    targetIndex > view.resultData.Data.length - 1
  ) {
    // 如果开启自动拉取下一页数据，并且当前索引已经到达最后一个元素，则拉取下一页数据
    await pullNextPage();
    return getNextFile(item);
  }
  if (targetIndex > view.resultData.Data.length - 1) {
    await gotoPageNo(view.queryParam.Page + 1);
    return view.resultData.Data[0];
  }
  return null;
};

const viewNextOne = async (type) => {
  const currentData =
    type == 'play' ? view.currentDataInPlayer : view.currentDataInEditor;
  const nextItem = await getNextFile(currentData);
  if (nextItem) {
    if (type == 'play') {
      view.currentDataInPlayer = nextItem;
    } else {
      view.currentDataInEditor = nextItem;
    }
    if (type == 'cut') {
      fileCutImageRef.value.open(nextItem);
    } else if (type == 'edit') {
      openFileInfoRef(nextItem);
    } else if (type == 'info') {
      fileEditRef.value.open(nextItem);
    } else if (type == 'play') {
      picInPic(nextItem);
    }
  }
};
const viewPrevOne = async (type) => {
  const currentData =
    type == 'play' ? view.currentDataInPlayer : view.currentDataInEditor;
  const currentIndex = getcurrentIndex(currentData);
  let targetItem;
  const targetIndex = currentIndex - 1;
  if (targetIndex < 0) {
    if (view.queryParam.Page == 1) {
      $q.notify({ type: 'negative', message: '已经是第一页了' });
      return;
    }
    await gotoPageNo(view.queryParam.Page - 1);
    targetItem = view.resultData.Data[view.queryParam.PageSize - 1];
  } else {
    targetItem = view.resultData.Data[targetIndex];
  }
  if (targetItem) {
    if (type == 'play') {
      view.currentDataInPlayer = targetItem;
    } else {
      view.currentDataInEditor = targetItem;
    }
    if (type == 'cut') {
      fileCutImageRef.value.open(targetItem);
    } else if (type == 'edit') {
      openFileInfoRef(targetItem);
    } else if (type == 'info') {
      fileEditRef.value.open(targetItem);
    } else if (type == 'play') {
      picInPic(targetItem);
    }

    if (
      (currentIndex > 10 && view.queryParam.PageSize == 10) ||
      currentIndex > 12
    ) {
      const targetElement = document.getElementById(targetItem.Id);
      if (targetElement) {
        targetElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
    }
  }
};

const openFileInfoRef = (item, playing) => {
  view.currentDataInEditor = item;
  if (playing) {
    view.playBy = 'dialog';
  }
  fileInfoRef.value.open({ item, playing });
};

const pictureRightClick = async (item, e) => {
  if (item.MovieType === '无' || !item.MovieType) {
    view.currentDataInEditor = item;
    fileEditRef.value.open(item);
    e.returnValue = false;
  } else if (view.playBy === 'fullscreen') {
    playByPage(item);
    e.returnValue = false;
  } else if (view.playBy === 'system') {
    playBySystem(item);
    e.returnValue = false;
  } else if (view.playBy === 'dialog') {
    openFileInfoRef(item, true);
    e.returnValue = false;
  } else {
    picInPic(item);
    e.returnValue = false;
  }
};

const refreshDebounceFn = async (item, delayMs = 1000) => {
  await indexButton.value.refreshIndex(item);
};


const searchKeyword = async (keyword) => {
  view.queryParam.Keyword = keyword;
  await fetchSearch();
};

const selectNode = (nodeId) => {
  view.queryParam.SearchNode = nodeId;
  view.queryParam.Page = 1;
  view.searchNodeDisplay = nodeId ? (view.nodeList?.find(p => p.id === nodeId)?.name || nodeId) : '本机';
  fetchSearch();
};

const fetchNodeList = async () => {
  try {
    const res = await GetLanPeersWithStats();
    if (res) {
      const data = res.Data || res;
      view.localNodeName = data.localNodeName || '';
      view.nodeList = data.peers || [];
    }
  } catch (e) {
    console.error('获取节点列表失败', e);
  }
};

const gotoPageNo = async (no) => {
  if (no && no > 0) {
    view.queryParam.Page = Number(no);
  } else {
    view.queryParam.Page = 1;
  }
  scrollTop();
  await fetchSearch();
};

const gotoPage = ref(1);
const pageNoGoto = (no) => {
  gotoPageNo(Number(no));
};

const currentPageSizeChange = async (size) => {
  if (size) {
    view.queryParam.PageSize = Number(size);
  }
  await fetchSearch();
};

const keywordChange = () => {
  const { Keyword } = view.queryParam;
  if (Keyword && Keyword.length == 1) {
    return;
  }
  if (!Keyword || Keyword == '') {
    if (view.allPageNo > 0) {
      view.queryParam.Page = view.allPageNo;
    }
  }
  fetchSearch();
};

const pullNextPage = async (n) => {
  if (
    view.queryParam.Page < view.resultData.TotalPage &&
    !isMoreLoading.value
  ) {
    if (!n) {
      n = 1;
    }
    isMoreLoading.value = true;
    view.queryParam.Page = view.queryParam.Page + n;
    // 取消分页时也使用 abort controller, 复用当前 controller
    const signal = searchAbortController.value?.signal;
    try {
      const data = await SearchAPI(view.queryParam, signal);
      if (signal?.aborted) return;
      view.resultData.Data.push(...(data?.Data || []));
    } catch (e) {
      if (e?.name === 'CanceledError' || e?.name === 'AbortError') return;
      console.error('分页请求异常:', e);
    } finally {
      isMoreLoading.value = false;
    }
  }
};

const throttledOnIntersection = useDebounceFn(pullNextPage, 500);

const onIntersection = async (entry) => {
  if (
    entry.isIntersecting &&
    systemProperty.searchPageAutoPullData &&
    !isMoreLoading.value
  ) {
    throttledOnIntersection();
  }
};

const fetchSearch = async (replace = false) => {
  if (isFetching.value) return;
  isFetching.value = true;

  // 取消前一个搜索请求
  if (searchAbortController.value) {
    searchAbortController.value.abort();
  }
  const currentController = new AbortController();
  searchAbortController.value = currentController;

  try {
    saveParam(replace);
    const { Keyword } = view.queryParam;
    if (!Keyword || Keyword == '') {
      view.allPageNo = view.queryParam.Page;
    }
    const data = await SearchAPI(view.queryParam, currentController.signal);
    // 如果已被取消（新的请求已发出），丢弃旧结果
    if (currentController.signal.aborted) return;
    view.resultData = { ...data };
    const { ResultSize, ResultCnt } = data;
    document.title = `${Keyword || ''}  ${ResultSize} {${ResultCnt}}`;
  } catch (e) {
    if (e?.name === 'CanceledError' || e?.name === 'AbortError') return;
    console.error('搜索请求异常:', e);
    $q.notify({ type: 'negative', message: '请求失败' });
  } finally {
    isFetching.value = false;
  }
};

const moveThis = async () => {
  moveView.targetPathDialog = false;
  const updated = await commonExec(() =>
    MoveFile({
      Id: moveView.targetId,
      Path: moveView.targetPath,
      Title: moveView.targetName,
      Host: view.resultData.Data?.find((f) => f.Id === moveView.targetId)?.NodeHost || '',
    })
  );
  if (updated) {
    const item = view.resultData.Data?.find((f) => f.Id === moveView.targetId);
    if (item) Object.assign(item, updated);
  }
};

const setMovieType = async (Id, Type) => {
  const updated = await commonExec(() => ResetMovieType(Id, Type));
  if (updated) {
    const item = view.resultData.Data?.find((f) => f.Id === Id);
    if (item) Object.assign(item, updated);
  }
};

const thisRoute = useRoute();
const { resolve, push } = useRouter();

const saveParam = (skipPush = false) => {
  systemProperty.syncSearchParam(view.queryParam);
  systemProperty.expireTime = new Date().getTime() + 1000 * 60 * 60 * 2;
  localStorage.setItem('queryParam', JSON.stringify(view.queryParam));
  sessionStorage.setItem('isAuthenticated', 'true');
  // 避免频繁 push 导致组件重创建，仅在需要时更新 URL
  if (skipPush) return;
  const { Page, PageSize, MovieType, SortField, SortType, Keyword, SearchNode } =
    view.queryParam;
  const currentQuery = thisRoute.query;
  if (
    currentQuery.Keyword !== Keyword ||
    currentQuery.Page !== String(Page) ||
    currentQuery.PageSize !== String(PageSize) ||
    currentQuery.MovieType !== MovieType ||
    currentQuery.SortField !== SortField ||
    currentQuery.SortType !== SortType ||
    currentQuery.SearchNode !== SearchNode
  ) {
    push({
      path: '/search',
      query: {
        Page,
        PageSize,
        MovieType,
        SortField,
        SortType,
        Keyword,
        SearchNode,
      },
    });
  }
};

const gotoNextPage = () => {
  gotoPageNo(view.queryParam.Page + 1);
};
const gotoPrevPage = () => {
  gotoPageNo(view.queryParam.Page - 1);
};

provide('searchKeyword', searchKeyword);
provide('gotoNextPage', gotoNextPage);
provide('gotoPrevPage', gotoPrevPage);

// 初始加载完成后，才响应 IndexButton 的 refreshDone 事件
const onIndexRefresh = () => {
  console.log('onIndexRefresh');
};

onMounted(async () => {
  if ($q.platform.is.mobile) {
    systemProperty.showStyle = 'sm';
  }
  if ($q.platform.is.desktop && systemProperty.showStyle == 'sm') {
    systemProperty.showStyle = 'lg';
  }
  document.title = '搜索';
  systemProperty.PlayingMovie = {};
  const {
    Page,
    PageSize,
    MovieType,
    SortField,
    SortType,
    Keyword,
    SearchNode,
    showStyle,
    from,
  } = thisRoute.query;
  await fetchGetSettingInfo();
  // 获取节点列表（异步，不阻塞初始化）
  GetLanPeersWithStats().then(peerRes => {
    if (peerRes) {
      const data = peerRes.Data || peerRes;
      view.localNodeName = data.localNodeName || '';
      view.nodeList = data.peers || [];
    }
  }).catch(() => { /* 忽略 */ });
  // 恢复 URL 中的节点选择（先设值，节点列表稍后异步到达后会更新显示）
  if (SearchNode) {
    view.queryParam.SearchNode = SearchNode;
    view.searchNodeDisplay = SearchNode;
  }
  if (Keyword) {
    view.queryParam.Keyword = Keyword;
  }
  if (Page && PageSize) {
    view.queryParam.Page = Number(Page);
    view.queryParam.PageSize = Number(PageSize);
    view.queryParam.MovieType = MovieType;
    view.queryParam.SortField = SortField;
    view.queryParam.SortType = SortType;
    view.queryParam.Keyword = Keyword;
    view.queryParam.SearchNode = SearchNode || '';
    view.queryParam.showStyle = showStyle;
  } else {
    if (from === 'index') {
      const piniaParam = systemProperty.FileSearchParam;
      if (piniaParam) {
        view.queryParam = piniaParam;
      }
    } else {
      const storage = JSON.parse(localStorage.getItem('queryParam'));
      if (storage) {
        view.queryParam = storage;
      }
    }
  }
  // 恢复高级过滤 UI 状态（从 localStorage/pinia 恢复的 queryParam 中同步）
  ensureFilterDefaults();
  fetchSearch(true);  // 异步执行
});

onUnmounted(() => {
  clearTimeout(sseDebounceTimer);
});
</script>

<style lang="scss" scoped>
// 隐藏滚动条
.scrollRef::-webkit-scrollbar {
  display: none;
}

.scrollRef {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

// 统一标签样式
.q-chip {
  border-radius: 6px;
  transition: all 0.3s ease;
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);

  &:hover {
    transform: translateY(-1px);
    background: rgba(0, 0, 0, 1);
  }
}

.card-top-tag {
  position: absolute;
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
  flex-direction: column;
  align-items: baseline;
  max-height: 200px;
  width: auto;
  z-index: 2;
}

.card-top-type {
  right: 0;
  position: absolute;
  width: 3.2rem;
  display: flex;
  flex-direction: column;
  z-index: 2;
}

.large-result {
  padding: 0px;
  width: 220px;
  height: 386px;
  overflow: hidden;
}

.large-result-image {
  width: 100%;
  height: 100%;

  &::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 20%;
    background: linear-gradient(180deg, rgba(0, 0, 0, 0), rgba(0, 0, 0, 0.3));
  }
}

.medium-result {
  padding: 0px;
  width: 224px;
  height: 198px;
  overflow: hidden;
}

.medium-result-image {
  width: 100%;
  height: 100%;

  &::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 24%;
    background: linear-gradient(180deg, rgba(0, 0, 0, 0), rgba(0, 0, 0, 0.3));
  }
}

.small-result {
  padding: 1px;
  width: calc((100% - 30px) / 2);
  height: 240px;
  overflow: hidden;
  align-items: center;
  justify-content: center;
}

.small-result-image {
  width: 100%;
  height: 100%;

  &::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 25%;
    background: linear-gradient(180deg, rgba(0, 0, 0, 0), rgba(0, 0, 0, 0.3));
  }
}

.float-btn {
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);

  .btn-row {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    scrollbar-width: 1px;
    scrollbar-color: transparent transparent;
  }

  .btn-row-responsive {
    overflow: hidden;
    gap: 2px;

    .q-btn:last-child {
      flex-shrink: 0;
    }
  }

  .content-row {
    display: -webkit-box;
    /* 将对象作为弹性伸缩盒子模型显示 */
    -webkit-box-orient: vertical;
    /* 设置子元素的排列方式为垂直方向 */
    line-clamp: 3;
    /* 设置显示的行数 */
    overflow: hidden;
    /* 隐藏溢出文本 */
    text-overflow: ellipsis;
    /* 显示省略号 */
    padding: 2px;
    line-height: 1.2 !important;
  }

  a {
    border-radius: 2px;
    transition: background 0.3s ease;

    &:hover {
      background: rgba(255, 255, 255, 0.8);
    }
  }
}

.chip-tag {
  margin-left: 0;
  padding: 0 2px;
  font-weight: 500;
  width: fit-content;
  color: orangered;
  background-color: rgba(250, 250, 250, 0.4);
}

.page-sticky {
  background-color: rgba(0, 0, 0, 0.8);
}

// 分页器样式
.q-pagination {
  &__button {
    border-radius: 6px !important;
    margin: 0 3px;
  }
}

// 调整卡片内文字的颜色和字体
.q-card p {
  color: #333;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
}

// 暗黑模式适配
.body--dark {
  .q-card {
    background: rgba(40, 40, 40, 0.9);
  }

  .q-chip {
    background: rgba(50, 50, 50, 0.9);
  }

  .q-img {
    background: linear-gradient(45deg, #2d2d2d, #1a1a1a);

    &::after {
      background: linear-gradient(180deg,
          rgba(0, 0, 0, 0),
          rgba(255, 255, 255, 0.1));
    }

    .q-img__error {
      background: rgba(255, 255, 255, 0.05);
    }
  }

  // 自然主题下图片占位色
  &.theme-natural .q-img {
    background: linear-gradient(45deg, #e2e8f0, #cbd5e1);

    .q-img__error {
      background: rgba(148, 163, 184, 0.3);

      img {
        filter: invert(0.8) brightness(0.8);
      }
    }
  }
}

// 输入框聚焦效果
.q-input {
  transition: box-shadow 0.3s ease;

  &:focus-within {
    box-shadow: 0 0 0 2px rgba(255, 165, 0, 0.3);
  }
}

.q-btn {
  transition: all 0.2s ease;

  &:hover {
    transform: scale(1.08);
    filter: brightness(110%);
  }

  &--rounded {
    margin: 2px;
  }
}

:deep(.q-item) {
  min-height: 40px;
  transition: background 0.2s ease;

  &:hover {
    background: var(--q-menu-hover);
  }

  &.q-item--active {
    background: var(--q-menu-active);
  }
}

:deep(.q-item__label--header) {
  font-size: 0.7rem;
  line-height: 1.5;
  letter-spacing: 0.5px;
}

// 搜索结果卡片增强样式
.search-result-card {
  border: grey 1px solid;
  border-radius: 12px !important;
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.8);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1) !important;
}

// 高级过滤面板 — 悬浮卡片样式
.advanced-filter-panel {
  position: fixed;
  top: 64px;
  left: 6px;
  right: 6px;
  background: rgba(255, 255, 255, 0.96);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 12px;
  box-shadow:
    0 12px 48px rgba(0, 0, 0, 0.15),
    0 4px 12px rgba(0, 0, 0, 0.08),
    inset 0 1px 0 rgba(255, 255, 255, 0.7);
  z-index: 99;
  transition: transform 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow:
      0 16px 56px rgba(0, 0, 0, 0.18),
      0 6px 16px rgba(0, 0, 0, 0.1),
      inset 0 1px 0 rgba(255, 255, 255, 0.7);
  }

  .q-input,
  .q-select {
    font-size: 0.9rem;
  }

  .text-caption {
    font-size: 0.8rem;
    font-weight: 600;
    letter-spacing: 0.3px;
  }
}

// 暗黑模式过滤面板
.body--dark .advanced-filter-panel {
  background: rgba(30, 30, 30, 0.96);
  border-color: rgba(255, 255, 255, 0.06);
  box-shadow:
    0 12px 48px rgba(0, 0, 0, 0.35),
    0 4px 12px rgba(0, 0, 0, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.05);

  &:hover {
    box-shadow:
      0 16px 56px rgba(0, 0, 0, 0.4),
      0 6px 16px rgba(0, 0, 0, 0.25),
      inset 0 1px 0 rgba(255, 255, 255, 0.05);
  }
}
</style>
