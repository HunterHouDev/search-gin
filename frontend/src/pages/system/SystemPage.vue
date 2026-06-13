<template>
  <div class="q-pa-md">
    <q-tabs
      v-model="tab"
      class="q-mb-md setting-tabs"
      align="justify"
      narrow-indicator
      active-color="white"
      indicator-color="white"
      glossy
      :style="{ backgroundColor: systemProperty.theme === 'star' ? 'rgba(15, 15, 26, 0.95)' : 'var(--q-primary)' }"
    >
      <q-tab name="info" label="系统信息" />
      <q-tab name="cluster" label="集群" />
      <q-tab name="user" label="用户管理" />
      <q-tab name="log" label="系统日志" />
    </q-tabs>

    <q-tab-panels v-model="tab" animated>
      <q-tab-panel name="info">
        <q-card class="q-mb-md theme-card">
          <q-card-section>
            <h6 class="text-subtitle1 q-mb-md">功能介绍</h6>
            <div class="SystemHtml" v-html="view.settingInfo.SystemHtml"></div>
          </q-card-section>
        </q-card>
        <q-card class="theme-card">
          <q-card-section>
            <p>网络访问 : </p>
            <a :href="view.ipAddr" class="text-primary">访问： {{ view.ipAddr }}</a>
            <p>userAgent : </p>
            <p class="text-wrap">{{ userAgent }}</p>
            <p>系统信息 : </p>
           <p>{{ $q.platform.is }}</p>
          </q-card-section>
        </q-card>
      </q-tab-panel>

      <q-tab-panel name="cluster">
        <q-card class="q-mb-md theme-card">
          <q-card-section>
            <div class="row items-center q-mb-sm">
              <h6 class="text-subtitle1 q-mb-none">本机信息</h6>
              <q-space />
              <q-toggle
                v-model="cluster.clusterEnabled"
                color="green"
                label="启用集群"
                left-label
                dense
                @update:model-value="toggleLanDiscovery"
              />
            </div>
            <div class="row q-gutter-sm">
              <q-chip outline color="primary" size="md">
                <q-icon name="computer" class="q-mr-xs" />
                节点: {{ cluster.localNodeHost }}
              </q-chip>
              <q-chip outline color="secondary" size="md">
                <q-icon name="badge" class="q-mr-xs" />
                别名: {{ cluster.localNodeName }}
              </q-chip>
            </div>
          </q-card-section>
        </q-card>

        <q-card class="q-mb-md theme-card">
          <q-card-section>
            <div class="row items-center justify-between q-mb-sm">
              <h6 class="text-subtitle1 q-mb-none">在线节点 ({{ cluster.peers.length }})</h6>
              <div class="row q-gutter-sm">
                <q-btn flat dense icon="add" size="sm" color="positive" @click="showAddPeerDialog = true">
                  添加
                </q-btn>
                <q-btn flat dense icon="refresh" size="sm" color="primary" @click="fetchPeers" :loading="cluster.loading">
                  刷新
                </q-btn>
              </div>
            </div>

            <q-table
              :rows="cluster.peers"
              :columns="peerColumns"
              row-key="id"
              flat
              dense
              :pagination="{ rowsPerPage: 20 }"
              hide-pagination
              :rows-per-page-options="[0]"
            >
              <template v-slot:body-cell-status="props">
                <q-td key="status" :props="props">
                  <q-icon
                    :name="props.row._checking ? 'sync' : (props.row._alive ? 'check_circle' : 'help')"
                    :color="props.row._checking ? 'grey' : (props.row._alive ? 'positive' : 'grey')"
                    size="sm"
                  />
                </q-td>
              </template>
              <template v-slot:body-cell-actions="props">
                <q-td key="actions" :props="props">
                  <q-btn
                    flat dense
                    icon="wifi_find"
                    size="sm"
                    color="primary"
                    :loading="props.row._checking"
                    @click="checkPeer(props.row)"
                  >
                    检测连通
                  </q-btn>
                </q-td>
              </template>
            </q-table>

            <div v-if="cluster.peers.length === 0 && !cluster.loading" class="text-center q-py-md text-grey">
              暂未发现其他在线节点
            </div>
          </q-card-section>
        </q-card>
      </q-tab-panel>

      <q-tab-panel name="user">
        <div class="q-gutter-md">
          <!-- 添加用户 -->
          <q-card flat bordered class="q-pa-md theme-card">
            <q-card-section>
              <div class="text-h6 q-mb-md">添加用户</div>
              <q-input v-model="newUser.username" label="用户名" class="q-mb-sm" />
              <q-input v-model="newUser.password" label="密码" type="password" class="q-mb-sm" />
              <q-input v-model="newUser.expireDate" label="有效期（可选，YYYY-MM-DD）" class="q-mb-sm">
                <template v-slot:append>
                  <q-icon name="event" class="cursor-pointer">
                    <q-popup-proxy cover transition-show="scale" transition-hide="scale">
                      <q-date v-model="newUser.expireDate" mask="YYYY-MM-DD" today-btn />
                    </q-popup-proxy>
                  </q-icon>
                </template>
              </q-input>
              <q-btn color="primary" label="添加" @click="addUser" />
            </q-card-section>
          </q-card>

          <!-- 用户列表 -->
          <q-card flat bordered class="q-pa-md theme-card">
            <q-card-section>
              <div class="text-h6 q-mb-md">用户列表</div>
              <q-list bordered separator>
                <q-item v-for="user in userList" :key="user.username">
                  <q-item-section>
                    <q-item-label>{{ user.username }}</q-item-label>
                    <q-item-label caption v-if="user.expireDate">有效期至：{{ user.expireDate }}</q-item-label>
                    <q-item-label caption v-else>永不过期</q-item-label>
                  </q-item-section>
                  <q-item-section side>
                    <q-btn flat round icon="delete" color="negative" @click="deleteUser(user.username)" />
                  </q-item-section>
                </q-item>
              </q-list>
            </q-card-section>
          </q-card>
        </div>
      </q-tab-panel>

      <q-tab-panel name="log">
        <q-card class="theme-card">
          <q-card-section>
            <div class="log-list">
              <div v-for="(item, index) in view.logs" :key="index" class="log-item q-py-xs">
                <span class="log-time">{{ item.time?.substring(0, 19) }}</span>
                <span class="log-separator"> - </span>
                <span class="log-msg">{{ item.msg }}</span>
              </div>
            </div>
          </q-card-section>
        </q-card>
      </q-tab-panel>
    </q-tab-panels>

    <!-- 添加节点弹窗 -->
    <q-dialog v-model="showAddPeerDialog" persistent @before-show="resetPeerTest">
      <q-card style="min-width: 450px" class="theme-card">
        <q-card-section>
          <div class="text-h6 q-mb-md">手动添加在线节点</div>
          <q-input
            v-model="newPeer.ip"
            label="IP 地址"
            placeholder="例如: 192.168.1.102"
            dense
            outlined
            autofocus
            class="q-mb-sm"
          />
          <div class="row q-gutter-sm">
            <q-input
              v-model="newPeer.port"
              label="API 端口"
              placeholder="10081"
              dense
              outlined
              style="max-width: 140px"
              class="q-mb-sm"
            />
            <q-input
              v-model="newPeer.filePort"
              label="文件端口"
              placeholder="10082"
              dense
              outlined
              style="max-width: 140px"
              class="q-mb-sm"
            />
          </div>
          <div class="row items-center q-gutter-sm q-mt-sm">
            <q-btn
              dense
              outline
              :color="ipTestResult === true ? 'positive' : (ipTestResult === false ? 'negative' : 'grey')"
              :icon="peerTestStatus === 'testing' ? 'sync' : (ipTestResult === true ? 'check_circle' : (ipTestResult === false ? 'cancel' : 'computer'))"
              :loading="peerTestStatus === 'testing' && !portTestDone"
              :disable="!newPeer.ip.trim() || peerTestStatus === 'testing'"
              @click="testIPConnection"
            >
              {{ peerTestStatus === 'testing' && !portTestDone ? 'IP检测中...' : (ipTestResult === true ? 'IP可达' : (ipTestResult === false ? 'IP不可达' : '检测IP')) }}
            </q-btn>
            <q-btn
              dense
              outline
              :color="portTestResult === true ? 'positive' : (portTestResult === false ? 'negative' : 'grey')"
              :icon="peerTestStatus === 'testing' && portTestDone ? 'sync' : (portTestResult === true ? 'check_circle' : (portTestResult === false ? 'cancel' : 'router'))"
              :loading="peerTestStatus === 'testing' && portTestDone"
              :disable="!newPeer.ip.trim() || !newPeer.port.trim() || peerTestStatus === 'testing'"
              @click="testPortConnection"
            >
              {{ peerTestStatus === 'testing' && portTestDone ? '端口检测中...' : (portTestResult === true ? '端口开放' : (portTestResult === false ? '端口不通' : '检测端口')) }}
            </q-btn>
          </div>
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat label="取消" color="grey" v-close-popup @click="resetPeerTest" />
          <q-btn flat label="添加" color="primary" :disable="!newPeer.ip.trim()" @click="addManualPeer" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useQuasar } from 'quasar';
