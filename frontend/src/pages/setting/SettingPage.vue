<template>
  <div class="setting-page">
    <!-- 顶部 Tab -->
    <q-tabs v-model="mainTab" class="main-tabs bg-black text-white" align="justify"
      :active-color="systemProperty.theme === 'natural' ? 'green' : 'white'"
      :indicator-color="systemProperty.theme === 'natural' ? 'green' : 'white'">
      <q-tab name="search" label="搜索设置" />
      <q-tab name="network" label="网络配置" />
      <q-tab name="dict" label="数据管理" />
    </q-tabs>

    <div class="setting-layout">
      <!-- 左侧分类导航 -->
      <div class="setting-sidebar">
        <div class="sidebar-tree">
          <div v-for="group in currentNavGroups" :key="group.name" class="tree-group">
            <div class="tree-group-header" @click="toggleGroup(group.name)">
              <q-icon :name="expandedGroups[group.name] ? 'expand_more' : 'chevron_right'" size="16px"
                class="tree-arrow" />
              <span class="group-label">{{ group.label }}</span>
            </div>
            <transition name="expand">
              <div v-show="expandedGroups[group.name]" class="tree-group-items">
                <div v-for="item in group.items" :key="item.id" class="tree-item"
                  :class="{ active: activeSection === item.id }" @click="scrollToSection(item.id)">
                  <span class="item-label">{{ item.label }}</span>
                </div>
              </div>
            </transition>
          </div>
        </div>
      </div>

      <!-- 右侧内容区域 -->
      <div class="setting-content" ref="contentRef" @scroll="onScroll">
        <!-- ========== 搜索设置 ========== -->
        <template v-if="mainTab === 'search'">
          <!-- 搜索配置 -->
          <section id="section-search" class="setting-section">
            <h3 class="section-title">搜索配置</h3>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">定时扫描</div>
                <div class="item-hint">开启后将定时扫描文件夹</div>
              </div>
              <div class="item-control">
                <q-toggle v-model="view.settingInfo.EnableTimeScan" color="primary" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">扫描目录</div>
                <div class="item-hint">选择要扫描的文件夹</div>
              </div>
              <div class="item-control">
                <MutiSelector v-bind:model-value="view.settingInfo.Dirs" :options="view.settingInfo.DirsLib"
                 :style="{ maxWidth: '80%' }" @onchange="(arr) => (view.settingInfo.Dirs = arr)" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">视频类型</div>
                <div class="item-hint">支持的视频文件扩展名</div>
              </div>
              <div class="item-control ">
                <MutiSelector v-bind:model-value="view.settingInfo.VideoTypes"
                  :style="{ maxWidth: '80%' }" :options="view.settingInfo.Types"
                  @onchange="(arr) => (view.settingInfo.VideoTypes = arr)" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">图片类型</div>
                <div class="item-hint">支持的图片文件扩展名</div>
              </div>
              <div class="item-control ">
                <MutiSelector v-bind:model-value="view.settingInfo.ImageTypes" :options="view.settingInfo.Types"
                  :style="{ maxWidth: '80%' }"
                  @onchange="(arr) => (view.settingInfo.ImageTypes = arr)" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">文档类型</div>
                <div class="item-hint">支持的文档文件扩展名</div>
              </div>
              <div class="item-control ">
                <MutiSelector v-bind:model-value="view.settingInfo.DocsTypes" :options="view.settingInfo.Types"
                  :style="{ maxWidth: '80%' }"
                  @onchange="(arr) => (view.settingInfo.DocsTypes = arr)" />
              </div>
            </div>
          </section>

          <!-- 播放器设置 -->
          <section id="section-player" class="setting-section">
            <h3 class="section-title">播放器设置</h3>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">系统播放器</div>
                <div class="item-hint">选择视频播放方式</div>
              </div>
              <div class="item-control">
                <q-select v-model="view.settingInfo.SystemPlayer" :options="[
                  { label: 'ffplay (内置)', value: 'ffplay' },
                  { label: 'system (系统默认)', value: 'system' }
                ]" emit-value map-options dense outlined class="select-control" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">播放器音量</div>
                <div class="item-hint">系统播放器默认音量 (0-100)</div>
              </div>
              <div class="item-control">
                <q-input v-model="view.settingInfo.SystemPlayerVolumn" type="number" :min="0" :max="100" dense outlined
                  class="number-input" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">播放器宽度</div>
                <div class="item-hint">系统播放器窗口宽度 (像素)</div>
              </div>
              <div class="item-control">
                <q-input v-model="view.settingInfo.SystemPlayerWidth" type="number" dense outlined
                  class="number-input" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">转码删除原文件</div>
                <div class="item-hint">转码完成后是否删除原始文件</div>
              </div>
              <div class="item-control">
                <q-toggle v-model="view.settingInfo.CutThenDelete" color="primary" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">硬件加速编码</div>
                <div class="item-hint">开启后H264/H265转码将调用GPU硬件加速</div>
              </div>
              <div class="item-control">
                <q-toggle v-model="view.settingInfo.HardwareAcceleration" color="primary" />
              </div>
            </div>

            <div v-if="view.settingInfo.HardwareAcceleration" class="setting-item sub-item">
              <div class="item-info">
                <div class="item-label">硬件加速模式</div>
                <div class="item-hint">
                  <span v-if="view.settingInfo.HardwareAccelMode" class="text-positive">
                    当前: {{ view.settingInfo.HardwareAccelMode }}
                  </span>
                  <span v-else class="text-warning">首次转码时自动检测</span>
                </div>
              </div>
            </div>
          </section>
        </template>
        <template v-if="mainTab === 'dict'">
          <!-- 启用配置 -->
          <section id="section-enable" class="setting-section">
            <h3 class="section-title">启用配置</h3>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">启用标签</div>
                <div class="item-hint">选择要启用的标签分类</div>
              </div>
              <div class="item-control ">
                <MutiSelector v-bind:model-value="view.settingInfo.Tags" :options="view.settingInfo.TagsLib"
                  :style="{ maxWidth: '80%' }" @onchange="(arr) => (view.settingInfo.Tags = arr)" />
              </div>
            </div>
          </section>

          <!-- 系统配置 -->
          <section id="section-system" class="setting-section">
            <h3 class="section-title">系统配置</h3>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">文件类型库</div>
                <div class="item-hint">可选的文件类型列表</div>
              </div>
              <div class="item-control ">
                <MutiInput v-model="view.settingInfo.Types" :style="{ maxWidth: '80%' }"
                  @onchange="(arr) => (view.settingInfo.Types = arr)" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">目录库</div>
                <div class="item-hint">可选的目录列表</div>
              </div>
              <div class="item-control ">
                <MutiInput v-model="view.settingInfo.DirsLib" :style="{ maxWidth: '80%' }"
                  @onchange="(arr) => (view.settingInfo.DirsLib = arr)" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">标签库</div>
                <div class="item-hint">可选的标签列表</div>
              </div>
              <div class="item-control ">
                <MutiInput v-model="view.settingInfo.TagsLib" :style="{ maxWidth: '80%' }"
                  @onchange="(arr) => (view.settingInfo.TagsLib = arr)" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">页面配置</div>
                <div class="item-hint">分页配置项</div>
              </div>
              <div class="item-control ">
                <MutiInput v-model="view.settingInfo.Pages" @onchange="(arr) => (view.settingInfo.Pages = arr)" />
              </div>
            </div>
          </section>
        </template>

        <!-- ========== 网络配置 ========== -->
        <template v-if="mainTab === 'network'">
          <section id="section-network" class="setting-section">
            <h3 class="section-title">网络配置</h3>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">Controller 端口</div>
                <div class="item-hint">API 服务监听端口</div>
              </div>
              <div class="item-control">
                <q-input v-model="view.settingInfo.ControllerHost" dense outlined class="port-input"
                  placeholder=":10081" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">File 端口</div>
                <div class="item-hint">文件服务监听端口</div>
              </div>
              <div class="item-control">
                <q-input v-model="view.settingInfo.FileHost" dense outlined class="port-input" placeholder=":10082" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">BaseUrl</div>
                <div class="item-hint">外部访问的基础 URL</div>
              </div>
              <div class="item-control">
                <q-input v-model="view.settingInfo.BaseUrl" dense outlined class="url-input" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">ImageUrl</div>
                <div class="item-hint">图片资源的基础 URL</div>
              </div>
              <div class="item-control">
                <q-input v-model="view.settingInfo.ImageUrl" dense outlined class="url-input" />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">备注</div>
                <div class="item-hint">关于网络配置的备注信息</div>
              </div>
              <div class="item-control textarea">
                <q-input v-model="view.settingInfo.Remark" type="textarea" autogrow :rows="3" dense outlined />
              </div>
            </div>

            <div class="setting-item">
              <div class="item-info">
                <div class="item-label">管理员密码</div>
                <div class="item-hint">设置后覆盖默认密码 qwer，留空则使用默认值</div>
              </div>
              <div class="item-control">
                <q-input v-model="view.settingInfo.AdminPassword" type="password" dense outlined
                  placeholder="留空则用默认密码 qwer" class="url-input" />
              </div>
            </div>
          </section>
        </template>

        <!-- 底部提交按钮 -->
        <div class="submit-bar">
          <q-btn :color="systemProperty.theme === 'star' ? 'black' : 'primary'" glossy rounded align="evenly"
            class="submit-btn" @click="submitForm">
            <q-icon name="save" class="q-mr-sm" />
            保存设置
          </q-btn>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useQuasar } from 'quasar';
