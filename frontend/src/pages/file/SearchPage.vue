<template>
  <q-layout
    view="lHh lpr lFf"
    container
    style="height: 93vh"
    class="shadow-2 rounded-borders"
    :style="themeStyle"
  >
    <!-- 头部 -->
    <q-header
      :style="themeStyle"
      elevated
      class="q-gutter-sm flex justify-center"
      style="
        backdrop-filter: blur(5px);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      "
    >
      <!-- 索引按钮 -->
      <IndexButton
        ref="indexButton"
        @refresh-done="fetchSearch"
        glossy
        dense
        :size="btnSize('head')"
      />

      <!-- 重命名按钮 -->
      <q-btn
        :loading="view.renameCount > 0"
        v-if="view.renameCount > 0"
        class="q-mt-sm"
        color="red"
        :size="btnSize('head')"
        dense
        :label="`重命名 (${view.renameCount})`"
      >
        <template v-slot:loading>
          <q-spinner-facebook size="xs"></q-spinner-facebook>
          {{ `r:${view.renameCount}` }}
        </template>
      </q-btn>
      <q-btn
        icon="ti-bar-chart-alt"
        class="cursor-pointer"
        dense
        :size="btnSize('head')"
        color="red"
        flat
      >
        <DataPop url />
      </q-btn>
      <!-- 排序字段选择 -->
      <q-btn-toggle
        v-if="!isSmall"
        :size="btnSize('head')"
        glossy
        v-model="view.queryParam.SortField"
        @update:model-value="fetchSearch()"
        :options="FieldEnum"
      />
      <!-- 移动端排序字段选择-->
      <q-btn-dropdown
        glossy
        v-if="isSmall"
        color="primary"
        :size="btnSize('head')"
        style="width: 4rem"
        :label="getLabelByValue(view.queryParam.SortField, FieldEnum)"
      >
        <q-list>
          <q-item
            v-for="item in FieldEnum"
            :key="item.label"
            clickable
            v-close-popup
            @click="
              view.queryParam.SortField = item.value;
              fetchSearch();
            "
          >
            <q-item-section>
              <q-item-label>{{ item.label }}</q-item-label>
            </q-item-section>
          </q-item>
        </q-list>
      </q-btn-dropdown>
      <!-- 移动端排序类型选择  -->
      <q-btn-dropdown
        glossy
        color="primary"
        dense
        :size="btnSize('head')"
        style="width: 3.5rem"
        :label="getLabelByValue(view.queryParam.SortType, DescEnum)"
      >
        <q-list>
          <q-item
            v-for="item in DescEnum"
            :key="item.label"
            clickable
            v-close-popup
            @click="
              view.queryParam.SortType = item.value;
              fetchSearch();
            "
          >
            <q-item-section>
              <q-item-label>{{ item.label }}</q-item-label>
            </q-item-section>
          </q-item>
        </q-list>
      </q-btn-dropdown>

      <!-- 电影类型选择   style="width: 26rem" -->
      <q-btn-toggle
        v-if="!isSmall"
        glossy
        push
        ripple
        stack
        :size="btnSize('head')"
        stretch
        v-model="view.queryParam.MovieType"
        @update:model-value="fetchSearch()"
        :options="MovieTypeSelects"
      />

      <!-- 移动端电影类型选择 -->
      <q-btn-dropdown
        v-if="isSmall"
        glossy
        push
        ripple
        color="primary"
        :label="getLabelByValue(view.queryParam.MovieType, MovieTypeSelects)"
      >
        <q-list>
          <q-item
            v-for="item in MovieTypeSelects"
            :key="item.label"
            clickable
            v-close-popup
            @click="
              view.queryParam.MovieType = item.value;
              fetchSearch();
            "
          >
            <q-item-section>
              <q-item-label>{{ item.label }}</q-item-label>
            </q-item-section>
          </q-item>
        </q-list>
      </q-btn-dropdown>

      <!-- 搜索框 -->
      <q-input
        dense
        type="search"
        style="
          max-width: 350px;
          border-radius: 4px;
          box-shadow: 0 0 4px rgba(0, 0, 0, 0.1);
        "
        outlined
        glossy
        :debounce="1000"
        id="searchBtn"
        label="..."
        v-model="view.queryParam.Keyword"
        filled
        clearable
        @clear="keywordChange"
        @update:model-value="keywordChange"
      >
        <template v-slot:prepend>
          <q-icon name="ti-list" class="cursor-pointer">
            <q-popup-proxy>
              <div style="width: 200px; max-height: 50vh">
                <q-list>
                  <q-item
                    clickable
                    v-ripple
                    v-for="word in suggestions"
                    :key="word"
                    @click="
                      view.queryParam.Keyword = word;
                      fetchSearch();
                    "
                  >
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
          <q-icon
            name="ti-search"
            title="搜"
            glossy
            class="cursor-pointer"
            @click="fetchSearch"
          >
          </q-icon>
        </template>
      </q-input>

      <!-- 仅重复项选择 -->
      <q-checkbox
        v-model="view.queryParam.OnlyRepeat"
        checked-icon="task_alt"
        unchecked-icon="ti-help"
        :val="true"
        flat
        dense
        @update:model-value="fetchSearch"
      >
        <q-tooltip class="bg-white text-primary"> 去重 </q-tooltip>
      </q-checkbox>
      <span
        v-if="isLarge || isMedium"
        style="align-items: center; align-content: center"
      >
        {{ view.resultShow }}
      </span>
      <!-- 主题选择器 -->
      <q-btn-dropdown
        flat
        dense
        no-caps
        glossy
        class="theme-selector-btn"
      >
        <template v-slot:label>
          <div class="row items-center q-gutter-xs">
            <q-icon :name="themeIcon" size="18px" />
            <span class="text-caption gt-xs">{{ currentThemeLabel }}</span>
            <q-icon name="arrow_drop_down" size="16px" class="q-ml-xs text-grey-5" />
          </div>
        </template>
        <q-list style="min-width: 160px; padding: 8px 0;">
          <!-- 主题选择 -->
          <div class="q-px-md q-py-xs">
            <q-item-label header class="text-grey-5 text-xs font-medium">主题外观</q-item-label>
          </div>
          <q-item
            clickable
            v-close-popup
            @click="setTheme('star')"
            :active="systemProperty.theme === 'star'"
            class="q-mx-sm rounded-lg"
          >
            <q-item-section avatar>
              <div class="w-8 h-8 rounded-lg flex items-center justify-center" style="background: linear-gradient(135deg, #4f46e5, #7c3aed);">
                <q-icon name="star" color="white" size="16px" />
              </div>
            </q-item-section>
            <q-item-section>
              <q-item-label class="font-medium">星空主题</q-item-label>
              <q-item-label caption class="text-xs">深蓝紫配色</q-item-label>
            </q-item-section>
            <q-item-section side v-if="systemProperty.theme === 'star'">
              <q-icon name="check" color="primary" size="18px" />
            </q-item-section>
          </q-item>
          <q-item
            clickable
            v-close-popup
            @click="setTheme('natural')"
            :active="systemProperty.theme === 'natural'"
            class="q-mx-sm rounded-lg"
          >
            <q-item-section avatar>
              <div class="w-8 h-8 rounded-lg flex items-center justify-center" style="background: linear-gradient(135deg, #22c55e, #84cc16);">
                <q-icon name="eco" color="white" size="16px" />
              </div>
            </q-item-section>
            <q-item-section>
              <q-item-label class="font-medium">自然主题</q-item-label>
              <q-item-label caption class="text-xs">温暖米色绿植</q-item-label>
            </q-item-section>
            <q-item-section side v-if="systemProperty.theme === 'natural'">
              <q-icon name="check" color="primary" size="18px" />
            </q-item-section>
          </q-item>
          <!-- 分隔线 -->
          <q-separator class="my-2" />
          <!-- 显示模式 -->
          <div class="q-px-md q-py-xs">
            <q-item-label header class="text-grey-5 text-xs font-medium">显示模式</q-item-label>
          </div>
          <q-item
            clickable
            v-close-popup
            @click="systemProperty.showImage = 'cover'"
            :active="systemProperty.showImage === 'cover'"
            class="q-mx-sm rounded-lg"
          >
            <q-item-section avatar>
              <div class="w-8 h-8 rounded-lg flex items-center justify-center bg-primary/10">
                <q-icon name="image" color="primary" size="16px" />
              </div>
            </q-item-section>
            <q-item-section>
              <q-item-label class="font-medium">封面模式</q-item-label>
              <q-item-label caption class="text-xs">展示完整封面图</q-item-label>
            </q-item-section>
            <q-item-section side v-if="systemProperty.showImage === 'cover'">
              <q-icon name="check" color="primary" size="18px" />
            </q-item-section>
          </q-item>
          <q-item
            clickable
            v-close-popup
            @click="systemProperty.showImage = 'poster'"
            :active="systemProperty.showImage === 'poster'"
            class="q-mx-sm rounded-lg"
          >
            <q-item-section avatar>
              <div class="w-8 h-8 rounded-lg flex items-center justify-center bg-primary/10">
                <q-icon name="movie" color="primary" size="16px" />
              </div>
            </q-item-section>
            <q-item-section>
              <q-item-label class="font-medium">海报模式</q-item-label>
              <q-item-label caption class="text-xs">展示电影海报</q-item-label>
            </q-item-section>
            <q-item-section side v-if="systemProperty.showImage === 'poster'">
              <q-icon name="check" color="primary" size="18px" />
            </q-item-section>
          </q-item>
          <!-- 分隔线 -->
          <q-separator class="my-2" />
          <!-- 卡片大小 -->
          <div class="q-px-md q-py-xs">
            <q-item-label header class="text-grey-5 text-xs font-medium">卡片大小</q-item-label>
          </div>
          <q-item
            clickable
            v-close-popup
            @click="systemProperty.showStyle = 'lg'"
            :active="systemProperty.showStyle === 'lg'"
            class="q-mx-sm rounded-lg"
          >
            <q-item-section avatar>
              <div class="w-8 h-8 rounded-lg flex items-center justify-center bg-primary/10">
                <q-icon name="view_module" color="primary" size="16px" />
              </div>
            </q-item-section>
            <q-item-section>
              <q-item-label class="font-medium">大</q-item-label>
              <q-item-label caption class="text-xs">大尺寸卡片</q-item-label>
            </q-item-section>
            <q-item-section side v-if="systemProperty.showStyle === 'lg'">
              <q-icon name="check" color="primary" size="18px" />
            </q-item-section>
          </q-item>
          <q-item
            clickable
            v-close-popup
            @click="systemProperty.showStyle = 'md'"
            :active="systemProperty.showStyle === 'md'"
            class="q-mx-sm rounded-lg"
          >
            <q-item-section avatar>
              <div class="w-8 h-8 rounded-lg flex items-center justify-center bg-primary/10">
                <q-icon name="grid_view" color="primary" size="16px" />
              </div>
            </q-item-section>
            <q-item-section>
              <q-item-label class="font-medium">中</q-item-label>
              <q-item-label caption class="text-xs">中等尺寸卡片</q-item-label>
            </q-item-section>
            <q-item-section side v-if="systemProperty.showStyle === 'md'">
              <q-icon name="check" color="primary" size="18px" />
            </q-item-section>
          </q-item>
          <q-item
            clickable
            v-close-popup
            @click="systemProperty.showStyle = 'sm'"
            :active="systemProperty.showStyle === 'sm'"
            class="q-mx-sm rounded-lg"
          >
            <q-item-section avatar>
              <div class="w-8 h-8 rounded-lg flex items-center justify-center bg-primary/10">
                <q-icon name="apps" color="primary" size="16px" />
              </div>
            </q-item-section>
            <q-item-section>
              <q-item-label class="font-medium">小</q-item-label>
              <q-item-label caption class="text-xs">紧凑尺寸卡片</q-item-label>
            </q-item-section>
            <q-item-section side v-if="systemProperty.showStyle === 'sm'">
              <q-icon name="check" color="primary" size="18px" />
            </q-item-section>
          </q-item>
        </q-list>
      </q-btn-dropdown>
      <!-- 设置按钮 -->
      <q-fab
        icon="ti-pencil-alt"
        direction="left"
        :color="view.runningTaskCount > 0 ? 'red' : 'orange'"
        glossy
        style="position: fixed; right: 10px; top: 60px"
      >
        <q-fab-action
          @click="openListEditRef('filelist')"
          color="primary"
          label="编辑"
        />
        <q-fab-action
          @click="openListEditRef('tasking')"
          color="primary"
          label="任务"
        />
        <q-fab-action
          @click="openListEditRef('setting')"
          color="primary"
          label="主题"
        />
        <q-fab-action
          @click="openListEditRef('history')"
          color="primary"
          label="历史"
        />
      </q-fab>
    </q-header>
    <!-- 底部 -->
    <q-footer elevated :style="themeStyle" class="glossy">
      <div class="flex flex-center">
              <!-- 页码输入框 -->
        <q-btn
          icon="settings"
          color="orange"
          flat
          dense
          @mouseenter="view.pageSetting = true"
        >
          <q-popup-proxy
            v-model="view.pageSetting"
            style="background: rgba(250, 250, 250, 0.8)"
          >
            <div
              class="q-gutter-md"
              style="
                width: 18rem;
                height: 8rem;
                display: flex;
                flex-direction: column;
                justify-content: space-evenly;
              "
            >
              
              <div class="row justify-between">
                <q-btn flat dense> 每页大小 </q-btn>
                <q-select
                  size="sm"
                  dense
                  flat
                  @update:model-value="currentPageSizeChange"
                  filled
                  bgColor="orange"
                  style="text-align: center; width: 40%"
                  v-model="view.queryParam.PageSize"
                  :options="pageOptions"
                >
                </q-select>
              </div>
              <div class="row justify-between">
                <q-btn flat dense>页码 </q-btn>
                <q-input
                  v-model="gotoPage"
                  :dense="true"
                  style="text-align: center; width: 40%"
                  bgColor="orange"
                  :max="view.resultData.TotalPage"
                  :min="1"
                  :debounce="1000"
                  @focus="focusEvent($event)"
                  @update:model-value="pageNoGoto"
                />
              </div>
            </div>
            <!-- 每页数量选择 -->
          </q-popup-proxy>
        </q-btn>
        <!-- 分页器 -->
        <q-pagination
          v-model="view.queryParam.Page"
          @update:model-value="gotoPageNo"
          color="deep-orange"
          :ellipses="true"
          :max="view.resultData.TotalPage || 0"
          :max-pages="isSmall ? 5 : 10"
          boundary-numbers
          direction-links
        ></q-pagination>

      </div>
      <div style="position: fixed; right: 10px; bottom: 40px">
        <q-btn icon="history" color="blue" glossy>
          <q-popup-proxy
            v-model="view.showHistory"
            style="background: rgba(250, 250, 250, 0.9); width: 400px"
          >
            <div>
              <span ripple flat
                >搜索记录
                <q-btn
                  ripple
                  flat
                  color="red"
                  @click="systemProperty.SearchRecords = []"
                  >清空</q-btn
                ></span
              >
              <q-list bordered separator style="height: 50vh; overflow: auto">
                <div
                  v-for="(his, idx) in systemProperty.SearchRecords.sort(
                    (a, b) => {
                      return b.createdAt - a.createdAt;
                    }
                  )"
                  :key="idx"
                >
                  <div
                    class="row justify-between cursor-pointer"
                    style="margin: 4px; padding: 4px; color: blue"
                    ripple
                    v-close-popup
                    align="left"
                    @click="redirectUrl(his)"
                  >
                    <div style="float: left">
                      {{
                        `${his.Page} -${his.PageSize} -${
                          getLabelByValue(his.MovieType, MovieTypeOptions) ||
                          '全部'
                        }-${getLabelByValue(
                          his.SortField,
                          FieldEnum
                        )} -${getLabelByValue(his.SortType, DescEnum)} `
                      }}
                    </div>
                    <div style="float: right">
                      {{ his.Keyword == null ? '无' : his.Keyword }} -
                      {{ date.formatDate(his.createdAt, 'HH:mm') }}
                    </div>
                  </div>
                </div></q-list
              >
            </div>
          </q-popup-proxy>
        </q-btn>
      </div>
    </q-footer>
    <!-- 页面内容 -->
    <q-page-container class="scrollRef">
      <q-page>
        <div class="row q-gutter-sm justify-start q-pl-sm" v-if="view.resultData.Data && view.resultData.Data.length > 0">
          <!-- 卡片列表 -->
          <transition-group name="card-list">
            <q-card
              v-for="item in view.resultData.Data"
              :key="item.Id"
              :id="item.Id"
              once
              v-bind:class="{
                'large-result': isLarge,
                'medium-result': isMedium,
                'small-result': isSmall,
              }"
              style="
                border-radius: 8px;
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.6);
                transition: all 0.3s ease-out;
                border: 1px solid var(--q-border);
                background: var(--q-bg-card);
              "
            :style="{
              backgroundColor:
                item.Id == view.currentDataInPlayer.Id
                  ? 'rgba(0, 0, 0, 0.3)'
                  : item.Id == view.currentDataInEditor.Id
                  ? 'rgba(0, 0, 0, 0.5)'
                  : '',
            }"
          >
            <div class="card-top-tag" style="width: 80%">
              <!-- 种草按钮 -->
              <q-btn
                text-color="white"
                color="red"
                :size="btnSize('top')"
                dense
                class="glossy"
                style="max-width: 5rem"
                @contextmenu="
                  (e) => {
                    refreshDebounceFn(item);
                    e.returnValue = false;
                  }
                "
              >
                <q-popup-proxy
                  style="background-color: rgba(250, 250, 250, 0.9)"
                >
                  <TagPop
                    :currentData="item"
                    :current-tag="item.Tags"
                    :delay="10"
                  />
                </q-popup-proxy>
                <span>{{ `种草/${item.PageNo}` }}</span>
              </q-btn>

              <!-- 标签列表 -->

              <q-chip
                square
                dense
                v-for="tag in item.Tags"
                :key="tag"
                :size="btnSize('top')"
                class="chip-tag"
              >
                <span @click="searchKeyword(tag)">{{
                  tag?.substring(0, 4)
                }}</span>
              </q-chip>
            </div>
            <div class="card-top-type" style="align-items: flex-end">
              <!-- 电影类型选择按钮 -->
              <q-btn
                dense
                :size="btnSize('top')"
                class="glossy"
                color="primary"
                :label="`${item.MovieType === '无' ? `分类 ` : item.MovieType}`"
              >
                <q-menu>
                  <q-list style="min-width: 68px">
                    <q-item
                      v-for="mt in MovieTypeOptions"
                      :key="mt.value"
                      clickable
                      v-close-popup
                    >
                      <q-item-section
                        @click="
                          setMovieType(item.Id, mt.value);
                          item.btnMovieType = false;
                        "
                        >{{ mt.label }}</q-item-section
                      >
                    </q-item>
                    <q-item clickable v-close-popup>
                      <q-item-section
                        style="color: blue"
                        @click="refreshDebounceFn(item)"
                        >刷新</q-item-section
                      >
                    </q-item>
                  </q-list>
                </q-menu>
              </q-btn>
              <q-btn
                dense
                glossy
                color="grey"
                size="sm"
                style="margin-top: 4px"
                v-if="formatSeries(item.Code)"
              >
                <span @click="searchKeyword(formatSeries(item.Code))">{{
                  formatSeries(item.Code).substring(0, 4)
                }}</span>
              </q-btn>
              <q-btn
                dense
                flat
                text-color="green"
                size="sm"
                style="margin-top: 4px"
                v-if="systemProperty.getPlayTime(item.Id)"
              >
                <span>{{ formatPlayTime(systemProperty.getPlayTime(item.Id)) }}</span>
              </q-btn>
              <!-- 文件类型标签 -->
              <q-chip
                square
                v-if="item.FileType != 'mp4'"
                :size="btnSize('top')"
                dense
                color="orange"
              >
                <span @click="searchKeyword(item.FileType)">
                  {{ item.FileType }}</span
                >
              </q-chip>
            </div>
            <!-- 图片 -->
            <q-img
              fit="fill"
              lazy="true"
              :lazy-src="getBlurImage(item.Id)"
              :class="{
                'large-result-image': isLarge,
                'medium-result-image': isMedium,
                'small-result-image': isSmall,
              }"
              :src="getImage(item.Id)"
              @contextmenu="(e) => pictureRightClick(item, e)"
              @click="openFileInfoRef(item)"
              style="
                border-radius: 6px 6px 0 0;
                background: linear-gradient(135deg, rgba(30, 30, 50, 0.8), rgba(15, 15, 26, 0.9));
                transition: opacity 0.4s ease-in-out, transform 0.3s ease;
                overflow: hidden;
                will-change: opacity;
                aspect-ratio: 2/3;
              "
              loading="lazy"
            >
              <template v-slot:loading>
                <q-spinner-ios color="white" size="2em"
                  >Loading...</q-spinner-ios
                >
              </template>
              <template v-slot:error>
                <!-- 图片加载失败时显示的占位图 -->
                <div class="text-subtitle1 text-white">
                  <q-icon name="image_not_supported" size="2em"></q-icon>
                </div>
              </template>
              <q-inner-loading
                :showing="item.Id == view.currentDataInEditor.Id"
              >
                <q-spinner-gears size="80px" color="primary" label="编辑中" />
              </q-inner-loading>
              <q-inner-loading
                :showing="item.Id == view.currentDataInPlayer.Id"
                label="播放中"
                label-class="text-teal"
                label-style="font-size: 1.1em"
              />
            </q-img>
            <div
              class="absolute-bottom float-btn"
              style="background-color: rgba(0, 0, 0, 0.3)"
            >
              <div>
                <div class="btn-row">
                  <!-- 播放按钮 -->
                  <q-btn
                    round
                    ripple
                    flat
                    glossy
                    color="white  "
                    :size="btnSize('footer')"
                    icon="play_circle_outline"
                    @click="playBySystem(item)"
                    title="播放"
                    v-if="showButton('播放') && !isSmall"
                  />
                  <!-- 单页播放按钮 -->
                  <q-btn
                    round
                    flat
                    ripple
                    glossy
                    color="yellow"
                    :size="btnSize('footer')"
                    icon="fullscreen"
                    title="单页播放"
                    @click="playByPage(item)"
                  />
                  <!-- 小播放按钮 -->
                  <q-btn
                    round
                    flat
                    ripple
                    glossy
                    color="blue"
                    :size="btnSize('footer')"
                    icon="tv"
                    @click="openFileInfoRef(item, true)"
                    title="小播放"
                  />
                  <!-- 画中画按钮 -->
                  <q-btn
                    round
                    flat
                    ripple
                    glossy
                    color="white"
                    :size="btnSize('footer')"
                    icon="picture_in_picture"
                    @click="picInPic(item)"
                    @contextmenu="
                      (e) => {
                        picInPic(item, true);
                        e.returnValue = false;
                      }
                    "
                    title="画中画"
                  />
                </div>
                <div class="btn-row">
                  <!-- 编辑按钮 -->
                  <q-btn
                    round
                    ripple
                    glossy
                    :size="btnSize('footer')"
                    color="grey-8"
                    icon="edit"
                    @click="
                      view.currentDataInEditor = item;
                      fileEditRef.open(item);
                    "
                    v-if="showButton('编辑')"
                    title="编辑"
                    style="box-shadow: 0 2px 6px rgba(128, 128, 128, 0.2)"
                  />
                  <!-- 文件夹按钮 -->
                  <q-btn
                    round
                    ripple
                    glossy
                    :size="btnSize('footer')"
                    color="primary"
                    icon="open_in_new"
                    @click="openFolder(item)"
                    v-if="showButton('文件夹') && !isSmall"
                    title="文件夹"
                  />
                  <!-- 网搜按钮 -->
                  <q-btn
                    round
                    ripple
                    glossy
                    :size="btnSize('footer')"
                    color="brown-5"
                    icon="ti-search"
                    title="网搜"
                    @click="searchCode(item)"
                  />
                  <!-- 删除按钮 -->
                  <q-btn
                    round
                    ripple
                    glossy
                    :size="btnSize('footer')"
                    color="red-7"
                    text-color="black"
                    icon="delete"
                    @click="confirmDelete(item)"
                    v-if="showButton('删除')"
                    style="box-shadow: 0 2px 6px rgba(255, 50, 50, 0.3)"
                    title="删除"
                  />
                  <!-- 截图按钮 -->
                  <q-btn
                    round
                    ripple
                    glossy
                    :size="btnSize('footer')"
                    color="black"
                    @click="
                      () => {
                        view.currentDataInEditor = item;
                        fileCutImageRef.open(item);
                      }
                    "
                    icon="ti-cut"
                    title="截图"
                  />
                  <!-- 移动按钮 -->
                  <q-btn
                    round
                    ripple
                    glossy
                    :size="btnSize('footer')"
                    color="black"
                    @click="
                      () => {
                        moveView.targetId = item.Id;
                        moveView.targetName = item.Title;
                        if (moveView.targetPath == '') {
                          moveView.targetPath = item.DirPath;
                        }
                        moveView.originPath = item.DirPath;
                        moveView.targetPathDialog = true;
                      }
                    "
                    icon="ti-location-arrow"
                    v-if="showButton('移动')"
                    title="移动"
                  />
                </div>
              </div>

              <div
                class="content-row"
                :style="{
                  height: isLarge ? '51px' : '34px',
                  fontSize: isLarge ? '14px' : '14px',
                  backgroundColor: 'rgba(250, 250, 250,0.8)',
                }"
              >
                <!-- backgroundColor:
                    item.Id == view.currentDataInPlayer.Id
                      ? 'rgba(0, 0, 0, 0.2)'
                      : item.Id == view.currentDataInEditor.Id
                      ? 'rgba(0, 0, 0, 0.5)'
                      : 'rgba(250, 250, 250,0.8)', -->
                <!-- 演员、编号、大小信息 -->
                <span
                  style="
                    color: green;
                    margin-right: 1px;
                    background-color: rgba(0, 0, 0, 0.1);
                  "
                  class="cursor-pointer"
                  >{{ getTimeAgo(item.MTime) }}
                  <q-popup-proxy>
                    <div
                      style="
                        width: 400px;
                        padding: 10px;
                        background-color: rgba(0, 0, 0, 0.1);
                      "
                    >
                      <div>
                        <span
                          style="
                            color: rgb(161, 100, 19);
                            background-color: rgba(0, 0, 0, 0.1);
                            margin-right: 1px;
                          "
                          class="cursor-pointer"
                          @click="copyText(item.Actress)"
                          >{{ item.Actress }}</span
                        >
                      </div>
                      <div>
                        <span
                          style="color: rgb(239, 30, 30)"
                          class="cursor-pointer"
                          @click="copyText(item.Code)"
                          >{{ item.Code }}</span
                        >
                      </div>
                      <div>
                        {{ formatTitle(item.Title) }}
                      </div>
                      <div
                        style="color: green"
                        class="cursor-pointer"
                        @click="searchKeyword(item.BaseDir)"
                      >
                        {{ item.BaseDir }}
                      </div>
                      <div style="color: grey">
                        {{ item.Path }}
                      </div>
                    </div>
                  </q-popup-proxy>
                </span>
                <span
                  @click="copyText(item.Title)"
                  class="cursor-pointer"
                  style="
                    color: rgb(239, 30, 30);
                    margin-right: 1px;
                    background-color: rgba(0, 0, 0, 0.1);
                  "
                >
                  {{ humanStorageSize(item.Size) }}
                </span>
                <span
                  style="
                    color: rgb(161, 100, 19);
                    background-color: rgba(0, 0, 0, 0.1);
                    margin-right: 1px;
                  "
                  class="cursor-pointer"
                  @click="goActress(item.Actress)"
                  >{{ item.Actress }}</span
                >

                <span
                  style="
                    color: rgb(239, 30, 30);
                    background-color: rgba(0, 0, 0, 0.1);
                    margin-right: 4px;
                  "
                  class="cursor-pointer"
                  @click="copyText(item.Code)"
                  >{{ item.title?item.Code?.substring(0, 12):item.Code }}
                  <q-tooltip class="bg-white text-primary">{{
                    item.Code
                  }}</q-tooltip>
                </span>

                {{ formatTitle(item.Title) }}
              </div>
            </div>
          </q-card>
            </transition-group>
        </div>
        <!-- 页面滚动器 -->
        <q-page-scroller
          position="bottom-right"
          :scroll-offset="150"
          :offset="[18, 100]"
        >
          <q-btn fab icon="keyboard_arrow_up" color="accent" />
        </q-page-scroller>

        <div v-if="view.queryParam.Page < view.resultData.TotalPage">
          <div style="height: 8vh">
            <q-btn
              v-show="systemProperty.searchPageAutoPullData && isMoreLoading"
              color="primary"
              label="加载中..."
            ></q-btn>
          </div>

          <div
            v-intersection="onIntersection"
            style="height: 8vh; color: #9e089e"
            @click="
              () => {
                pullNextPage();
              }
            "
          >
            点击可加载更多数据
          </div>
        </div>
      </q-page>
      <!-- 上一页按钮 -->
      <q-page-sticky
        style="z-index: 9"
        position="bottom-left"
        v-if="view.queryParam.Page > 1"
        :offset="[6, isSmall ? 200 : 366]"
      >
        <q-btn
          glossy
          class="page-sticky"
          flat
          text-color="blue"
          :label="`P${view.queryParam.Page - 1}`"
          @click="gotoPageNo(view.queryParam.Page - 1)"
        ></q-btn>
      </q-page-sticky>

      <!-- 下一页按钮 -->
      <q-page-sticky
        style="z-index: 9"
        position="bottom-right"
        :offset="[10, isSmall ? 200 : 366]"
      >
        <q-btn
          v-if="view.queryParam.Page < view.resultData.TotalPage"
          flat
          glossy
          class="page-sticky"
          text-color="blue"
          :label="`P${view.queryParam.Page + 1}`"
          @click="gotoPageNo(view.queryParam.Page + 1)"
        ></q-btn>
      </q-page-sticky>
    </q-page-container>
  </q-layout>

  <!-- 视频播放器 -->

  <InnerVideoPlayer
    ref="videoRef"
    @next-one="viewNextOne('play')"
    @prev-one="viewPrevOne('play')"
    @refresh-disk="refreshDebounceFn"
    @choose-data="
      (d) => {
        saveParam();
        view.currentDataInPlayer = d;
      }
    "
    @close="view.currentDataInPlayer = {}"
  />

  <!-- 文件编辑对话框 -->
  <FileEdit
    ref="fileEditRef"
    @plus-one="view.renameCount = view.renameCount + 1"
    @sub-one="view.renameCount = view.renameCount - 1"
    @next-one="viewNextOne('info')"
    @prev-one="viewPrevOne('info')"
    @success="refreshDebounceFn"
    @hide="view.currentDataInEditor = {}"
  />
  <!-- 文件信息对话框 -->
  <FileInfo
    ref="fileInfoRef"
    @next-one="viewNextOne('edit')"
    @prev-one="viewPrevOne('edit')"
    @hide="view.currentDataInEditor = {}"
  />

  <!-- 列表编辑对话框 @close="fetchSearch" -->
  <ListEdit
    ref="listEditRef"
    @callback-word="
      (e) => {
        searchKeyword(e);
      }
    "
  />
  <!-- 截图对话框 -->
  <Screenshot
    ref="fileCutImageRef"
    @next-one="viewNextOne('cut')"
    @prev-one="viewPrevOne('cut')"
    @hide="view.currentDataInEditor = {}"
    @close="
      () => {
        window.location.reload();
      }
    "
  />
  <q-dialog v-model="moveView.targetPathDialog" title="移动文件">
    <q-card style="min-width: 350px; width: 600px">
      <q-card-section>
        <!-- <div class="text-h6"  @click="moveView.targetPath = moveView.originPath">地址:{{ moveView.originPath }}</div> -->
        <q-input
          bg-color="green"
          label="原始地址"
          outlined
          :readonly="true"
          stack-label
          filled
          autogrow
          v-model="moveView.originPath"
          @click="moveView.targetPath = moveView.originPath"
        >
        </q-input>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input
          label="文件夹"
          stack-label
          filled
          autogrow
          v-model="moveView.targetPath"
        >
        </q-input>
      </q-card-section>
      <q-card-section class="q-pt-none">
        <q-input
          label="文件名"
          stack-label
          filled
          autogrow
          v-model="moveView.targetName"
        />
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn flat label="取消" v-close-popup />
        <q-btn flat label="移动" @click="moveThis" v-close-popup />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup>
