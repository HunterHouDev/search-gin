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
      elevated
      class="q-gutter-sm flex justify-center"
      style="
        backdrop-filter: blur(10px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        border-bottom: 1px solid var(--q-border);
      "
    >
      <q-tabs
        v-model="tab"
        style="width: 100%; position: fixed; z-index: 9"
        :style="{ backgroundColor: systemProperty.theme === 'star' ? 'rgba(15, 15, 26, 0.95)' : 'var(--q-primary)' }"
        no-caps
        glossy
        inline-label
        class="shadow-1 setting-tabs"
        active-color="white"
        indicator-color="white"
        align="justify"
      >
        <q-tab name="search" label="搜索设置" />
        <q-tab name="base" label="基础设置" />
        <q-tab name="note" label="网络设置" />
        <q-tab name="system" label="系统设置" />
        <q-tab name="user" label="用户管理" />
      </q-tabs>
    </q-header>
    <q-page-container class="scroll" style="margin-top: 4rem">
      
      <q-tab-panels v-model="tab" animated>
        <q-tab-panel name="search">
          <q-field color="primary" label="定时扫描" stack-label>
            <template v-slot:control>
              <div class="row q-gutter-md">
                <q-radio
                  v-model="view.settingInfo.EnableTimeScan"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="true"
                  label="开启"
                />
                <q-radio
                  v-model="view.settingInfo.EnableTimeScan"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="false"
                  label="关闭"
                />
              </div>
            </template>
          </q-field>
          <q-field color="primary" label="转码删除原文件" stack-label>
            <template v-slot:control>
              <div class="row q-gutter-md">
                <q-radio
                  v-model="view.settingInfo.CutThenDelete"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="true"
                  label="是"
                />
                <q-radio
                  v-model="view.settingInfo.CutThenDelete"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="false"
                  label="否"
                />
              </div>
            </template>
          </q-field>
          
          <q-field color="primary" label="系统播放" stack-label>
            <template v-slot:control>
              <div class="row q-gutter-md">
                <q-radio
                  v-model="view.settingInfo.SystemPlayer"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  val="ffplay"
                  label="ffplay"
                />
                <q-radio
                  v-model="view.settingInfo.SystemPlayer"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  val="system"
                  label="system"
                />
              </div>
            </template>
          </q-field>
          
          <q-field color="primary" label="硬件加速编码" stack-label hint="开启后H264/H265转码将调用GPU硬件加速">
            <template v-slot:control>
              <div class="row q-gutter-md items-center">
                <q-radio
                  v-model="view.settingInfo.HardwareAcceleration"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="true"
                  label="开启"
                />
                <q-radio
                  v-model="view.settingInfo.HardwareAcceleration"
                  checked-icon="task_alt"
                  unchecked-icon="panorama_fish_eye"
                  :val="false"
                  label="关闭"
                />
                <span v-if="view.settingInfo.HardwareAcceleration && view.settingInfo.HardwareAccelMode" class="text-caption text-positive" style="margin-left: 8px;">
                  当前: {{ view.settingInfo.HardwareAccelMode }}
                </span>
                <span v-else-if="view.settingInfo.HardwareAcceleration" class="text-caption text-warning" style="margin-left: 8px;">
                  首次转码时自动检测
                </span>
              </div>
            </template>
          </q-field>

          <q-field color="primary" label="Buttons" stack-label>
            <template v-slot:control>
              <MutiSelector
                v-bind:model-value="view.settingInfo.Buttons"
                :options="buttonEnum"
                @onchange="(arr) => (view.settingInfo.Buttons = arr)"
              />
            </template>
          </q-field>

          <q-field color="primary" label="Dirs" stack-label>
            <template v-slot:control>
              <MutiSelector
                v-bind:model-value="view.settingInfo.Dirs"
                :options="view.settingInfo.DirsLib"
                @onchange="(arr) => (view.settingInfo.Dirs = arr)"
              />
            </template>
          </q-field>
          <q-field color="primary" label="MovieTypes" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.MovieTypes"
                @onchange="(arr) => (view.settingInfo.MovieTypes = arr)"
              />
            </template>
          </q-field>
          <q-field color="primary" label="VideoTypes" stack-label>
            <template v-slot:control>
              <MutiSelector
                v-bind:model-value="view.settingInfo.VideoTypes"
                :options="view.settingInfo.Types"
                @onchange="(arr) => (view.settingInfo.VideoTypes = arr)"
              />
            </template>
          </q-field>
          <q-field color="primary" label="Tags" stack-label>
            <template v-slot:control>
              <MutiSelector
                v-bind:model-value="view.settingInfo.Tags"
                :options="view.settingInfo.TagsLib"
                @onchange="(arr) => (view.settingInfo.Tags = arr)"
              />
            </template>
          </q-field>
        </q-tab-panel>

        <q-tab-panel name="base">
          <q-input
            v-model="view.settingInfo.SystemPlayerVolumn"
            :max="100"
            :min="0"
            type="number"
            label="系统播放器音量"
          />
          <q-input
            v-model="view.settingInfo.SystemPlayerWidth"
            label="系统播放器宽度"
          />
          <q-input
            v-model="view.settingInfo.ControllerHost"
            label="ControllerHost"
          />
          <q-input v-model="view.settingInfo.ImageHost" label="ImageHost" />
          <q-input v-model="view.settingInfo.StreamHost" label="StreamHost" />
          <q-field color="primary" label="DirsLib" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.DirsLib"
                @onchange="(arr) => (view.settingInfo.DirsLib = arr)"
              />
            </template>
          </q-field>
          <q-field color="primary" label="TagsLib" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.TagsLib"
                @onchange="(arr) => (view.settingInfo.TagsLib = arr)"
              />
            </template>
          </q-field>
          <q-field color="primary" label="Types" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.Types"
                @onchange="(arr) => (view.settingInfo.Types = arr)"
              />
            </template>
          </q-field>
          <q-field color="primary" label="Pages" stack-label>
            <template v-slot:control>
              <MutiInput
                v-model="view.settingInfo.Pages"
                @onchange="(arr) => (view.settingInfo.Pages = arr)"
              />
            </template>
          </q-field>
        </q-tab-panel>
        <q-tab-panel name="note">
          <q-input v-model="view.settingInfo.BaseUrl" label="BaseUrl" />
          <q-input v-model="view.settingInfo.ImageUrl" label="ImageUrl" />
          <q-input v-model="view.settingInfo.OMUrl" label="OMUrl" />
          <q-input
            type="textarea"
            autogrow
            v-model="view.settingInfo.Remark"
            label="Remark"
          />
        </q-tab-panel>

        <q-tab-panel name="system">
          <q-editor
            v-model="view.settingInfo.SystemHtml"
            :dense="$q.screen.lt.md"
            :toolbar="[
              [
                {
                  label: $q.lang.editor.align,
                  icon: $q.iconSet.editor.align,
                  fixedLabel: true,
                  list: 'only-icons',
                  options: ['left', 'center', 'right', 'justify'],
                },
                {
                  label: $q.lang.editor.align,
                  icon: $q.iconSet.editor.align,
                  fixedLabel: true,
                  options: ['left', 'center', 'right', 'justify'],
                },
              ],
              [
                'bold',
                'italic',
                'strike',
                'underline',
                'subscript',
                'superscript',
              ],
              ['token', 'hr', 'link', 'custom_btn'],
              ['print', 'fullscreen'],
              [
                {
                  label: $q.lang.editor.formatting,
                  icon: $q.iconSet.editor.formatting,
                  list: 'no-icons',
                  options: ['p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'code'],
                },
                {
                  label: $q.lang.editor.fontSize,
                  icon: $q.iconSet.editor.fontSize,
                  fixedLabel: true,
                  fixedIcon: true,
                  list: 'no-icons',
                  options: [
                    'size-1',
                    'size-2',
                    'size-3',
                    'size-4',
                    'size-5',
                    'size-6',
                    'size-7',
                  ],
                },
                {
                  label: $q.lang.editor.defaultFont,
                  icon: $q.iconSet.editor.font,
                  fixedIcon: true,
                  list: 'no-icons',
                  options: [
                    'default_font',
                    'arial',
                    'arial_black',
                    'comic_sans',
                    'courier_new',
                    'impact',
                    'lucida_grande',
                    'times_new_roman',
                    'verdana',
                  ],
                },
                'removeFormat',
              ],
              ['quote', 'unordered', 'ordered', 'outdent', 'indent'],

              ['undo', 'redo'],
              ['viewsource'],
            ]"
            :fonts="{
              arial: 'Arial',
              arial_black: 'Arial Black',
              comic_sans: 'Comic Sans MS',
              courier_new: 'Courier New',
              impact: 'Impact',
              lucida_grande: 'Lucida Grande',
              times_new_roman: 'Times New Roman',
              verdana: 'Verdana',
            }"
          />
        </q-tab-panel>

        <!-- 用户管理面板 -->
        <q-tab-panel name="user">
          <div class="q-gutter-md">
            <!-- 修改密码卡片 -->
            <q-card flat bordered class="q-pa-md">
              <q-card-section>
                <div class="text-h6 q-mb-md">
                  <q-icon name="lock" class="q-mr-sm" />
                  修改密码
                </div>
              </q-card-section>
              <q-card-section>
                <q-input
                  v-model="passwordForm.oldPassword"
                  label="当前密码"
                  type="password"
                  class="q-mb-md"
                  :rules="[val => !!val || '请输入当前密码']"
                />
                <q-input
                  v-model="passwordForm.newPassword"
                  label="新密码"
                  type="password"
                  class="q-mb-md"
                  :rules="[val => !!val || '请输入新密码']"
                />
                <q-input
                  v-model="passwordForm.confirmPassword"
                  label="确认新密码"
                  type="password"
                  class="q-mb-md"
                  :rules="[
                    val => !!val || '请确认新密码',
                    val => val === passwordForm.newPassword || '两次密码不一致'
                  ]"
                />
                <q-btn
                  color="primary"
                  label="修改密码"
                  @click="changePassword"
                  :loading="loading"
                />
              </q-card-section>
            </q-card>

            <!-- 用户列表卡片（仅超管可见） -->
            <q-card v-if="isSuperAdmin" flat bordered class="q-pa-md">
              <q-card-section>
                <div class="text-h6 q-mb-md">
                  <q-icon name="people" class="q-mr-sm" />
                  用户管理
                </div>
              </q-card-section>
              <q-card-section>
                <!-- 添加用户表单 -->
                <div class="q-mb-lg">
                  <div class="text-subtitle1 q-mb-sm">添加新用户</div>
                  <q-input
                    v-model="newUser.username"
                    label="用户名"
                    class="q-mb-sm"
                  />
                  <q-input
                    v-model="newUser.password"
                    label="密码"
                    type="password"
                    class="q-mb-sm"
                  />
                  <q-select
                    v-model="newUser.role"
                    :options="['user', 'super_admin']"
                    label="角色"
                    class="q-mb-sm"
                  />
                  <q-input
                    v-model="newUser.expireDate"
                    label="有效期（可选，格式：YYYY-MM-DD，留空永不过期）"
                    class="q-mb-sm"
                  >
                    <template v-slot:append>
                      <q-icon name="event" class="cursor-pointer">
                        <q-popup-proxy cover transition-show="scale" transition-hide="scale">
                          <q-date
                            v-model="newUser.expireDate"
                            mask="YYYY-MM-DD"
                            today-btn
                          />
                        </q-popup-proxy>
                      </q-icon>
                    </template>
                  </q-input>
                  <q-btn
                    color="primary"
                    label="添加用户"
                    @click="addUser"
                    :loading="loading"
                  />
                </div>

                <!-- 用户列表 -->
                <q-separator class="q-my-md" />
                <div class="text-subtitle1 q-mb-sm">用户列表</div>
                <q-list bordered separator>
                  <q-item v-for="user in userList" :key="user.username">
                    <q-item-section>
                      <q-item-label>{{ user.username }}</q-item-label>
                      <q-item-label caption>{{ user.role === 'super_admin' ? '超管' : '普通用户' }}</q-item-label>
                      <q-item-label caption v-if="user.expireDate">有效期至：{{ user.expireDate }}</q-item-label>
                      <q-item-label caption v-else>永不过期</q-item-label>
                    </q-item-section>
                    <q-item-section side>
                      <q-btn
                        flat
                        round
                        icon="delete"
                        color="negative"
                        @click="deleteUser(user.username)"
                        :disable="user.username === 'admin'"
                      />
                    </q-item-section>
                  </q-item>
                </q-list>
              </q-card-section>
            </q-card>
          </div>
        </q-tab-panel>
      </q-tab-panels>
    </q-page-container>
    <q-footer elevated class="glossy">
      <q-btn
        align="evenly"
        :color="systemProperty.theme === 'star' ? 'black' : 'primary'"
        glossy
        ripple
        rounded
        class="w100"
        style="height: 100%"
        @click="submitForm"
        >提...交</q-btn
      >
    </q-footer>
  </q-layout>

  <!-- <q-page-sticky position="bottom" :offset="[20, 20]">
    
  </q-page-sticky> -->
