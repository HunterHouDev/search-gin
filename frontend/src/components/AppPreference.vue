<template>
  <q-btn-dropdown no-caps flat glossy dense>
    <template v-slot:label>
      <q-icon :name="themeIcon" size="14px" class="theme-icon" />
      {{ currentThemeLabel }}
    </template>

    <div class="theme-panel row">
      <!-- 左侧分类 -->
      <div class="theme-sidebar bg-grey-1">
        <q-list dense>
          <q-item clickable v-ripple :active="tab === 'theme'" @click="tab = 'theme'"
            class="sidebar-item" active-class="bg-primary text-white">
            <q-item-section avatar>
              <q-icon name="palette" size="16px" />
            </q-item-section>
            <q-item-section>主题</q-item-section>
          </q-item>
          <q-item clickable v-ripple :active="tab === 'show'" @click="tab = 'show'"
            class="sidebar-item" active-class="bg-primary text-white">
            <q-item-section avatar>
              <q-icon name="image" size="16px" />
            </q-item-section>
            <q-item-section>显示</q-item-section>
          </q-item>
          <q-item clickable v-ripple :active="tab === 'size'" @click="tab = 'size'"
            class="sidebar-item" active-class="bg-primary text-white">
            <q-item-section avatar>
              <q-icon name="grid_view" size="16px" />
            </q-item-section>
            <q-item-section>卡片</q-item-section>
          </q-item>
          <q-item clickable v-ripple :active="tab === 'behavior'" @click="tab = 'behavior'"
            class="sidebar-item" active-class="bg-primary text-white">
            <q-item-section avatar>
              <q-icon name="settings" size="16px" />
            </q-item-section>
            <q-item-section>行为</q-item-section>
          </q-item>
        </q-list>
      </div>

      <!-- 右侧内容 -->
      <div class="theme-content q-pa-sm">
        <!-- 主题 -->
        <div v-show="tab === 'theme'" class="column q-gutter-sm">
          <q-btn flat align="left" v-close-popup :color="systemProperty.theme === 'star' ? 'primary' : 'grey-7'"
            @click="setTheme('star')">
            <q-icon left name="star" class="q-mr-sm" />
            <div class="text-left">
              <div class="text-body2">星空主题</div>
              <div class="text-caption text-grey-5">深蓝紫配色</div>
            </div>
            <q-space />
            <q-icon v-if="systemProperty.theme === 'star'" name="check" color="primary" />
          </q-btn>
          <q-btn flat align="left" v-close-popup :color="systemProperty.theme === 'natural' ? 'primary' : 'grey-7'"
            @click="setTheme('natural')">
            <q-icon left name="eco" class="q-mr-sm" />
            <div class="text-left">
              <div class="text-body2">自然主题</div>
              <div class="text-caption text-grey-5">温暖米色绿植</div>
            </div>
            <q-space />
            <q-icon v-if="systemProperty.theme === 'natural'" name="check" color="primary" />
          </q-btn>
        </div>

        <!-- 显示模式 -->
        <div v-show="tab === 'show'" class="column q-gutter-sm">
          <q-btn flat align="left" :color="systemProperty.showImage === 'cover' ? 'primary' : 'grey-7'"
            @click="systemProperty.showImage = 'cover'">
            <q-icon left name="image" class="q-mr-sm" />
            <div class="text-left">
              <div class="text-body2">封面模式</div>
              <div class="text-caption text-grey-5">展示完整封面图</div>
            </div>
            <q-space />
            <q-icon v-if="systemProperty.showImage === 'cover'" name="check" color="primary" />
          </q-btn>
          <q-btn flat align="left" :color="systemProperty.showImage === 'poster' ? 'primary' : 'grey-7'"
            @click="systemProperty.showImage = 'poster'">
            <q-icon left name="movie" class="q-mr-sm" />
            <div class="text-left">
              <div class="text-body2">海报模式</div>
              <div class="text-caption text-grey-5">展示电影海报</div>
            </div>
            <q-space />
            <q-icon v-if="systemProperty.showImage === 'poster'" name="check" color="primary" />
          </q-btn>
        </div>

        <!-- 卡片大小 -->
        <div v-show="tab === 'size'" class="column q-gutter-sm">
          <q-btn flat align="left" :color="systemProperty.showStyle === 'lg' ? 'primary' : 'grey-7'"
            @click="setShowStyle('lg')">
            <q-icon left name="view_module" class="q-mr-sm" />
            <div class="text-left">
              <div class="text-body2">大尺寸</div>
              <div class="text-caption text-grey-5">大尺寸卡片</div>
            </div>
            <q-space />
            <q-icon v-if="systemProperty.showStyle === 'lg'" name="check" color="primary" />
          </q-btn>
          <q-btn flat align="left" :color="systemProperty.showStyle === 'md' ? 'primary' : 'grey-7'"
            @click="setShowStyle('md')">
            <q-icon left name="grid_view" class="q-mr-sm" />
            <div class="text-left">
              <div class="text-body2">中尺寸</div>
              <div class="text-caption text-grey-5">中等尺寸卡片</div>
            </div>
            <q-space />
            <q-icon v-if="systemProperty.showStyle === 'md'" name="check" color="primary" />
          </q-btn>
          <q-btn flat align="left" :color="systemProperty.showStyle === 'sm' ? 'primary' : 'grey-7'"
            @click="setShowStyle('sm')">
            <q-icon left name="apps" class="q-mr-sm" />
            <div class="text-left">
              <div class="text-body2">小尺寸</div>
              <div class="text-caption text-grey-5">紧凑尺寸卡片</div>
            </div>
            <q-space />
            <q-icon v-if="systemProperty.showStyle === 'sm'" name="check" color="primary" />
          </q-btn>
        </div>

        <!-- 用户行为 -->
        <div v-show="tab === 'behavior'" class="column q-gutter-xs">
          <q-item tag="label" dense>
            <q-item-section>
              <q-item-label>搜索自动加载</q-item-label>
              <q-item-label caption>进入搜索页自动拉取数据</q-item-label>
            </q-item-section>
            <q-item-section side>
              <q-toggle v-model="systemProperty.searchPageAutoPullData" color="primary" size="sm" />
            </q-item-section>
          </q-item>
          <q-item tag="label" dense>
            <q-item-section>
              <q-item-label>种草多选</q-item-label>
              <q-item-label caption>标签操作允许多选</q-item-label>
            </q-item-section>
            <q-item-section side>
              <q-toggle v-model="systemProperty.submitMutiTag" color="primary" size="sm"
                :true-value="true" :false-value="false" />
            </q-item-section>
          </q-item>
          <q-item tag="label" dense>
            <q-item-section>
              <q-item-label>图鉴点击</q-item-label>
              <q-item-label caption>新窗口打开作者页面</q-item-label>
            </q-item-section>
            <q-item-section side>
              <q-toggle v-model="systemProperty.goAuthorNewWidow" color="primary" size="sm" />
            </q-item-section>
          </q-item>
          <q-item tag="label" dense>
            <q-item-section>
              <q-item-label>Search 点击</q-item-label>
              <q-item-label caption>新窗口打开搜索结果</q-item-label>
            </q-item-section>
            <q-item-section side>
              <q-toggle v-model="systemProperty.goSearchNewWidow" color="primary" size="sm" />
            </q-item-section>
          </q-item>
        </div>
      </div>
    </div>
  </q-btn-dropdown>