import { date, format, useQuasar } from 'quasar';
const { humanStorageSize } = format;

const isSmall = computed(() => {
  return systemProperty.showStyle === 'sm';
});

const isLarge = computed(() => {
  return systemProperty.showStyle === 'lg';
});

const isMedium = computed(() => {
  return systemProperty.showStyle === 'md';
});

import {
  DeleteFile,
  MoveFile,
  OpenFileFolder,
  PlayMovie,
  ResetMovieType,
  SearchAPI,
  TransferTasksInfo,
} from 'components/api/searchAPI';
import { computed, onMounted,onUnmounted, provide, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { GetSettingInfo } from 'components/api/settingAPI';
import {
  DescEnum,
  FieldEnum,
  /* formatCode, */
  formatSeries,
  formatTitle,
  getLabelByValue,
  MovieTypeOptions,
  MovieTypeSelects,
} from 'components/utils';
import { getJpg, getPng } from 'components/utils/images';

const getImage = (id) => {
  if (systemProperty.showImage === 'poster') {
    return getPng(id);
  }
  return getJpg(id);
};

const getBlurImage = (id) => {
  if (systemProperty.showImage === 'poster') {
    return `${getPng(id)}?blur=20`;
  }
  return `${getJpg(id)}?blur=20`;
};

import DataPop from 'components/DataPop.vue';
import IndexButton from 'components/IndexButton.vue';
import TagPop from 'components/TagPop.vue';
import { useSystemProperty } from 'stores/System';
import FileEdit from './components/FileEditDialog.vue';
import FileInfo from './components/FileInfoDialog.vue';
import ListEdit from './components/ListEditDialog.vue';
import Screenshot from './components/ScreenshotDialog.vue';
import InnerVideoPlayer from './components/VideoPlayerInPicture.vue';

import { onKeyStroke, useClipboard, useDebounceFn } from '@vueuse/core';

// 变量声明
const $q = useQuasar();
const fileEditRef = ref(null);
const fileInfoRef = ref(null);
const listEditRef = ref(null);
const videoRef = ref(null);
const indexButton = ref(null);
const fileCutImageRef = ref(null);
const isMoreLoading = ref(false);
const isFetching = ref(false);
const pageOptions = ref([10, 12, 20, 30, 50, 200]);

const moveView = reactive({
  targetPath: '',
  targetPathDialog: false,
  targetId: '',
});
const view = reactive({
  renameCount: 0,
  indexDone: 0,
  currentDataInPlayer: {},
  currentDataInEditor: {},
  settingInfo: {},
  allPageNo: 0,
  resultShow: '',
  queryParam: {
    Keyword: '',
    MovieType: '',
    OnlyRepeat: false,
    Page: 1,
    PageSize: 20,
    SortField: 'MTime',
    SortType: 'desc',
  },
  resultData: {},
});

const source = ref('Hello');
const { copy } = useClipboard({ source });

const systemProperty = useSystemProperty();
const suggestions = computed(() => {
  return systemProperty.getSuggestions;
});

const listButtons = computed(() => {
  return view.settingInfo.Buttons;
});

const today = new Date();
const reg = /\w+[-_]\d+/;

const scrollTop = () => {
  const target = document.getElementsByClassName('scroll');
  if (target && target[2]) {
    target[2].scrollTo(0, 0, 500);
  }
};

const themeStyle = computed(() => {
  return {
    color: 'var(--q-text-primary)',
    backgroundColor: 'var(--q-bg-card)',
  };
});

const themeIcon = computed(() => {
  return systemProperty.theme === 'natural' ? 'eco' : 'star';
});

const currentThemeLabel = computed(() => {
  return systemProperty.theme === 'natural' ? '自然' : '星空';
});

const setTheme = (theme) => {
  systemProperty.theme = theme;
  const html = document.documentElement;
  if (theme === 'natural') {
    html.classList.add('theme-natural');
  } else {
    html.classList.remove('theme-natural');
  }
};

onKeyStroke(['Enter'], () => {
  fetchSearch();
});

const btnSize = (position) => {
  if (position === 'head') {
    if (isLarge.value) {
      return '14px';
    }
    if (isMedium.value) {
      return '13px';
    }
    return '12px';
  }
  if (position === 'top') {
    if (isLarge.value) {
      return '14px';
    }
    if (isMedium.value) {
      return '12px';
    }
    return '12px';
  }
  if (position === 'footer' || !position) {
    if (isLarge.value) {
      return '14px';
    }
    if (isMedium.value) {
      return '12px';
    }
    return '12px';
  }
};

const getTimeAgo = (MTime) => {
  const days = date.getDateDiff(today, MTime, 'days');
  if (days > 365) {
    const years = Math.floor(days / 365);
    return `${years}年`;
  }
  if (days > 30) {
    const months = Math.floor(days / 30);
    return `${months}月`;
  }
  if (days > 0) {
    return `${days}天`;
  }
  return '今天';
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

const redirectUrl = (item) => {
  view.queryParam = { ...item };
  fetchSearch();
  return;
};
const listEditCallback = (data) => {
  const { settingInfo } = data;
  if (settingInfo) {
    view.settingInfo = settingInfo;
  }
};

const showButton = (name) => {
  if (!listButtons.value || listButtons.value.length === 0) {
    return true;
  }
  return listButtons.value.indexOf(name) >= 0;
};

const simgleWindow = computed(() => {
  return systemProperty.singleWindow;
});

const playByPage = (item) => {
  systemProperty.savePlayTime(item.Id);
  const url = `#/playing/${item.Id}?a=refresh`;
  view.playBy = 'fullscreen';
  if ($q.platform.is.electron) {
    window.electron.createWindow({ router: url });
  } else {
    console.log('singleWindow', simgleWindow.value);
    const options = `width=${simgleWindow.value.width},height=${simgleWindow.value.height},titleBarStyle=`;
    window.open(url, 'player', options);
  }
};

const searchCode = (item) => {
  let vcode = item.Code;
  vcode = vcode.replace(/[\r\n\t]+/g, '');
  vcode = vcode.replace(/&nbsp;/g, '');
  vcode = vcode.trimEnd();
  const itemCode = vcode.match(reg);
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

const focusEvent = (e) => {
  e.target.select();
};

const openFolder = (item) => {
  if ($q.platform.is.electron) {
    window.electron.showInFolder(item.Path);
  } else {
    commonExec(OpenFileFolder(item.Id));
  }
};

const playBySystem = (item) => {
  systemProperty.savePlayTime(item.Id);
  view.playBy = 'system';
  if ($q.platform.is.electron) {
    window.electron.playMovie(item.Id);
  } else {
    commonExec(PlayMovie(item.Id));
  }
};

const confirmDelete = (item) => {
  $q.dialog({
    title: item.Name,
    message: '确定删除吗?',
    cancel: true,
    persistent: true,
  }).onOk(() => {
    commonExec(DeleteFile(item.Id)).then(() => {
      refreshDebounceFn(item);
    });
  });
};

const fetchGetSettingInfo = async () => {
  const data = await GetSettingInfo();
  view.settingInfo = data.data;
  systemProperty.SettingInfo = data.data;
  if (view.settingInfo.Pages && view.settingInfo.Pages.length > 0) {
    pageOptions.value = view.settingInfo.Pages.map((item) => {
      return Number(item);
    });
  }
};

const commonExec = async (exec) => {
  const { Code, Message } = (await exec) || {};
  if (Code !== 200) {
    $q.notify({ message: `${Message}`, position: 'top-right' });
  }
};

const copyText = async (str) => {
  if (str && str.startsWith('-')) {
    str = str.substring(1);
  }
  await copy(str);
  $q.notify({ message: `${str}`, position: 'top-right' });
};

const goActress = (Actress) => {
  if (!systemProperty.goActressNewWidow) {
    view.queryParam.Keyword = Actress;
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
        Keyword: Actress,
      },
    });
    window.open(routeData.href, '_blank');
  }
};