import { useRoute, useRouter } from 'vue-router';
import { GetSettingInfo, GetIpAddr, GeMemeryLog, GetLanPeers, PostSettingInfo, AddLanPeer, GetUsers, AddUser, DeleteUser, PingHost } from '../../components/api/settingAPI';
import { useSystemProperty } from '../../stores/System';

const systemProperty = useSystemProperty();
const $q = useQuasar();
const route = useRoute();
const router = useRouter();
const tab = ref((route.query.tab as string) || 'info');
const newPeer = reactive({
  ip: '',
  port: '10081',
  filePort: '10082',
});
const showAddPeerDialog = ref(false);
const peerTestStatus = ref<'idle' | 'testing' | 'done'>('idle');
const ipTestResult = ref<boolean | null>(null);
const portTestResult = ref<boolean | null>(null);
const portTestDone = ref(false);

// 用户管理
const newUser = reactive({
  username: '',
  password: '',
  expireDate: '',
});
const userList = ref<any[]>([]);

const fetchUsers = async () => {
  try {
    const res = await GetUsers();
    if (res.code === 200) {
      userList.value = res.data;
    }
  } catch (error) {
    console.error('获取用户列表失败:', error);
  }
};

const addUser = async () => {
  if (!newUser.username || !newUser.password) {
    $q.notify({ type: 'warning', message: '请填写用户名和密码' });
    return;
  }
  try {
    const res = await AddUser(newUser.username, newUser.password, newUser.expireDate);
    if (res.code === 200) {
      $q.notify({ type: 'positive', message: '添加成功' });
      newUser.username = '';
      newUser.password = '';
      newUser.expireDate = '';
      fetchUsers();
    } else {
      $q.notify({ type: 'negative', message: res.message || '添加失败' });
    }
  } catch (error) {
    $q.notify({ type: 'negative', message: '添加失败' });
    console.error(error);
  }
};