</template>

<script setup>
import { useQuasar } from 'quasar';

import { onMounted, reactive, ref, computed } from 'vue';
import {
  GetSettingInfo,
  PostSettingInfo,
  GetIpAddr,
  GetUsers,
  AddUser,
  DeleteUser,
  ChangePassword,
} from '../../components/api/settingAPI';
import MutiSelector from '../../components/MutiSelector.vue';
import MutiInput from '../../components/MutiInput.vue';
import { buttonEnum } from '../../components/model/Setting';
import { useSystemProperty } from '../../stores/System';

const $q = useQuasar();
const tab = ref('search');
const view = reactive({
  settingInfo: {
    Dirs: [],
    DirsLib: [],
    Types: [],
    VideoTypes: [],
    Tags: [],
    TagsLib: [],
    Buttons: [],
    MovieTypes: [],
    Pages: [],
    SystemHtml: '',
    HardwareAcceleration: false,
    HardwareAccelMode: '',
  },
  ipAddr: '',
});
const loading = ref(false);

const systemProperty = useSystemProperty();

// 用户管理相关
const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
});
const newUser = reactive({
  username: '',
  password: '',
  role: 'user',
  expireDate: '',
});
const userList = ref([]);

// 判断当前用户是否为超管
const isSuperAdmin = computed(() => {
  return localStorage.getItem('userRole') === 'super_admin';
});