import { onMounted, reactive, ref, computed, watch } from 'vue';
import {
  GetSettingInfo,
  PostSettingInfo,
  GetIpAddr,
  AppRestart,
} from '../../components/api/settingAPI';
import MutiSelector from '../../components/MutiSelector.vue';
import MutiInput from '../../components/MutiInput.vue';
import { useSystemProperty } from '../../stores/System';

const $q = useQuasar();
const contentRef = ref(null);
const mainTab = ref('search');
const activeSection = ref('');

const expandedGroups = reactive({
  search: true,
  player: true,
  enable: true,
  system: true,
  network: true,
});

// 搜索设置的导航
const searchNavGroups = [
  {
    name: 'search',
    label: '搜索配置',
    items: [{ id: 'section-search', label: '搜索选项' }],
  },
  {
    name: 'player',
    label: '播放器设置',
    items: [{ id: 'section-player', label: '播放器选项' }],
  },
];

// 数据管理的导航
const dictNavGroups = [
  {
    name: 'enable',
    label: '界面控制',
    items: [{ id: 'section-enable', label: '启用选项' }],
  },
  {
    name: 'system',
    label: '系统配置',
    items: [{ id: 'section-system', label: '配置选项' }],
  },
];

// 网络配置的导航
const networkNavGroups = [
  {
    name: 'network',
    label: '网络配置',
    items: [{ id: 'section-network', label: '网络选项' }],
  },
];