const deleteUser = async (username: string) => {
  try {
    const res = await DeleteUser(username);
    if (res.code === 200) {
      $q.notify({ type: 'positive', message: '删除成功' });
      fetchUsers();
    } else {
      $q.notify({ type: 'negative', message: res.message || '删除失败' });
    }
  } catch (error) {
    $q.notify({ type: 'negative', message: '删除失败' });
    console.error(error);
  }
};

const view = reactive({
  settingInfo: {} as any,
  ipAddr: '',
  logs: [] as any[],
});

const fetchSearch = async () => {
  const { data } = await GetSettingInfo();
  view.settingInfo = data;
  // nil/未配置 → 默认 true
  cluster.clusterEnabled = data.enableLanDiscovery !== false;
};

const userAgent = computed(() => {
  return navigator.userAgent;
});

const queryIpAddr = async () => {
  const { Code, Data } = await GetIpAddr();
  if (Code == '200') {
    view.ipAddr = `http://${Data}:${window.location.port || 10081}`;
  }
};

const fetchLogs = async () => {
  const { data } = await GeMemeryLog();
  view.logs = Array.isArray(data) ? data.reverse() : [];
};

let logIntervalId: ReturnType<typeof setInterval>;

// ── 多节点集群 ──
const cluster = reactive({
  localNodeHost: '',
  localNodeName: '',
  peers: [],
  loading: false,
  clusterEnabled: true,
});

const peerColumns = [
  { name: 'status', label: '状态', field: '_alive', align: 'center' as const, sortable: false },
  { name: 'id', label: '节点 ID', field: 'id', align: 'left' as const, sortable: true },
  { name: 'name', label: '别名', field: 'name', align: 'left' as const, sortable: true },
  { name: 'ip', label: 'IP 地址', field: 'ip', align: 'left' as const, sortable: true },
  { name: 'port', label: 'API 端口', field: 'port', align: 'left' as const, sortable: true },
  { name: 'filePort', label: '文件端口', field: 'filePort', align: 'left' as const, sortable: true,
    format: (v: any) => v || '10082' },
  { name: 'lastSeen', label: '最后心跳', field: 'lastSeen', align: 'left' as const, sortable: true,
    format: (v: any) => v ? new Date(v * 1000).toLocaleString() : '-' },
  { name: 'actions', label: '操作', field: '', align: 'center' as const, sortable: false },
];

const fetchPeers = async () => {
  cluster.loading = true;
  try {
    const res = await GetLanPeers();
    if (res) {
      cluster.localNodeHost = res.localNodeHost || '';
      cluster.localNodeName = res.localNodeName || '';
      cluster.peers = (res.peers || []).map((p: any) => ({ ...p, _alive: null, _checking: false }));
    }
  } catch (e) {
    console.error('获取集群信息失败', e);
  } finally {
    cluster.loading = false;
  }
};