const themeStyle = computed(() => {
  return {
    color: 'var(--q-text-primary)',
    backgroundColor: 'var(--q-bg-card)',
  };
});

const submitForm = async () => {
  view.settingInfo.Dirs = view.settingInfo.Dirs.sort();
  view.settingInfo.DirsLib = view.settingInfo.DirsLib.sort();
  view.settingInfo.Types = view.settingInfo.Types.sort();
  view.settingInfo.VideoTypes = view.settingInfo.VideoTypes.sort();
  
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
  console.log(Code, Message);
  if (Code != 200) {
    $q.notify({ message: `${Message}` });
    // window.location.reload()
  } else {
    $q.notify({ message: `${Message}` });
  }
};

const fetchSearch = async () => {
  const { data } = await GetSettingInfo();
  console.log(data);
  view.settingInfo = {
    Dirs: [],
    DirsLib: [],
    Types: [],
    VideoTypes: [],
    Tags: [],
    TagsLib: [],
    Buttons: [],
    MovieTypes: [],
    Pages: [],
    SystemHtml: '',
    HardwareAcceleration: false,
    HardwareAccelMode: '',
    ...data,
  };
};

const queryIpAddr = async () => {
  const { Code, Data } = await GetIpAddr();
  if (Code == '200') {
    view.ipAddr = `http://${Data}:10081`;
  }
};