const currentNavGroups = computed(() => {
  switch (mainTab.value) {
    case 'search': return searchNavGroups;
    case 'dict': return dictNavGroups;
    case 'network': return networkNavGroups;
    default: return searchNavGroups;
  }
});

const allSectionIds = computed(() => {
  const groups = currentNavGroups.value;
  return groups.flatMap(g => g.items.map(i => i.id));
});

const toggleGroup = (name) => {
  expandedGroups[name] = !expandedGroups[name];
};

const scrollToSection = (id) => {
  const el = document.getElementById(id);
  if (el && contentRef.value) {
    const container = contentRef.value;
    const offsetTop = el.offsetTop - container.offsetTop;
    container.scrollTo({
      top: offsetTop,
      behavior: 'smooth',
    });
  }
};

const onScroll = () => {
  if (!contentRef.value) return;
  const container = contentRef.value;
  const scrollTop = container.scrollTop;
  const offset = 80;

  for (let i = allSectionIds.value.length - 1; i >= 0; i--) {
    const el = document.getElementById(allSectionIds.value[i]);
    if (el && el.offsetTop - container.offsetTop <= scrollTop + offset) {
      activeSection.value = allSectionIds.value[i];

      for (const group of currentNavGroups.value) {
        const item = group.items.find(it => it.id === activeSection.value);
        if (item) {
          expandedGroups[group.name] = true;
        }
      }
      break;
    }
  }
};

// 切换 tab 时重置滚动位置和激活项
watch(mainTab, () => {
  activeSection.value = '';
  if (contentRef.value) {
    contentRef.value.scrollTop = 0;
  }
});

const view = reactive({
  settingInfo: {
    Dirs: [],
    DirsLib: [],
    Types: [],
    VideoTypes: [],
    Tags: [],
    TagsLib: [],
    MovieTypes: [],
    Pages: [],
    EnableTimeScan: true,
    CutThenDelete: false,
    SystemPlayer: 'ffplay',
    SystemPlayerVolumn: '30',
    SystemPlayerWidth: '1280',
    HardwareAcceleration: false,
    HardwareAccelMode: '',
    FileHost: ':10081',
  },
  ipAddr: '',
});
const systemProperty = useSystemProperty();

const submitForm = async () => {
  const oldControllerHost = view.settingInfo.ControllerHost;
  view.settingInfo.Dirs = (view.settingInfo.Dirs || []).sort();
  view.settingInfo.DirsLib = (view.settingInfo.DirsLib || []).sort();
  view.settingInfo.Types = (view.settingInfo.Types || []).sort();
  view.settingInfo.VideoTypes = (view.settingInfo.VideoTypes || []).sort();

  const tagsLib = view.settingInfo.TagsLib || [];
  const dirsLib = view.settingInfo.DirsLib || [];
  const types = view.settingInfo.Types || [];
  view.settingInfo.Tags = (view.settingInfo.Tags || []).filter((item) => {
    return tagsLib.includes(item);
  });
  const sortedTags = [];
  tagsLib.forEach((item) => {
    if (view.settingInfo.Tags.includes(item)) {
      sortedTags.push(item);
    }
  });
  view.settingInfo.Tags = sortedTags;
  view.settingInfo.Dirs = (view.settingInfo.Dirs || []).filter((item) => {
    return dirsLib.includes(item);
  });
  view.settingInfo.VideoTypes = (view.settingInfo.VideoTypes || []).filter((item) => {
    return types.includes(item);
  });
  const { Code, Message } = await PostSettingInfo(view.settingInfo);
  if (Code != 200) {
    $q.notify({ message: `${Message}` });
  } else {
    $q.notify({ message: `${Message}` });
    const newControllerHost = view.settingInfo.ControllerHost;
    if (oldControllerHost !== newControllerHost) {
      systemProperty.setControllerHost(newControllerHost);
      $q.dialog({
        title: '端口已变更',
        message: `端口从 ${oldControllerHost} 变更为 ${newControllerHost}，需要重启应用才能生效。是否立即重启？`,
        cancel: true,
        persistent: true,
      }).onOk(async () => {
        await AppRestart();
      });
    }
  }
};

