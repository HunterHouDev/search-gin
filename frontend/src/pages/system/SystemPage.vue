<template>
  <div class="q-pa-sm">
    <q-tabs
      v-model="tab"
      class="q-mb-xs bg-black text-white"
      align="justify"
      :active-color="systemProperty.theme === 'natural' ? 'green' : 'white'"
      :indicator-color="systemProperty.theme === 'natural' ? 'green' : 'white'"
    >
      <q-tab name="info" label="系统信息" />
      <q-tab name="cluster" label="集群信息" />
      <q-tab name="user" label="用户管理" />
      <q-tab name="log" label="系统日志" />
    </q-tabs>

    <q-tab-panels v-model="tab" animated>
      <q-tab-panel name="info" class="q-pa-xs">
        <q-card class="">
          <q-card-section class="q-pa-sm">
            <div class="text-caption">网络访问：<a :href="view.ipAddr" class="text-primary">{{ view.ipAddr }}</a></div>
            <div class="text-caption text-wrap">userAgent：{{ userAgent }}</div>
            <div class="text-caption">{{ $q.platform.is }}</div>
          </q-card-section>
        </q-card>
      </q-tab-panel>

      <q-tab-panel name="cluster" class="q-pa-xs">
        <q-card class="q-mb-sm ">
          <q-card-section class="q-pa-sm">
            <div class="row items-center">
              <div class="text-subtitle2">本机信息</div>
              <q-space />
              <q-toggle
                v-model="cluster.clusterEnabled"
                color="green"
                label="启用集群"
                left-label
                @update:model-value="toggleCluster"
              />
            </div>
            <div class="row q-gutter-xs">
              <q-chip outline color="blue"  >
               <q-icon name="badge" class="q-mr-xs" size="14px" />
                别名: {{ cluster.localNodeName }}
              </q-chip>
              <q-chip outline color="primary"  >
                <q-icon name="computer" class="q-mr-xs" size="14px" />
                节点: {{ cluster.localNodeHost }}
              </q-chip>
            </div>
          </q-card-section>
        </q-card>

        <q-card class="q-pa-xs">
          <q-card-section class="q-pa-sm">
              <div class="row items-center justify-between q-mb-xs">
                <div class="text-subtitle2">在线节点 ({{ cluster.peers.length }})</div>
                <div class="row q-gutter-xs">
                  <q-btn flat  icon="refresh"  color="primary" @click="fetchPeers" :loading="cluster.loading">刷新</q-btn>
                </div>
              </div>

              <!-- 发现/添加节点：输入子网前缀扫描 /24，或输入完整 IP 单机检测 -->
              <div class="row items-center q-gutter-xs q-mb-sm">
                <q-input dense outlined v-model="cluster.discoverInput" label="IP / 子网前缀"
                  placeholder="如 192.168.1 或 192.168.1.50" style="max-width: 220px"
                  :disable="cluster.discovering" @keyup.enter="discoverPeers" />
                <q-btn flat icon="wifi_find" color="info" @click="discoverPeers"
                  :loading="cluster.discovering" :disable="!cluster.discoverInput">发现</q-btn>
              </div>

              <!-- 发现节点列表 -->
              <q-slide-transition>
                <div v-if="cluster.discovered.length > 0" class="q-mb-sm">
                  <q-card flat bordered class="bg-blue-1">
                    <q-card-section class="q-pa-sm">
                      <div class="row items-center q-mb-xs">
                        <div class="text-caption text-weight-bold text-primary">已发现 {{ cluster.discovered.length }} 个节点</div>
                        <q-space />
                        <q-btn flat dense icon="close" size="sm" @click="cluster.discovered = []" />
                      </div>
                      <div v-for="d in cluster.discovered" :key="d.ip" class="row items-center q-gutter-xs q-mb-xs">
                        <q-chip outline color="primary" size="sm" icon="computer">
                          {{ d.nodeName || d.ip }}
                        </q-chip>
                        <span class="text-caption text-grey">{{ d.ip }}:{{ d.port }}</span>
                        <q-space />
                        <q-btn v-if="!d._existing" dense flat size="sm" color="positive" icon="add" @click="addDiscoveredPeer(d)"
                          :disable="d._adding" :loading="d._adding">添加</q-btn>
                        <q-chip v-else dense outline color="grey" size="sm" icon="check">已存在</q-chip>
                      </div>
                      <div v-if="cluster.discovered.length === 0" class="text-caption text-grey q-py-xs text-center">
                        未发现节点
                      </div>
                    </q-card-section>
                  </q-card>
                </div>
              </q-slide-transition>

            <q-table
              :rows="cluster.peers"
              :columns="peerColumns"
              row-key="id"
              flat

              :pagination="{ rowsPerPage: 20 }"
              hide-pagination
              :rows-per-page-options="[0]"
              class="compact-table"
            >
              <template v-slot:body-cell-status="props">
                <q-td key="status" :props="props">
                  <q-icon
                    v-if="props.row.disabled"
                    name="block"
                    color="negative"
                    size="xs"
                  />
                  <q-icon
                    v-else
                    :name="props.row._checking ? 'sync' : (props.row._alive ? 'check_circle' : 'help')"
                    :color="props.row._checking ? 'grey' : (props.row._alive ? 'positive' : 'grey')"
                    size="xs"
                  />
                </q-td>
              </template>
              <template v-slot:body-cell-actions="props">
                <q-td key="actions" :props="props" class="q-pa-xs">
                  <q-btn flat  icon="wifi_find"  color="primary"
                    :loading="props.row._checking" @click="checkPeer(props.row)" class="q-mr-xs">检测连通</q-btn>
                  <q-btn flat  :icon="props.row.disabled ? 'play_arrow' : 'pause'"
                    :color="props.row.disabled ? 'positive' : 'warning'"
                    @click="togglePeer(props.row)" class="q-mr-xs">{{ props.row.disabled ? '启用' : '禁用' }}</q-btn>
                  <q-btn flat  icon="delete"  color="negative"
                    @click="removePeer(props.row)">删除</q-btn>
                </q-td>
              </template>
            </q-table>

            <div v-if="cluster.peers.length === 0 && !cluster.loading" class="text-center q-py-sm text-grey">
              <div class="text-caption q-mb-xs">暂未发现其他在线节点</div>
              <div class="text-caption">
                <div v-if="!cluster.clusterEnabled" class="text-warning">⚠ 集群模式已关闭，请在「本机信息」中开启</div>
                <div v-else class="text-info">
                    集群已启用，可通过子网扫描发现节点<br>
                    • 确保其他节点也已启用集群<br>
                    • 所有节点需在同一网络可达<br>
                    • 在输入框中输入子网前缀（如 192.168.1）点击「发现」
                  </div>
              </div>
            </div>
          </q-card-section>
        </q-card>
      </q-tab-panel>

      <q-tab-panel name="user" class="q-pa-xs">
        <q-card flat bordered class="q-pa-xs">
          <q-card-section class="q-pa-sm">
            <div class="row items-center justify-between q-mb-xs">
              <div class="text-subtitle2">用户列表</div>
              <q-btn flat dense color="primary" icon="person_add" size="sm" @click="showAddUserDialog = true">添加用户</q-btn>
            </div>
            <q-list bordered separator>
              <q-item v-for="user in userList" :key="user.username" class="q-py-xs">
                <q-item-section>
                  <q-item-label class="text-caption">{{ user.username }}</q-item-label>
                  <q-item-label caption v-if="user.expireDate" class="text-caption">有效期至：{{ user.expireDate }}</q-item-label>
                  <q-item-label caption v-else class="text-caption">永不过期</q-item-label>
                </q-item-section>
                <q-item-section side>
                  <q-btn flat round icon="delete" color="negative" size="sm" @click="deleteUser(user.username)" />
                </q-item-section>
              </q-item>
              <div v-if="userList.length === 0" class="text-center text-grey q-py-md">
                <q-icon name="group_off" size="3em" class="q-mb-sm" />
                <div class="text-caption">暂无用户，点击右上角添加</div>
              </div>
            </q-list>
          </q-card-section>
        </q-card>

        <!-- 添加用户弹窗 -->
        <q-dialog v-model="showAddUserDialog" persistent>
          <q-card style="min-width: 360px">
            <q-card-section class="q-pa-sm">
              <div class="text-subtitle2 q-mb-sm">添加用户</div>
              <q-input v-model="newUser.username" label="用户名" class="q-mb-xs" autofocus />
              <q-input v-model="newUser.password" label="密码" type="password" class="q-mb-xs" />
              <q-input v-model="newUser.expireDate" label="有效期（可选）" class="q-mb-xs">
                <template v-slot:append>
                  <q-icon name="event" class="cursor-pointer">
                    <q-popup-proxy cover transition-show="scale" transition-hide="scale">
                      <q-date v-model="newUser.expireDate" mask="YYYY-MM-DD" today-btn />
                    </q-popup-proxy>
                  </q-icon>
                </template>
              </q-input>
            </q-card-section>
            <q-card-actions align="right" class="q-pa-sm q-pt-none">
              <q-btn flat dense label="取消" color="grey" v-close-popup @click="resetNewUser" />
              <q-btn flat dense label="添加" color="primary" @click="addUser" />
            </q-card-actions>
          </q-card>
        </q-dialog>
      </q-tab-panel>

      <q-tab-panel name="log" class="q-pa-xs">
        <q-card class="q-pa-xs">
          <q-card-section class="q-pa-sm">
            <div class="row items-center q-gutter-sm q-mb-sm">
              <q-btn-toggle
                v-model="logTab"
                :options="[
                  { label: '内存日志', value: 'memory' },
                  { label: '本地日志', value: 'local' },
                ]"
                toggleTextColor="white"
                toggleColor="blue"
              />
              <q-space />
              <q-btn dense flat icon="refresh" size="sm" @click="logTab === 'memory' ? fetchMemoryLog() : fetchLocalLog()" />
            </div>
            <!-- 内存日志 -->
            <template v-if="logTab === 'memory'">
              <div class="row items-center q-gutter-sm q-mb-md">
                <q-btn-toggle
                  v-model="logTimeFilter"
                  :options="logTimeOptions"
                   flat no-caps class="q-ml-xs"
                />
                <q-select
                  v-model="logTypeFilter"
                  :options="logTypeOptions"
                   clearable placeholder="类型"
                  class="col-2" style="min-width:100px"
                />
                <q-input
                  v-model="logKeyword"  debounce="300"
                  placeholder="过滤关键词" clearable class="col-3"
                >
                  <template v-slot:prepend>
                    <q-icon name="search" />
                  </template>
                </q-input>
                <q-btn
                  :icon="logSortAsc ? 'arrow_upward' : 'arrow_downward'"
                  flat  @click="logSortAsc = !logSortAsc"
                >
                  <q-tooltip>{{ logSortAsc ? '时间正序' : '时间倒序' }}</q-tooltip>
                </q-btn>
              </div>
              <div class="log-list">
                <div v-for="item in memoryPageData" :key="item.time" class="log-item q-py-xs">
                  <span class="log-type-dot" :class="logTypeColor(logExtractType(item.msg))" />
                  <span class="log-time">{{ item.time.substring(0, 19) }}</span>
                  <span class="log-separator"> - </span>
                  <span class="log-msg">{{ simplifyLog(item.msg) }}</span>
                </div>
                <div v-if="memoryPageData.length === 0" class="text-center text-grey q-py-md">
                  暂无匹配的日志
                </div>
              </div>
              <div class="row justify-center q-mt-md" v-if="memoryTotalPages > 1">
                <q-pagination
                  v-model="memoryPage" :max="memoryTotalPages" :max-pages="7"
                  boundary-links direction-links
                />
              </div>
            </template>

            <!-- 本地日志 -->
            <template v-if="logTab === 'local'">
              <div class="log-list">
                <div v-for="(line, idx) in localPageData" :key="idx" class="log-item q-py-xs">
                  <span class="log-raw">{{ line }}</span>
                </div>
                <div v-if="localPageData.length === 0" class="text-center text-grey q-py-md">
                  暂无日志
                </div>
              </div>
              <div class="row justify-center q-mt-md" v-if="localTotalPages > 1">
                <q-pagination
                  v-model="localPage" :max="localTotalPages" :max-pages="7"
                  boundary-links direction-links
                />
              </div>
            </template>
          </q-card-section>
        </q-card>
      </q-tab-panel>
    </q-tab-panels>

  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useQuasar } from 'quasar';
