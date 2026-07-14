<template>
  <q-btn-dropdown  flat glossy dense class="app-prefer-dropdown">
    <template v-slot:label>
      <q-icon :name="themeIcon" size="14px" class="theme-icon" />
      {{ currentThemeLabel }}
    </template>

    <div class="theme-panel row q-gutter-md q-pa-md">
      <!-- 主题 -->
      <div class="section-col column q-gutter-sm">
        <div class="text-caption text-weight-bold text-grey-7 q-mb-xs">主题</div>
        <q-btn flat align="left"  :color="systemProperty.theme === 'star' ? 'primary' : 'grey-7'"
          @click="setTheme('star')" class="option-btn">
          <q-icon left name="star" class="q-mr-sm" />
          <span class="text-body2">星空主题</span>
          <q-space />
          <q-icon v-if="systemProperty.theme === 'star'" name="check" color="primary" />
        </q-btn>
        <q-btn flat align="left"  :color="systemProperty.theme === 'natural' ? 'primary' : 'grey-7'"
          @click="setTheme('natural')" class="option-btn">
          <q-icon left name="eco" class="q-mr-sm" />
          <span class="text-body2">自然主题</span>
          <q-space />
          <q-icon v-if="systemProperty.theme === 'natural'" name="check" color="primary" />
        </q-btn>
      </div>

      <!-- 显示模式 -->
      <div class="section-col column q-gutter-sm">
        <div class="text-caption text-weight-bold text-grey-7 q-mb-xs">显示</div>
        <q-btn flat align="left" :color="systemProperty.showImage === 'cover' ? 'primary' : 'grey-7'"
          @click="systemProperty.showImage = 'cover'" class="option-btn">
          <q-icon left name="image" class="q-mr-sm" />
          <span class="text-body2">封面模式</span>
          <q-space />
          <q-icon v-if="systemProperty.showImage === 'cover'" name="check" color="primary" />
        </q-btn>
        <q-btn flat align="left" :color="systemProperty.showImage === 'poster' ? 'primary' : 'grey-7'"
          @click="systemProperty.showImage = 'poster'" class="option-btn">
          <q-icon left name="movie" class="q-mr-sm" />
          <span class="text-body2">海报模式</span>
          <q-space />
          <q-icon v-if="systemProperty.showImage === 'poster'" name="check" color="primary" />
        </q-btn>
      </div>

      <!-- 卡片大小 -->
      <div class="section-col column q-gutter-sm">
        <div class="text-caption text-weight-bold text-grey-7 q-mb-xs">卡片</div>
        <q-btn flat align="left" :color="systemProperty.showStyle === 'lg' ? 'primary' : 'grey-7'"
          @click="setShowStyle('lg')" class="option-btn">
          <q-icon left name="view_module" class="q-mr-sm" />
          <span class="text-body2">大尺寸</span>
          <q-space />
          <q-icon v-if="systemProperty.showStyle === 'lg'" name="check" color="primary" />
        </q-btn>
        <q-btn flat align="left" :color="systemProperty.showStyle === 'md' ? 'primary' : 'grey-7'"
          @click="setShowStyle('md')" class="option-btn">
          <q-icon left name="grid_view" class="q-mr-sm" />
          <span class="text-body2">中尺寸</span>
          <q-space />
          <q-icon v-if="systemProperty.showStyle === 'md'" name="check" color="primary" />
        </q-btn>
        <q-btn flat align="left" :color="systemProperty.showStyle === 'sm' ? 'primary' : 'grey-7'"
          @click="setShowStyle('sm')" class="option-btn">
          <q-icon left name="apps" class="q-mr-sm" />
          <span class="text-body2">小尺寸</span>
          <q-space />
          <q-icon v-if="systemProperty.showStyle === 'sm'" name="check" color="primary" />
        </q-btn>
      </div>

      <!-- 行为 -->
      <div class="section-col column q-gutter-xs">
        <div class="text-caption text-weight-bold text-grey-7 q-mb-xs">行为</div>
        <q-item tag="label" dense>
          <q-item-section>
            <q-item-label class="text-body2">搜索自动加载</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-toggle v-model="systemProperty.searchPageAutoPullData" color="primary" size="sm" />
          </q-item-section>
        </q-item>
        <q-item tag="label" dense>
          <q-item-section>
            <q-item-label class="text-body2">标签允许多选</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-toggle v-model="systemProperty.submitMutiTag" color="primary" size="sm"
              :true-value="true" :false-value="false" />
          </q-item-section>
        </q-item>
        <q-item tag="label" dense>
          <q-item-section>
            <q-item-label class="text-body2">图鉴新窗口</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-toggle v-model="systemProperty.goAuthorNewWidow" color="primary" size="sm" />
          </q-item-section>
        </q-item>
        <q-item tag="label" dense>
          <q-item-section>
            <q-item-label class="text-body2">Search新窗口</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-toggle v-model="systemProperty.goSearchNewWidow" color="primary" size="sm" />
          </q-item-section>
        </q-item>
      </div>
    </div>
  </q-btn-dropdown>
</template>

<script setup>
import { computed } from 'vue';
import { useSystemProperty } from 'stores/System';

const systemProperty = useSystemProperty();

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
  min-width: 680px;
}

.section-col {
  min-width: 180px;
  flex: 1;


}

.section-col .option-btn {
  width: 100%;
  justify-content: flex-start;
  padding: 6px 10px;
  border-radius: 8px;
}


.section-col .q-item {
  padding: 6px 10px;
  min-height: 0;
}

.section-col .q-item__label--caption {
  font-size: 11px;
}
</style>