</template>

<script setup>
import { computed, ref } from 'vue';
import { useSystemProperty } from 'stores/System';

const systemProperty = useSystemProperty();

const tab = ref('theme');

const themeIcon = computed(() => systemProperty.theme === 'natural' ? 'eco' : 'star');
const currentThemeLabel = computed(() => systemProperty.theme === 'natural' ? '自然' : '星空');

const setTheme = (theme) => {
  systemProperty.theme = theme;
  const html = document.documentElement;
  if (theme === 'natural') {
    html.classList.add('theme-natural');
  } else {
    html.classList.remove('theme-natural');
  }
};

const setShowStyle = (style) => {
  systemProperty.showStyle = style;
};
</script>

<style scoped>
.theme-panel {
  min-width: 360px;
  min-height: 210px;
}

.theme-sidebar {
  width: 80px;
  min-width: 80px;
  border-right: 1px solid rgba(0, 0, 0, 0.08);
  padding: 4px 0;
}

.sidebar-item {
  min-height: 40px;
  border-radius: 0 20px 20px 0;
  margin: 2px 6px 2px 0;
}

.theme-content {
  flex: 1;
  min-width: 0;
}

.theme-content .q-btn {
  width: 100%;
  justify-content: flex-start;
  padding: 8px 12px;
  border-radius: 8px;
}
</style>