const checkPeer = async (peer: any) => {
  peer._checking = true;
  peer._alive = false;
  try {
    const url = `http://${peer.ip}:${peer.port}/api/heartBeat`;
    const resp = await fetch(url, { method: 'GET', signal: AbortSignal.timeout(5000) });
    peer._alive = resp.ok;
  } catch {
    peer._alive = false;
  } finally {
    peer._checking = false;
  }
};

const testIPConnection = async () => {
  const ip = newPeer.ip.trim();
  if (!ip) return;
  peerTestStatus.value = 'testing';
  portTestDone.value = false;
  ipTestResult.value = null;
  try {
    const res = await PingHost(ip);
    ipTestResult.value = res?.alive === true;
  } catch {
    ipTestResult.value = false;
  } finally {
    peerTestStatus.value = 'done';
  }
};

const testPortConnection = async () => {
  const ip = newPeer.ip.trim();
  const port = newPeer.port.trim() || '10081';
  if (!ip || !port) return;
  peerTestStatus.value = 'testing';
  portTestDone.value = true;
  portTestResult.value = null;
  try {
    const url = `http://${ip}:${port}/api/heartBeat`;
    const resp = await fetch(url, { method: 'GET', signal: AbortSignal.timeout(5000) });
    portTestResult.value = resp.ok;
  } catch {
    portTestResult.value = false;
  } finally {
    peerTestStatus.value = 'done';
  }
};

const resetPeerTest = () => {
  peerTestStatus.value = 'idle';
  ipTestResult.value = null;
  portTestResult.value = null;
  portTestDone.value = false;
};

const addManualPeer = async () => {
  const ip = newPeer.ip.trim();
  const port = newPeer.port.trim() || '10081';
  const filePort = newPeer.filePort.trim() || '10082';
  if (!ip) return;

  // 检查是否已在在线列表中
  const exists = cluster.peers.some((p: any) => p.ID === `${ip}:${port}`);
  if (exists) {
    $q.notify({ message: '节点已在线', color: 'warning', position: 'top', timeout: 2000 });
    return;
  }

  try {
    const res = await AddLanPeer(ip, port, filePort);
    if (res.success) {
      $q.notify({ message: '添加成功', color: 'positive', position: 'top', timeout: 2000 });
      newPeer.ip = '';
      newPeer.port = '10081';
      newPeer.filePort = '10082';
      showAddPeerDialog.value = false;
      await fetchPeers();
    } else {
      $q.notify({ message: res.msg || '添加失败', color: 'negative', position: 'top', timeout: 2000 });
    }
  } catch (e) {
    console.error('添加节点失败', e);
    $q.notify({ message: '添加失败', color: 'negative', position: 'top', timeout: 2000 });
  }
};

const toggleLanDiscovery = async (val: boolean) => {
  try {
    view.settingInfo.enableLanDiscovery = val;
    await PostSettingInfo(view.settingInfo);
    $q.notify({
      message: val ? '集群模式已开启' : '集群模式已关闭',
      caption: '将在下次启动时生效',
      color: 'positive',
      position: 'top',
      timeout: 3000,
    });
  } catch (e) {
    cluster.clusterEnabled = !val;
    console.error('保存集群设置失败', e);
  }
};

// tab 切换时同步到 URL query
watch(tab, (val) => {
  router.replace({ query: { ...route.query, tab: val } });
});

onMounted(() => {
  document.title = '系统信息';
  fetchSearch();
  queryIpAddr();
  fetchLogs();
  fetchPeers();
  fetchUsers();
  logIntervalId = setInterval(() => {
    fetchLogs();
  }, 5000);
});

onUnmounted(() => {
  if (logIntervalId) {
    clearInterval(logIntervalId);
  }
});
</script>
<style lang="scss" scoped>
.setting-tabs {
  border-radius: 8px 8px 0 0;

  .q-tab {
    font-weight: 500;
    letter-spacing: 0.5px;

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
  background: var(--q-bg-card);
  border: 1px solid var(--q-border);
  color: var(--q-text-primary);
}

.text-subtitle1 {
  color: var(--q-text-secondary);
}

.text-primary {
  color: var(--q-primary);
}

.text-wrap {
  word-break: break-all;
}

.SystemHtml {
  padding: 0rem;
  margin: 0;
  color: var(--q-text-primary);
}

.log-list {
  max-height: 70vh;
  overflow-y: auto;
}

.log-item {
  border-bottom: 1px solid var(--q-border-light);
  font-family: monospace;
  font-size: 0.9rem;
}

.log-time {
  color: var(--q-text-secondary);
}

.log-separator {
  color: var(--q-border);
}

.log-msg {
  color: var(--q-text-primary);
}
</style>