const picInPic = async (item, webFullScreen) => {
  if (!item) {
    return;
  }
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

const openListEditRef = (tabName) => {
  listEditRef.value.open({
    queryParam: view.queryParam,
    settingInfo: view.settingInfo,
    cb: listEditCallback,
    tabName,
  });
};

const openFileInfoRef = (item, playing) => {
  view.currentDataInEditor = item;
  if (playing) {
    view.playBy = 'dialog';
  }
  fileInfoRef.value.open({ item, playing });
};

const pictureRightClick = async (item, e) => {
  console.log('pictureRightClick', view.playBy);
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

const refreshDebounceFn = async (item) => {
  await indexButton.value.refreshIndex(item);
  const timer = setTimeout(async () => {
    await fetchSearch();
    clearTimeout(timer);
  }, 500);
};

const searchKeyword = async (keyword) => {
  view.queryParam.Keyword = keyword;
  await fetchSearch();
};

const pageTimestamps = new Map();
const CACHE_EXPIRE_TIME = 10 * 60 * 1000;

const clearImageCache = (currentPage) => {
  const now = Date.now();
  pageTimestamps.set(currentPage, now);
  if (window.caches) {
    caches.keys().then((names) => {
      names.forEach((name) => {
        if (name.includes('image')) {
          const match = name.match(/page-(\d+)/);
          if (match) {
            const page = parseInt(match[1]);
            const timestamp = pageTimestamps.get(page);
            if (timestamp && now - timestamp > CACHE_EXPIRE_TIME) {
              caches.delete(name);
              pageTimestamps.delete(page);
            }
          }
        }
      });
    });
  }
};

const gotoPageNo = async (no) => {
  console.log('gotoPageNo', no);
  clearImageCache(view.queryParam.Page);
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
  console.log('pageNoGoto', no);
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
    const data = await SearchAPI(view.queryParam);
    view.resultData.Data.push(...data.Data);
    isMoreLoading.value = false;
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
let refreshTask = null;
const fetchSearch = async (newBlank) => {
  if (isFetching.value) return;
  isFetching.value = true;

  try {
    saveParam(newBlank);
    const { Keyword } = view.queryParam;
    if (!Keyword || Keyword == '') {
      view.allPageNo = view.queryParam.Page;
    }
    const data = await SearchAPI(view.queryParam);
    console.log('搜索结果:', data);
    view.resultData = { ...data };
    const { ResultSize, ResultCnt, CurSize, CurCnt } = data;
    document.title = `${Keyword || ''}  ${ResultSize} {${ResultCnt}}`;
    view.resultShow = `${ResultSize} {${ResultCnt}} ${CurSize} {${CurCnt}}`;
  } catch (e) {
    console.error('搜索请求异常:', e);
    $q.notify({ type: 'negative', message: '请求失败' });
  } finally {
    isFetching.value = false;
  }

  if (refreshTask) {
    clearTimeout(refreshTask);
  }
  refreshTask = setTimeout(() => {
    fetchSearch();
  }, 180000);
};

const moveThis = async () => {
  console.log('moveThis', moveView);
  moveView.targetPathDialog = false;
  const res = await MoveFile({
    Id: moveView.targetId,
    Path: moveView.targetPath,
    Title: moveView.targetName,
  });
  if (res.Code === 200) {
    $q.notify({
      type: 'negative',
      message: res.Message,
      position: 'top-right',
    });
  } else {
    $q.notify({
      type: 'negative',
      message: res.Message,
      position: 'top-right',
    });
  }
};

const setMovieType = async (Id, Type) => {
  const { Code, Message } = await ResetMovieType(Id, Type);
  if (Code === 200) {
    $q.notify({ type: 'negative', message: Message, position: 'top-right' });
  } else {
    $q.notify({ type: 'warning', message: Message, position: 'top-right' });
  }
};

const fetchTasking = async () => {
  const res = await TransferTasksInfo();
  let runningTaskCount = 0
  Object.keys(res.Data).forEach((key) => {
    const v = res.Data[key];
     if (v.Status == '执行中') {
      runningTaskCount++;
    }
  });
  view.runningTaskCount = runningTaskCount;
};

const thisRoute = useRoute();
const { resolve, push } = useRouter();

const saveParam = () => {
  systemProperty.syncSearchParam(view.queryParam);
  systemProperty.expireTime = new Date().getTime() + 1000 * 60 * 60 * 2;
  localStorage.setItem('queryParam', JSON.stringify(view.queryParam));
  localStorage.setItem('isAuthenticated', 'true');
  const { Page, PageSize, MovieType, SortField, SortType, Keyword } =
    view.queryParam;
  push({
    path: '/search',
    query: {
      Page,
      PageSize,
      MovieType,
      SortField,
      SortType,
      Keyword,
    },
  });
};

const gotoNextPage = () => {
  gotoPageNo(view.queryParam.Page + 1);
};
const gotoPrevPage = () => {
  gotoPageNo(view.queryParam.Page - 1);
};

provide('refreshDebounceFn', refreshDebounceFn);
provide('searchKeyword', searchKeyword);
provide('gotoNextPage', gotoNextPage);
provide('gotoPrevPage', gotoPrevPage);
let taskInterval = null;
onMounted(async () => {
  taskInterval = setInterval(() => {
    fetchTasking()
  }, 9000);
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
    showStyle,
    from,
  } = thisRoute.query;
  await fetchGetSettingInfo();
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
    view.queryParam.showStyle = showStyle;
  } else {
    if (from === 'index') {
      const piniaParam = systemProperty.FileSearchParam;
      if (piniaParam) {
        console.log('piniaParam', piniaParam);
        view.queryParam = piniaParam;
      }
    } else {
      const storage = JSON.parse(localStorage.getItem('queryParam'));
      if (storage) {
        console.log('storage', storage);
        view.queryParam = storage;
      }
    }
  }
  fetchSearch();
});

onUnmounted(() => {
  if (refreshTask) {
    clearTimeout(refreshTask);
    refreshTask = null;
  }
  if (taskInterval) {
    clearInterval(taskInterval);
    taskInterval = null;
  }
});
</script>

<style lang="scss" scoped>
// 列表过渡动画
.card-list-enter-active {
  transition: all 0.4s ease-out;
}

.card-list-leave-active {
  transition: all 0.3s ease-in;
}

.card-list-enter-from {
  opacity: 0;
  transform: translateY(20px) scale(0.95);
}

.card-list-leave-to {
  opacity: 0;
  transform: translateY(-10px) scale(0.98);
}

.card-list-move {
  transition: transform 0.3s ease;
}

// 隐藏滚动条
.scrollRef::-webkit-scrollbar {
  display: none;
}

// 兼容 Firefox
.scrollRef {
  scrollbar-width: none;
}

// 兼容 IE 和 Edge
.scrollRef {
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

.mr10 {
  margin-right: 4px;
}

.card-top-tag {
  position: absolute;
  display: flex-start;
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
  height: 376px;
  overflow: hidden;
}

.large-result-image {
  width: 100%;
  height: auto;

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
  height: 192px;
  overflow: hidden;
}

.medium-result-image {
  width: 100%;
  height: auto;

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

.small-result {
  padding: 1px;
  width: 180px;
  height: 240px;
  overflow: hidden;
  align-items: center;
  justify-content: center;
}

.small-result-image {
  width: 100%;
  height: auto;

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

.float-btn {
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);

  .btn-row {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    scrollbar-width: 1px;
    scrollbar-color: transparent transparent;
  }

  .content-row {
    display: -webkit-box; /* 将对象作为弹性伸缩盒子模型显示 */
    -webkit-box-orient: vertical; /* 设置子元素的排列方式为垂直方向 */
    line-clamp: 3; /* 设置显示的行数 */
    overflow: hidden; /* 隐藏溢出文本 */
    text-overflow: ellipsis; /* 显示省略号 */

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

.q-card__section--vert {
  padding: 4px;
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
      background: linear-gradient(
        180deg,
        rgba(0, 0, 0, 0),
        rgba(255, 255, 255, 0.1)
      );
    }

    .q-img__error {
      background: rgba(255, 255, 255, 0.05);

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

.theme-selector-btn {
  border-radius: 8px;
  transition: all 0.3s ease;

  &:hover {
    background: rgba(99, 102, 241, 0.15);
  }

  :deep(.q-btn__content) {
    font-weight: 500;
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
</style>