import { useRoute, useRouter } from 'vue-router';
import { GetSettingInfo, GetIpAddr, GeMemeryLog, GetLocalLog, GetLanPeers, PostSettingInfo, AddLanPeer, RemoveLanPeer, TogglePeer, GetUsers, AddUser, DeleteUser, DiscoverLanPeers } from '../../components/api/settingAPI';
import { useSystemProperty } from '../../stores/System';

const systemProperty = useSystemProperty();
const $q = useQuasar();

const themeStyle = computed(() => ({
  color: 'var(--q-text-primary)',
  backgroundColor: 'var(--q-bg-dark)',
}));
const route = useRoute();
const router = useRouter();
const tab = ref((route.query.tab as string) || 'info');

// 用户管理
const showAddUserDialog = ref(false);
const newUser = reactive({
  username: '',
  password: '',
  expireDate: '',
});
const userList = ref<any[]>([]);

const resetNewUser = () => {
  newUser.username = '';
  newUser.password = '';
  newUser.expireDate = '';
};

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
      showAddUserDialog.value = false;
      resetNewUser();
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

// ── 日志 ──
const logTab = ref('memory');
const logKeyword = ref('');
const logSortAsc = ref(true);
const logTypeFilter = ref(null);
const logTimeFilter = ref('');
const logTimeOptions = [
  { label: '全部', value: '' },
  { label: '今天', value: 'today' },
  { label: '昨天', value: 'yesterday' },
  { label: '≥3天', value: 'older' },
];

