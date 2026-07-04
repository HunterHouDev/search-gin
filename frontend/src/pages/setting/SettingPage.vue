<template>
  <div class="setting-page">
    <!-- 顶部 Tab -->
    <q-tabs v-model="mainTab" class="main-tabs bg-black text-white" align="justify"
      :active-color="systemProperty.theme === 'natural' ? 'green' : 'white'"
      :indicator-color="systemProperty.theme === 'natural' ? 'green' : 'white'">
      <q-tab name="search" label="搜索设置" />
      <q-tab name="network" label="网络配置" />
      <q-tab name="dict" label="数据管理" />
      <q-tab name="permission" label="用户管理" />
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
                <div class="item-hint">选择优先使用的硬件加速方案，auto 为自动选择</div>
              </div>
              <div class="item-control">
                <q-select
                  v-model="view.settingInfo.HardwareAccelMode"
                  :options="hwAccelModeOptions"
                  outlined
                  dense
                  emit-value
                  map-options
                  style="min-width: 200px"
                />
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
              <div class="item-control row items-center q-gutter-sm">
                <MutiInput v-model="view.settingInfo.TagsLib" :style="{ maxWidth: '80%' }"
                  @onchange="(arr) => (view.settingInfo.TagsLib = arr)" />
                <q-btn dense flat size="md" icon="cloud_download" color="primary" label="统计标签"
                  :loading="tagLoading" @click="openTagStatsDialog">
                  <q-tooltip>加载扫描到的标签</q-tooltip>
                </q-btn>
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
                <div class="item-hint">设置管理员登录密码，必须配置</div>
              </div>
              <div class="item-control">
                <q-input v-model="view.settingInfo.AdminPassword" type="password" dense outlined
                  placeholder="设置管理员密码" class="url-input" />
              </div>
            </div>
          </section>
        </template>

        <!-- ========== 权限管理 ========== -->
        <template v-if="mainTab === 'permission'">
          <section id="section-users" class="setting-section">
            <h3 class="section-title">账号管理</h3>
            <div class="text-caption text-grey-6 q-mb-md">管理用户账号，每个账号可分配角色</div>
            <div class="row q-gutter-sm">
              <q-card v-for="user in permView.users" :key="user.username" class="user-card" bordered flat>
                <q-card-section class="q-pa-sm">
                  <div class="row items-center no-wrap">
                    <q-icon name="account_circle" size="32px" color="primary" class="q-mr-sm" />
                    <div class="col">
                      <div class="text-body2 text-weight-medium">{{ user.username }}</div>
                      <div class="text-caption text-grey-6">
                        <template v-if="user.role">角色：{{ user.role }}</template>
                        <template v-else>无角色</template>
                      </div>
                    </div>
                    <q-btn flat round icon="edit" color="primary" size="md" @click="openEditUser(user)" />
                    <q-btn flat round icon="delete" color="negative" size="md" @click="deleteUser(user.username)" />
                  </div>
                </q-card-section>
              </q-card>
              <q-card class="user-card add-card" bordered flat @click="openAddUser">
                <q-card-section class="q-pa-sm row items-center justify-center" style="height:100%;min-height:72px;">
                  <q-icon name="add" size="28px" color="primary" />
                  <span class="text-primary q-ml-sm text-body2">添加账号</span>
                </q-card-section>
              </q-card>
            </div>
          </section>

          <!-- ========== 角色管理 ========== -->
          <section id="section-roles" class="setting-section">
            <h3 class="section-title">自定义角色</h3>
            <div class="text-caption text-grey-6 q-mb-md">创建角色模板，在新增或编辑用户时分配角色。</div>

            <div class="role-list">
              <div v-for="(role, idx) in rolesView.list" :key="role.name" class="role-card">
                <div class="role-header">
                  <div class="role-name">
                    <q-icon name="badge" class="q-mr-sm" />
                    <strong>{{ role.label || role.name }}</strong>
                    <code class="q-ml-sm text-grey-6">{{ role.name }}</code>
                  </div>
                  <div class="role-actions">
                    <q-btn dense flat icon="edit" color="primary" @click="editRole(role, idx)" />
                    <q-btn dense flat icon="delete" color="negative" @click="confirmDeleteRole(role, idx)" />
                  </div>
                </div>
                <div class="role-perms">
                  <q-chip v-for="p in rolePermLabels(role.permissions)" :key="p" dense size="md" color="primary" text-color="white">
                    {{ p }}
                  </q-chip>
                  <span v-if="!role.permissions.length" class="text-grey-5 text-caption">未配置权限</span>
                </div>
              </div>
              <q-card class="role-card add-card" bordered flat @click="openNewRoleDialog">
                <q-card-section class="row items-center justify-center" style="height:100%;min-height:80px;">
                  <q-icon name="add" size="28px" color="primary" />
                  <span class="text-primary q-ml-sm text-body2">新建角色</span>
                </q-card-section>
              </q-card>
            </div>
          </section>
        </template>

        <!-- 角色编辑对话框 -->
        <q-dialog v-model="rolesView.dialog.show" persistent>
          <q-card class="dialog-fixed-footer" style="min-width: 750px; max-width: 900px">
            <q-card-section>
              <div class="text-h6">{{ rolesView.dialog.isNew ? '新建角色' : '编辑角色' }}</div>
            </q-card-section>
            <q-card-section class="q-pt-none dialog-scroll-body">
              <q-input v-model="rolesView.dialog.name" label="角色标识 (name)" dense outlined
                :disable="!rolesView.dialog.isNew" placeholder="如：editor、viewer"
                class="q-mb-md" />
              <q-input v-model="rolesView.dialog.label" label="角色名称 (label)" dense outlined
                placeholder="如：编辑员、浏览者" class="q-mb-md" />
              <div class="text-caption text-grey-7 q-mb-sm">权限配置</div>
              <div class="perm-section">
                <h4 class="perm-group-title">菜单权限</h4>
                <div class="perm-grid">
                  <div v-for="perm in permMenuPerms" :key="perm.key" class="perm-item">
                    <q-checkbox v-model="rolesView.dialog.permissions" :val="perm.key" :label="perm.name" dense />
                    <div class="perm-desc">{{ perm.description }}</div>
                  </div>
                </div>
              </div>
              <div class="perm-section">
                <h4 class="perm-group-title">操作权限</h4>
                <div class="perm-grid">
                  <div v-for="perm in permOpPerms" :key="perm.key" class="perm-item">
                    <q-checkbox v-model="rolesView.dialog.permissions" :val="perm.key" :label="perm.name" dense />
                    <div class="perm-desc">{{ perm.description }}</div>
                  </div>
                </div>
              </div>
            </q-card-section>
            <q-card-actions align="right" class="dialog-footer-actions">
              <q-btn flat label="取消" v-close-popup />
              <q-btn color="primary" glossy :loading="rolesView.dialog.saving" @click="saveRoleDialog">
                {{ rolesView.dialog.isNew ? '创建' : '保存' }}
              </q-btn>
            </q-card-actions>
          </q-card>
        </q-dialog>

        <!-- 新增/编辑用户弹窗 -->
        <q-dialog v-model="permView.showUserDialog" persistent>
          <q-card style="min-width: 380px">
            <q-card-section class="q-pa-sm">
              <div class="text-subtitle2 q-mb-sm">{{ permView.isEditing ? '编辑用户' : '添加用户' }}</div>
              <q-input v-model="permView.userForm.username" label="用户名" class="q-mb-xs"
                :disable="permView.isEditing" autofocus />
              <q-input v-model="permView.userForm.password" label="密码" type="password"
                :hint="permView.isEditing ? '留空则不修改密码' : ''" class="q-mb-xs" />
              <q-select v-model="permView.userForm.role" :options="permView.roleOptions"
                option-value="name" option-label="label" emit-value map-options
                dense outlined clearable placeholder="选择角色（可选）" class="q-mb-xs" />
              <q-input v-model="permView.userForm.expireDate" label="有效期（可选）" class="q-mb-xs">
                <template v-slot:append>
                  <q-icon name="event" class="cursor-pointer">
                    <q-popup-proxy cover transition-show="scale" transition-hide="scale">
                      <q-date v-model="permView.userForm.expireDate" mask="YYYY-MM-DD" today-btn />
                    </q-popup-proxy>
                  </q-icon>
                </template>
              </q-input>
            </q-card-section>
            <q-card-actions align="right" class="q-pa-sm q-pt-none">
              <q-btn flat dense label="取消" color="grey" v-close-popup @click="resetUserForm" />
              <q-btn flat dense :label="permView.isEditing ? '保存' : '添加'" color="primary" @click="saveUser" />
            </q-card-actions>
          </q-card>
        </q-dialog>

        <!-- 标签统计弹窗 -->
        <q-dialog v-model="tagDialog.show" persistent>
          <q-card style="min-width: 400px; max-width: 500px">
            <q-card-section class="row items-center">
              <div class="text-h6">加载统计标签</div>
              <q-space />
              <q-btn flat dense icon="close" v-close-popup />
            </q-card-section>
            <q-card-section class="q-pt-none">
              <div class="text-caption text-grey-6 q-mb-sm">
                已扫描 {{ tagDialog.allTags.length }} 个标签，勾选要添加到标签库的标签
              </div>
              <div class="row q-gutter-sm" style="max-height: 300px; overflow-y: auto">
                <q-checkbox v-for="tag in tagDialog.allTags" :key="tag" v-model="tagDialog.checked" :val="tag"
                  :label="tag" dense class="col-5 q-mb-xs" />
              </div>
              <div v-if="tagDialog.allTags.length === 0" class="text-center text-grey-5 q-py-md">
                未扫描到标签，请先执行全量扫描
              </div>
            </q-card-section>
            <q-card-actions align="right">
              <q-btn flat label="全选" color="primary" @click="tagDialog.checked = [...tagDialog.allTags]" />
              <q-btn flat label="取消" v-close-popup />
              <q-btn color="primary" glossy label="添加到标签库"
                :disable="tagDialog.checked.length === 0"
                @click="addTagsToLib" />
            </q-card-actions>
          </q-card>
        </q-dialog>

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