const fetchSearch = async () => {
  const { data } = await GetSettingInfo();
  view.settingInfo = {
    Dirs: [],
    DirsLib: [],
    Types: [],
    VideoTypes: [],
    Tags: [],
    TagsLib: [],
    MovieTypes: [],
    Pages: [],
    EnableTimeScan: true,
    CutThenDelete: false,
    SystemPlayer: 'ffplay',
    SystemPlayerVolumn: '30',
    SystemPlayerWidth: '1280',
    HardwareAcceleration: false,
    HardwareAccelMode: '',
    FileHost: ':10081',
    ...data,
  };
};

const queryIpAddr = async () => {
  const { Code, Data } = await GetIpAddr();
  if (Code == '200') {
    view.ipAddr = `http://${Data}:10081`;
  }
};

onMounted(() => {
  document.title = '设置';
  fetchSearch();
  queryIpAddr();
});
</script>

<style lang="scss" scoped>
.setting-page {
  height: 93vh;
  display: flex;
  flex-direction: column;
}

.main-tabs {
  flex-shrink: 0;
}

.setting-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
  background: var(--q-bg-card);
}

.setting-sidebar {
  width: 180px;
  min-width: 180px;
  border-right: 1px solid var(--q-border);
  display: flex;
  flex-direction: column;
  background: var(--q-bg-darker);
  flex-shrink: 0;
}

.sidebar-tree {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.tree-group {
  margin-bottom: 4px;
}

.tree-group-header {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  color: var(--q-text-primary);
  font-weight: 500;
  font-size: 0.9rem;
  user-select: none;
  transition: background 0.15s;

  &:hover {
    background: var(--q-bg-hover);
  }
}

.tree-arrow {
  margin-right: 4px;
  color: var(--q-text-muted);
  transition: transform 0.2s;
}

.group-label {
  flex: 1;
}

.tree-group-items {
  overflow: hidden;
}

.tree-item {
  display: flex;
  align-items: center;
  padding: 6px 12px 6px 36px;
  cursor: pointer;
  color: var(--q-text-muted);
  font-size: 0.85rem;
  transition: all 0.15s;
  border-left: 2px solid transparent;

  &:hover {
    background: var(--q-bg-hover);
    color: var(--q-text-primary);
  }

  &.active {
    color: var(--q-primary);
    background: var(--q-bg-hover);
    border-left-color: var(--q-primary);
  }
}

.expand-enter-active,
.expand-leave-active {
  transition: all 0.2s ease;
  max-height: 200px;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
}

.setting-content {
  flex: 1;
  scroll-behavior: smooth;
  margin-bottom: 60px;
  overflow-y: auto;
}

.setting-section {
  padding: 20px 24px;
  border-bottom: 1px solid var(--q-border-light);

  &:last-child {
    border-bottom: none;
  }
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--q-text-primary);
  margin: 0 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--q-border-light);
}

.setting-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid var(--q-border-light);

  &:last-child {
    border-bottom: none;
  }

  &.sub-item {
    padding-left: 24px;
    opacity: 0.9;
  }

  &.vertical {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
}

.item-info {
  flex: 1;
  min-width: 0;
}

.item-label {
  font-size: 0.95rem;
  color: var(--q-text-primary);
  margin-bottom: 2px;
  min-width: 200px;
}

.item-hint {
  font-size: 0.8rem;
  color: var(--q-text-muted);
  min-width: 200px;
}

.item-control {
  flex-shrink: 0;
  margin-right: 5%;
  min-height: 40px;
  max-width: 80vw;
  display: flex;
  height: auto;
  justify-content: flex-end;
}

.textarea {
  width: 70vw;
}

.select-control {
  width: 70vw;
}

.number-input {
  width: 70vw;
}

.port-input {
  width: 70vw;
}

.url-input {
  width: 70vw;
}

.submit-bar {
  border-top: 1px solid var(--q-border);
  background: var(--q-bg-card);
  position: absolute;
  width: 160px;
  right: 200px;
  bottom: 20px;
}

.submit-btn {
  width: 100%;
}
</style>