interface LogItem {
  type: string;
  time: string;
  msg: string;
}

const logPageSize = 50;
const allMemoryLogs = ref([] as LogItem[]);
const memoryPage = ref(1);
const allLocalLines = ref([] as LogItem[]);
const localPage = ref(1);

function logExtractType(msg: string) {
  if (!msg) return '';
  const m = msg.match(/^[^：:　\s]+/);
  return m ? m[0] : '';
}
const logTypeColorMap: Record<string, string> = {
  '扫描': 'type-scan', '添加': 'type-add', '取消': 'type-cancel',
  '开始': 'type-scan', '完成': 'type-done', '拒绝': 'type-deny',
  '首次': 'type-join', '新节点': 'type-join', '全量': 'type-scan',
  'Plan': 'type-info', 'ScanAll': 'type-scan', '索引': 'type-info',
  '搜索': 'type-search',
};
// 简化 JSON 日志：提取 状态码 路径 数据长度
function simplifyLog(msg: string): string {
  try {
    const obj = JSON.parse(msg);
    const parts: string[] = [];
    if (obj.statusCode) parts.push(String(obj.statusCode));
    if (obj.method && obj.path) parts.push(`${obj.method} ${obj.path}`);
    else if (obj.path) parts.push(obj.path);
    if (obj.dataLength != null && obj.dataLength !== 0) parts.push(`${obj.dataLength}B`);
    if (obj.latency != null) parts.push(`${obj.latency}ms`);
    return parts.join(' ') || msg.substring(0, 120);
  } catch {
    return msg.substring(0, 120);
  }
}