<script setup lang="ts">
import { useQuasar } from 'quasar';
import { onMounted, nextTick, reactive, ref, computed, watch } from 'vue';
import {
  GetSettingInfo,
  PostSettingInfo,
  GetIpAddr,
  AppRestart,
  GetUsers,
  GetAllPermissions,
  UpdateUserRole,
  CreateRole,
  UpdateRole,
  DeleteRole,
  GetRoles,
  AddUser,
  DeleteUser,
} from '../../components/api/settingAPI';
import { TagSizeMap } from '../../components/api/homeAPI';
import MutiSelector from '../../components/MutiSelector.vue';
import MutiInput from '../../components/MutiInput.vue';
import { useSystemProperty } from '../../stores/System';
import { useRoute, useRouter } from 'vue-router';

const $q = useQuasar();
const route = useRoute();
const router = useRouter();
const contentRef = ref<HTMLElement | null>(null);
const mainTab = ref((route.query.tab as string) || 'search');
const activeSection = ref('');

const expandedGroups: Record<string, boolean> = reactive({
  search: true,
  player: true,
  enable: true,
  users: true,
  roles: true,
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

// 权限管理的导航
const permissionNavGroups = [
  {
    name: 'users',
    label: '用户列表',
    items: [{ id: 'section-users', label: '用户列表' }],
  },
  {
    name: 'roles',
    label: '自定义角色',
    items: [{ id: 'section-roles', label: '自定义角色' }],
  },
];

const currentNavGroups = computed(() => {
  switch (mainTab.value) {
    case 'search': return searchNavGroups;
    case 'dict': return dictNavGroups;
    case 'network': return networkNavGroups;
    case 'permission': return permissionNavGroups;
    default: return searchNavGroups;
  }
});

const allSectionIds = computed(() => {
  const groups = currentNavGroups.value;
  return groups.flatMap(g => g.items.map(i => i.id));
});

const toggleGroup = (name: string) => {
  expandedGroups[name] = !expandedGroups[name];
};

const scrollToSection = (id: string) => {
  router.replace({ query: { ...route.query, tab: mainTab.value, section: id } });
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

// 切换 tab 时更新 URL + 重置滚动
watch(mainTab, (val) => {
  router.replace({ query: { ...route.query, tab: val, section: undefined } });
  activeSection.value = '';
  if (contentRef.value) {
    contentRef.value.scrollTop = 0;
  }
});

// 页面加载后，如果 URL 中有 section 参数，自动滚动到该锚点
onMounted(() => {
  const section = route.query.section as string;
  if (section) {
    nextTick(() => {
      scrollToSection(section);
    });
  }
});

// ── 用户/角色状态 ──────────────────────────────────────────────────
const permView = reactive({
  users: [] as { username: string; role: string; permissions: string[]; expireDate?: string }[],
  allPerms: [] as { key: string; name: string; group: string; description: string }[],
  roleOptions: [] as { name: string; label: string }[],
  showUserDialog: false,
  isEditing: false,
  userForm: { username: '', password: '', role: '', expireDate: '' },
});

const permMenuPerms = computed(() =>
  permView.allPerms.filter(p => p.group === '菜单')
);

const permOpPerms = computed(() =>
  permView.allPerms.filter(p => p.group === '操作')
);

// ── 角色管理 ──────────────────────────────────────────────────────
const rolesView = reactive({
  list: [] as { name: string; label: string; permissions: string[] }[],
  dialog: {
    show: false,
    isNew: true,
    editIdx: -1,
    name: '',
    label: '',
    permissions: [] as string[],
    saving: false,
  },
})

const rolePermLabels = (perms: string[]) => {
  const allPerms = permView.allPerms
  return perms.map(p => {
    const found = allPerms.find(d => d.key === p)
    return found ? found.name : p
  })
}

const openNewRoleDialog = () => {
  rolesView.dialog.isNew = true
  rolesView.dialog.editIdx = -1
  rolesView.dialog.name = ''
  rolesView.dialog.label = ''
  rolesView.dialog.permissions = []
  rolesView.dialog.show = true
}

const editRole = (role: { name: string; label: string; permissions: string[] }, idx: number) => {
  rolesView.dialog.isNew = false
  rolesView.dialog.editIdx = idx
  rolesView.dialog.name = role.name
  rolesView.dialog.label = role.label
  rolesView.dialog.permissions = [...role.permissions]
  rolesView.dialog.show = true
}

const saveRoleDialog = async () => {
  const d = rolesView.dialog
  if (!d.name.trim()) {
    $q.notify({ type: 'warning', message: '角色标识不能为空', position: 'top' })
    return
  }
  if (!d.label.trim()) {
    $q.notify({ type: 'warning', message: '角色名称不能为空', position: 'top' })
    return
  }
  d.saving = true
  try {
    let res
    if (d.isNew) {
      res = await CreateRole(d.name, d.label, d.permissions)
    } else {
      res = await UpdateRole(d.name, d.label, d.permissions)
    }
    if (res?.Code === 200) {
      $q.notify({ type: 'positive', message: d.isNew ? '角色创建成功' : '角色更新成功', position: 'top' })
      d.show = false
      await fetchRoles()
      await fetchUsers()
    } else {
      $q.notify({ type: 'negative', message: res?.Message || '操作失败', position: 'top' })
    }
  } catch (e: any) {
    $q.notify({ type: 'negative', message: e?.response?.data?.message || '操作失败', position: 'top' })
  } finally {
    d.saving = false
  }
}

const confirmDeleteRole = (role: { name: string; label: string }, _idx: number) => {
  $q.dialog({
    title: '确认删除',
    message: `确定删除角色 "${role.label || role.name}" 吗？该角色下的用户将恢复为默认角色。`,
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    try {
      const res = await DeleteRole(role.name)
      if (res?.Code === 200) {
        $q.notify({ type: 'positive', message: '角色已删除', position: 'top' })
        await fetchRoles()
        await fetchUsers()
      } else {
        $q.notify({ type: 'negative', message: res?.Message || '删除失败', position: 'top' })
      }
    } catch (e: any) {
      $q.notify({ type: 'negative', message: e?.response?.data?.message || '删除失败', position: 'top' })
    }
  })
}

// ── 用户管理 ──────────────────────────────────────────────────────
const resetUserForm = () => {
  permView.userForm = { username: '', password: '', role: '', expireDate: '' }
}

const openAddUser = () => {
  permView.isEditing = false
  resetUserForm()
  permView.showUserDialog = true
}

const openEditUser = (user: { username: string; role?: string }) => {
  permView.isEditing = true
  permView.userForm = {
    username: user.username,
    password: '',
    role: user.role || '',
    expireDate: '',
  }
  permView.showUserDialog = true
}

const saveUser = async () => {
  const { username, password, role, expireDate } = permView.userForm
  if (!username) {
    $q.notify({ type: 'warning', message: '请填写用户名' })
    return
  }
  if (!permView.isEditing && !password) {
    $q.notify({ type: 'warning', message: '请填写密码' })
    return
  }

  try {
    if (permView.isEditing) {
      // 编辑模式：更新角色
      await UpdateUserRole(username, role)
      $q.notify({ type: 'positive', message: '用户已更新', position: 'top' })
    } else {
      // 新增模式：创建用户
      const res = await AddUser(username, password, expireDate, role)
      if (res?.Code !== 200 && res?.code !== 200) {
        $q.notify({ type: 'negative', message: res?.Message || res?.message || '添加失败', position: 'top' })
        return
      }
      $q.notify({ type: 'positive', message: '添加成功', position: 'top' })
    }
    permView.showUserDialog = false
    resetUserForm()
    await fetchUsers()
  } catch (e: any) {
    $q.notify({ type: 'negative', message: e?.response?.data?.message || '操作失败', position: 'top' })
  }
}

const deleteUser = async (username: string) => {
  $q.dialog({
    title: '确认删除',
    message: `确定删除用户 "${username}" 吗？`,
    cancel: true,
    persistent: true,
  }).onOk(async () => {
    try {
      const res = await DeleteUser(username)
      if (res?.Code === 200 || res?.code === 200) {
        $q.notify({ type: 'positive', message: '删除成功', position: 'top' })
        await fetchUsers()
      } else {
        $q.notify({ type: 'negative', message: res?.Message || res?.message || '删除失败', position: 'top' })
      }
    } catch (e: any) {
      $q.notify({ type: 'negative', message: e?.response?.data?.message || '删除失败', position: 'top' })
    }
  })
}

const fetchRoles = async () => {
  try {
    const res = await GetRoles()
    const data = res?.Data || res?.data
    if (Array.isArray(data)) {
      rolesView.list = data
      permView.roleOptions = data.map((r: any) => ({ name: r.name, label: r.label || r.name }))
    }
  } catch { /* ignore */ }
}

const fetchUsers = async () => {
  try {
    const res = await GetUsers()
    const data = res?.Data || res?.data
    if (Array.isArray(data)) {
      permView.users = data.filter((u: any) => u.username !== 'admin')
    }
  } catch { /* ignore */ }
}

const fetchAllPerms = async () => {
  try {
    const res = await GetAllPermissions()
    const data = res?.Data || res?.data
    if (Array.isArray(data)) {
      permView.allPerms = data
    }
  } catch { /* ignore */ }
}

// ── 标签统计弹窗 ──────────────────────────────────────────────────
const tagLoading = ref(false);
const tagDialog = reactive({
  show: false,
  allTags: [] as string[],
  checked: [] as string[],
});

const openTagStatsDialog = async () => {
  tagLoading.value = true;
  try {
    const res = await TagSizeMap();
    const data = Array.isArray(res) ? res : (res?.Data || res?.data || []);
    tagDialog.allTags = data.map((t: any) => t.Name || t.name || '').filter(Boolean).sort();
    tagDialog.checked = [];
    tagDialog.show = true;
  } catch {
    $q.notify({ type: 'negative', message: '获取标签统计失败', position: 'top' });
  } finally {
    tagLoading.value = false;
  }
};

const addTagsToLib = () => {
  const current = new Set(view.settingInfo.TagsLib || []);
  tagDialog.checked.forEach(t => current.add(t));
  view.settingInfo.TagsLib = Array.from(current).sort();
  tagDialog.show = false;
  $q.notify({ type: 'positive', message: `已添加 ${tagDialog.checked.length} 个标签`, position: 'top' });
};

const view = reactive({
  settingInfo: {
    Dirs: [] as string[],
    DirsLib: [] as string[],
    Types: [] as string[],
    VideoTypes: [] as string[],
    ImageTypes: [] as string[],
    DocsTypes: [] as string[],
    Tags: [] as string[],
    TagsLib: [] as string[],
    MovieTypes: [] as string[],
    Pages: [] as string[],
    EnableTimeScan: true,
    CutThenDelete: false,
    SystemPlayer: 'ffplay',
    SystemPlayerVolumn: '30',
    SystemPlayerWidth: '1280',
    HardwareAcceleration: false,
    HardwareAccelMode: '',
    AvailableHwAccelModes: [] as string[],
    AdminPassword: '',
    ControllerHost: '',
    FileHost: ':10082',
    BaseUrl: '',
    ImageUrl: '',
    Remark: '',
    EnableLanDiscovery: null,
    NodeName: '',
    DiscoveryPeers: [] as string[],
  },
  ipAddr: '',
});
const systemProperty = useSystemProperty();

// 硬件加速模式下拉选项（来自后端扫描的可用方案）
const hwAccelModeOptions = computed(() => {
  return (view.settingInfo.AvailableHwAccelModes || []).map((m: string) => ({
    label: m === 'auto' ? '自动选择' : m,
    value: m,
  }));
});

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
  const sortedTags: string[] = [];
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
    FileHost: ':10082',
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
  fetchUsers();
  fetchAllPerms();
  fetchRoles();
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

// ── 权限管理 ──────────────────────────────────────────────────────
.perm-section {
  margin-top: 24px;
  padding: 16px;
  background: var(--q-bg-lighter, rgba(255,255,255,0.03));
  border-radius: 8px;
  border: 1px solid var(--q-border, rgba(139,143,168,0.15));
}

.perm-group-title {
  font-size: 15px;
  font-weight: 600;
  margin: 0 0 12px 0;
  color: var(--q-text-primary);
}

.perm-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 8px;
}

.perm-item {
  padding: 6px 10px;
  border-radius: 6px;
  border: 1px solid var(--q-border, rgba(139,143,168,0.1));
  background: var(--q-bg-card);
}

.perm-desc {
  font-size: 12px;
  color: var(--q-text-secondary, rgba(139,143,168,0.7));
  margin-top: 2px;
  margin-left: 28px;
}

.perm-actions {
  margin-top: 20px;
  display: flex;
  align-items: center;
}

.user-card {
  width: 240px;
  border-radius: 12px;
  border: 1px solid rgba(0,0,0,0.12);
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
  transition: box-shadow 0.2s, border-color 0.2s;
}
.user-card:hover {
  border-color: rgba(0,0,0,0.2);
  box-shadow: 0 3px 10px rgba(0,0,0,0.15);
}

.role-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.role-card {
  flex: 1;
  min-width: 260px;
  max-width: 400px;
  border-radius: 12px;
  padding: 16px;
  border: 1px solid rgba(0,0,0,0.12);
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
  transition: box-shadow 0.2s, border-color 0.2s;
}
.role-card:hover {
  border-color: rgba(0,0,0,0.2);
  box-shadow: 0 3px 10px rgba(0,0,0,0.15);
}
.role-header {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}
.role-name {
  flex: 1;
  display: flex;
  align-items: center;
}
.role-perms {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.add-card {
  cursor: pointer;
  border-style: dashed !important;
  border-color: rgba(0,0,0,0.2) !important;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s, border-color 0.2s;
}
.add-card:hover {
  background: rgba(0,0,0,0.03);
  border-color: var(--q-primary, #1976d2) !important;
}

.dialog-fixed-footer {
  display: flex;
  flex-direction: column;
  max-height: 85vh;
}
.dialog-scroll-body {
  flex: 1;
  overflow-y: auto;
}
.dialog-footer-actions {
  flex-shrink: 0;
  border-top: 1px solid rgba(0,0,0,0.08);
}
</style>