// 获取用户列表
const fetchUsers = async () => {
  if (!isSuperAdmin.value) return;
  
  try {
    const res = await GetUsers();
    if (res.code === 200) {
      userList.value = res.data;
    }
  } catch (error) {
    console.error('获取用户列表失败:', error);
  }
};

// 修改密码
const changePassword = async () => {
  if (!passwordForm.oldPassword || !passwordForm.newPassword || !passwordForm.confirmPassword) {
    $q.notify({ type: 'warning', message: '请填写完整信息' });
    return;
  }
  
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    $q.notify({ type: 'warning', message: '两次密码不一致' });
    return;
  }
  
  loading.value = true;
  try {
    const username = localStorage.getItem('username');
    const res = await ChangePassword(username, passwordForm.oldPassword, passwordForm.newPassword);
    if (res.code === 200) {
      $q.notify({ type: 'positive', message: '密码修改成功' });
      passwordForm.oldPassword = '';
      passwordForm.newPassword = '';
      passwordForm.confirmPassword = '';
    } else {
      $q.notify({ type: 'negative', message: res.message || '密码修改失败' });
    }
  } catch (error) {
    $q.notify({ type: 'negative', message: '密码修改失败' });
    console.error('修改密码错误:', error);
  } finally {
    loading.value = false;
  }
};

