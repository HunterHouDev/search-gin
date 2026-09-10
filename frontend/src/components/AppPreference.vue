<template>
  <q-btn-dropdown flat glossy dense class="app-prefer-dropdown" content-class="app-prefer-menu">
    <template v-slot:label>
      <q-icon :name="themeIcon" size="14px" class="theme-icon" />
      <span class="theme-label-text q-ml-xs">{{ currentThemeLabel }}</span>
    </template>

    <div class="theme-panel row q-pa-md">
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
      <div class="section-col section-wide column q-gutter-xs">
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
/* 菜单根节点定宽：放在面板上会被 q-menu 的收缩宽度裁掉 */
/* q-menu 根节点不继承 scoped 属性，且会内联写 max-width，故用 :global + !important 覆盖 */
:global(.app-prefer-menu) {
  width: 680px !important;
  max-width: calc(100vw - 16px) !important;
}

.theme-panel {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.section-col {
  min-width: 180px;
  flex: 1 1 180px;
}

/* 窄屏：菜单占满视口宽度，分区折成两列，行为分区独占一行；按钮只留图标 */
@media (max-width: 599px) {
  :global(.app-prefer-menu) {
    width: calc(100vw - 12px) !important;
    max-width: calc(100vw - 12px) !important;
  }

  .theme-panel {
    padding: 8px;
    gap: 12px;
  }

  .section-col {
    flex: 1 1 140px;
    min-width: 0;
  }

  .section-col.section-wide {
    flex-basis: 100%;
  }

  .theme-label-text {
    display: none;
  }
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