function logTypeColor(t: string) {
  return logTypeColorMap[t] || 'type-default';
}

const logTypeOptions = computed(() => {
  const seen = new Set<string>();
  const types: string[] = [];
  for (const item of allMemoryLogs.value) {
    const t = logExtractType(item.msg);
    if (t && !seen.has(t)) { seen.add(t); types.push(t); }
  }
  return types.sort();
});

function getDateYMD(timeStr: string) {
  return timeStr ? timeStr.substring(0, 10) : '';
}
function todayYMD() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
function daysAgoYMD(n: number) {
  const d = new Date(); d.setDate(d.getDate() - n);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

const memoryFiltered = computed(() => {
  let list = [...allMemoryLogs.value];
  const kw = logKeyword.value?.trim().toLowerCase();
  if (kw) list = list.filter((item: any) => item.msg?.toLowerCase().includes(kw) || item.time?.includes(kw));
  if (logTypeFilter.value) list = list.filter((item: any) => logExtractType(item.msg) === logTypeFilter.value);
  if (logTimeFilter.value) {
    const today = todayYMD();
    const yesterday = daysAgoYMD(1);
    const threeDaysAgo = daysAgoYMD(3);
    list = list.filter((item: any) => {
      const d = getDateYMD(item.time);
      if (!d) return false;
      switch (logTimeFilter.value) {
        case 'today': return d === today;
        case 'yesterday': return d === yesterday;
        case 'older': return d <= threeDaysAgo;
        default: return true;
      }
    });
  }
  list.sort((a: any, b: any) => logSortAsc.value ? a.time.localeCompare(b.time) : b.time.localeCompare(a.time));
  return list;
});

const memoryTotalPages = computed(() => Math.max(1, Math.ceil(memoryFiltered.value.length / logPageSize)));
const memoryPageData = computed(() => {
  const start = (memoryPage.value - 1) * logPageSize;
  return memoryFiltered.value.slice(start, start + logPageSize);
});

watch([logKeyword, logTypeFilter, logTimeFilter, logSortAsc], () => { memoryPage.value = 1; });

const localTotalPages = computed(() => Math.max(1, Math.ceil(allLocalLines.value.length / logPageSize)));
const localPageData = computed(() => {
  const start = (localPage.value - 1) * logPageSize;
  return allLocalLines.value.slice(start, start + logPageSize);
});

async function fetchLocalLog() {
  const { data } = await GetLocalLog();
  allLocalLines.value = Array.isArray(data) ? data : [];
}
async function fetchMemoryLog() {
  const { data } = await GeMemeryLog();
  allMemoryLogs.value = Array.isArray(data) ? data : [];
}

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

let logIntervalId: ReturnType<typeof setInterval> | undefined;

// ── 多节点集群 ──
const cluster = reactive({
  localNodeHost: '',
  localNodeName: '',
  peers: [],
  loading: false,
  discovering: false,
  discovered: [] as { ip: string; port: string; filePort?: string; nodeName?: string; _adding?: boolean; _existing?: boolean }[],
  discoverInput: '',
  clusterEnabled: true,
});

const peerColumns = [
  { name: 'status', label: '状态', field: '_alive', align: 'center' as const, sortable: false },
  { name: 'actions', label: '操作', field: '', align: 'center' as const, sortable: false },
  { name: 'id', label: '节点 ID', field: 'id', align: 'left' as const, sortable: true },
  { name: 'name', label: '别名', field: 'name', align: 'left' as const, sortable: true },
  { name: 'ip', label: 'IP 地址', field: 'ip', align: 'left' as const, sortable: true },
  { name: 'port', label: 'API 端口', field: 'port', align: 'left' as const, sortable: true },
  { name: 'filePort', label: '文件端口', field: 'filePort', align: 'left' as const, sortable: true,
    format: (v: any) => v || '10082' },
  { name: 'lastSeen', label: '最后心跳', field: 'lastSeen', align: 'left' as const, sortable: true,
    format: (v: any) => v ? new Date(v * 1000).toLocaleString() : '-' },
];

const fetchPeers = async () => {
  cluster.loading = true;
  try {
    const res = await GetLanPeers();
    if (res) {
      cluster.localNodeHost = res.localNodeHost || '';
      cluster.localNodeName = res.localNodeName || '';
      cluster.peers = (res.peers || []).map((p: any) => ({ ...p, _alive: null, _checking: false }));
      if (res.localSubnet && !cluster.discoverInput) {
        cluster.discoverInput = res.localSubnet;
      }
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

// 发现节点：输入子网前缀扫 /24，或完整 IP 单机检测
const discoverPeers = async () => {
  const input = cluster.discoverInput.trim();
  if (!input) return;

  cluster.discovering = true;
  cluster.discovered = [];
  try {
    const res = await DiscoverLanPeers(input);
    const peersList = res.peers || [];
    if (res.success && Array.isArray(peersList)) {
      const existingIds = new Set(cluster.peers.map((p: any) => p.id));
      cluster.discovered = peersList.map((d: any) => ({
        ...d,
        _adding: false,
        _existing: existingIds.has(`${d.ip}:${d.port}`),
      }));
    }
  } catch (e) {
    $q.notify({ message: '发现节点失败', color: 'negative', position: 'top', timeout: 2000 });
  } finally {
    cluster.discovering = false;
  }
};

const addDiscoveredPeer = async (d: any) => {
  d._adding = true;
  try {
    const res = await AddLanPeer(d.ip, d.port, d.filePort || '10082');
    if (res.success) {
      $q.notify({ message: `已添加 ${d.ip}`, color: 'positive', position: 'top', timeout: 2000 });
      d._existing = true;
      await fetchPeers();
    } else {
      $q.notify({ message: res.msg || '添加失败', color: 'negative', position: 'top', timeout: 2000 });
    }
  } catch {
    $q.notify({ message: '添加失败', color: 'negative', position: 'top', timeout: 2000 });
  } finally {
    d._adding = false;
  }
};

const toggleCluster = async (val: boolean) => {
  try {
    view.settingInfo.enableLanDiscovery = val;
    await PostSettingInfo(view.settingInfo);
    cluster.clusterEnabled = val;
    $q.notify({
      message: val ? '集群模式已开启' : '集群模式已关闭',
      color: 'positive',
      position: 'top',
      timeout: 3000,
    });
  } catch (e) {
    cluster.clusterEnabled = !val;
    console.error('切换集群模式失败', e);
  }
};

const togglePeer = async (peer: any) => {
  const newDisabled = !peer.disabled;
  try {
    const res = await TogglePeer(peer.id, newDisabled);
    if (res?.success) {
      peer.disabled = newDisabled;
      $q.notify({
        message: newDisabled ? '节点已禁用，搜索将跳过' : '节点已启用',
        color: 'positive',
        position: 'top',
        timeout: 2000,
      });
    }
  } catch (e) {
    console.error('切换节点状态失败', e);
  }
};

const removePeer = async (peer: any) => {
  try {
    const res = await RemoveLanPeer(peer.id);
    if (res?.success) {
      $q.notify({ message: '节点已删除', color: 'positive', position: 'top', timeout: 2000 });
      await fetchPeers();
    } else {
      $q.notify({ message: res?.msg || '删除失败', color: 'negative', position: 'top', timeout: 2000 });
    }
  } catch (e) {
    console.error('删除节点失败', e);
    $q.notify({ message: '删除失败', color: 'negative', position: 'top', timeout: 2000 });
  }
};

// tab 切换时同步到 URL query，日志 tab 开启定时刷新
watch(tab, (val) => {
  router.replace({ query: { ...route.query, tab: val } });
  if (val === 'log') {
    logIntervalId = setInterval(() => {
      if (logTab.value === 'memory') fetchMemoryLog();
      else fetchLocalLog();
    }, 5000);
  } else {
    if (logIntervalId) { clearInterval(logIntervalId); logIntervalId = undefined; }
  }
});

onMounted(() => {
  document.title = '系统信息';
  fetchSearch();
  queryIpAddr();
  fetchMemoryLog();
  fetchLocalLog();
  fetchPeers();
  fetchUsers();
  // 如果初始 tab 就是日志，启动定时轮询
  if (tab.value === 'log') {
    logIntervalId = setInterval(() => {
      if (logTab.value === 'memory') fetchMemoryLog();
      else fetchLocalLog();
    }, 5000);
  }
});

onUnmounted(() => {
  if (logIntervalId) {
    clearInterval(logIntervalId);
  }
});
</script>
<style lang="scss" scoped>

.text-wrap {
  word-break: break-all;
}

.log-list {
  max-height: 70vh;
  overflow-y: auto;
}

.log-item {
  border-bottom: 1px solid var(--q-border-light);
  font-family: monospace;
  font-size: 0.9rem;
  display: flex;
  align-items: center;
}

.log-time {
  color: var(--q-text-primary);
  flex-shrink: 0;
}

.log-separator {
  color: var(--q-border);
  flex-shrink: 0;
}

.log-msg {
  color: var(--q-text-primary);
}

.log-raw {
  color: var(--q-text-primary);
  white-space: pre-wrap;
  word-break: break-all;
}

.text-subtitle1 {
  color: var(--q-text-secondary);
}

/* 类型颜色圆点 */
.log-type-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
  flex-shrink: 0;
}
.type-scan   { background: #42a5f5; }
.type-add    { background: #66bb6a; }
.type-cancel { background: #ef5350; }
.type-done   { background: #26a69a; }
.type-deny   { background: #ff7043; }
.type-join   { background: #ab47bc; }
.type-info   { background: #78909c; }
.type-search { background: #ffca28; }
.type-default { background: #90a4ae; }

:deep(.compact-table) {
  .q-table__top,
  .q-table__bottom {
    display: none;
  }
  td {
    padding: 2px 4px !important;
    font-size: 0.8rem;
  }
  th {
    padding: 2px 4px !important;
    font-size: 0.8rem;
    font-weight: 600;
  }
}
</style>