// 添加用户（仅超管）
const addUser = async () => {
  if (!newUser.username || !newUser.password) {
    $q.notify({ type: 'warning', message: '请填写用户名和密码' });
    return;
  }
  
  loading.value = true;
  try {
    const res = await AddUser(newUser.username, newUser.password, newUser.role, newUser.expireDate);
    if (res.code === 200) {
      $q.notify({ type: 'positive', message: '用户添加成功' });
      newUser.username = '';
      newUser.password = '';
      newUser.role = 'user';
      newUser.expireDate = '';
      fetchUsers(); // 刷新用户列表
    } else {
      $q.notify({ type: 'negative', message: res.message || '添加用户失败' });
    }
  } catch (error) {
    $q.notify({ type: 'negative', message: '添加用户失败' });
    console.error('添加用户错误:', error);
  } finally {
    loading.value = false;
  }
};

// 删除用户（仅超管）
const deleteUser = async (username) => {
  if (username === 'admin') {
    $q.notify({ type: 'warning', message: '不能删除默认超管账户' });
    return;
  }
  
  loading.value = true;
  try {
    const res = await DeleteUser(username);
    if (res.code === 200) {
      $q.notify({ type: 'positive', message: '用户删除成功' });
      fetchUsers(); // 刷新用户列表
    } else {
      $q.notify({ type: 'negative', message: res.message || '删除用户失败' });
    }
  } catch (error) {
    $q.notify({ type: 'negative', message: '删除用户失败' });
    console.error('删除用户错误:', error);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  document.title = '设置';
  fetchSearch();
  queryIpAddr();
  fetchUsers();
});
</script>
<style lang="scss" scoped>
.setting-tabs {
  .q-tab {
    font-weight: 500;
    letter-spacing: 0.5px;
    transition: all 0.3s ease;

    &--active {
      font-weight: 600;
    }
  }

  :deep(.q-tab__indicator) {
    height: 3px;
    border-radius: 3px 3px 0 0;
  }
}

.theme-card {
  transition: all 0.3s ease;
  border: 2px solid var(--q-border);
  background: var(--q-bg-card);

  &:hover {
    border-color: var(--q-primary);
    transform: translateY(-2px);
    box-shadow: var(--q-shadow);
  }

  &.theme-card-active {
    border-color: var(--q-primary);
    box-shadow: 0 0 0 2px var(--q-primary-light);
    background: rgba(99, 102, 241, 0.08);
  }
}

.cursor-pointer {
  cursor: pointer;
}

:deep(.q-tab-panel) {
  background: transparent;
  padding: 16px 8px;
}

:deep(.q-field__label) {
  color: var(--q-primary);
  font-weight: 500;
}

:deep(.q-field__native) {
  color: var(--q-text-primary);
}

:deep(.q-radio__label) {
  color: var(--q-text-primary);
}

:deep(.q-input) {
  .q-field__control {
    background: var(--q-bg-input);
    border-radius: 8px;

    &:before {
      border-color: var(--q-border);
    }

    &:hover:before {
      border-color: var(--q-border-hover);
    }
  }

  &.q-field--focused .q-field__control {
    border-color: var(--q-primary);
    box-shadow: 0 0 0 2px var(--q-primary-light);
  }
}

:deep(.q-editor) {
  background: var(--q-bg-input);
  border: 1px solid var(--q-border);
  border-radius: 8px;

  .q-editor__toolbar {
    background: var(--q-bg-card);
    border-bottom: 1px solid var(--q-border);
  }

  .q-editor__content {
    color: var(--q-text-primary);
  }
}

:deep(.q-tab-panels) {
  background: transparent;
}

:deep(.q-footer) {
  background: var(--q-bg-card);
  border-top: 1px solid var(--q-border);
}

.theme-selection-card {
  background: var(--q-bg-card);
  border: 1px solid var(--q-border);
  border-radius: 12px;
  transition: all 0.3s ease;

  .text-h6 {
    color: var(--q-text-primary);
    display: flex;
    align-items: center;
  }
}

:deep(.q-card.theme-card) {
  background: var(--q-bg-input);
  border: 2px solid var(--q-border);
  border-radius: 12px;
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--q-primary);
    transform: translateY(-4px);
    box-shadow: var(--q-shadow);
  }

  &.theme-card-active {
    border-color: var(--q-primary);
    box-shadow: 0 0 0 3px var(--q-primary-light), var(--q-shadow);
    background: rgba(99, 102, 241, 0.1);
  }

  .text-subtitle1 {
    font-weight: 600;
    color: var(--q-text-primary);
  }

  .text-caption {
    color: var(--q-text-muted);
  }
}
</style>
