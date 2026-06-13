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
            <h6 class="text-subtitle1 q-mb-sm">手动添加在线节点</h6>
            <div class="row q-gutter-sm">
              <q-input
                v-model="newPeerInput"
                placeholder="例如: 192.168.1.102:10081"
                dense
                outlined
                style="max-width: 250px"
                @keyup.enter="addManualPeer"
              />
              <q-btn color="primary" dense icon="add" @click="addManualPeer" :disable="!newPeerInput.trim()">
                添加
              </q-btn>
            </div>
            <p class="text-grey text-caption q-mt-sm">
              添加后会自动验证节点是否可达，成功后显示在在线节点列表中
            </p>
          </q-card-section>
        </q-card>

        <q-card class="q-mb-md theme-card">
          <q-card-section>
            <div class="row items-center justify-between q-mb-sm">
              <h6 class="text-subtitle1 q-mb-none">在线节点 ({{ cluster.peers.length }})</h6>
              <q-btn flat dense icon="refresh" size="sm" color="primary" @click="fetchPeers" :loading="cluster.loading">
                刷新
              </q-btn>
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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
import { useQuasar } from 'quasar';
import { GetSettingInfo, GetIpAddr, GeMemeryLog, GetLanPeers, PostSettingInfo, AddLanPeer } from '../../components/api/settingAPI';
import { useSystemProperty } from '../../stores/System';

const systemProperty = useSystemProperty();
const $q = useQuasar();
const tab = ref('info');
const newPeerInput = ref('');
const view = reactive({
  settingInfo: {} as any,
  ipAddr: '',
  logs: [],
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

let logIntervalId;

// ── 多节点集群 ──
const cluster = reactive({
  localNodeHost: '',
  localNodeName: '',
  peers: [],
  loading: false,
  clusterEnabled: true,
});

const peerColumns = [
  { name: 'status', label: '状态', field: '_alive', align: 'center', sortable: false },
  { name: 'id', label: '节点 ID', field: 'id', align: 'left', sortable: true },
  { name: 'name', label: '别名', field: 'name', align: 'left', sortable: true },
  { name: 'ip', label: 'IP 地址', field: 'ip', align: 'left', sortable: true },
  { name: 'lastSeen', label: '最后心跳', field: 'lastSeen', align: 'left', sortable: true,
    format: (v) => v ? new Date(v * 1000).toLocaleString() : '-' },
  { name: 'actions', label: '操作', field: '', align: 'center', sortable: false },
];

const fetchPeers = async () => {
  cluster.loading = true;
  try {
    const res = await GetLanPeers();
    if (res) {
      cluster.localNodeHost = res.localNodeHost || '';
      cluster.localNodeName = res.localNodeName || '';
      cluster.peers = (res.peers || []).map(p => ({ ...p, _alive: null, _checking: false }));
    }
  } catch (e) {
    console.error('获取集群信息失败', e);
  } finally {
    cluster.loading = false;
  }
};

const checkPeer = async (peer) => {
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

const addManualPeer = async () => {
  const addr = newPeerInput.value.trim();
  if (!addr) return;

  // 检查是否已在在线列表中
  const exists = cluster.peers.some(p => p.ID === addr || `${p.IP}:${p.Port}` === addr);
  if (exists) {
    $q.notify({ message: '节点已在线', color: 'warning', position: 'top', timeout: 2000 });
    return;
  }

  try {
    const res = await AddLanPeer(addr);
    if (res.success) {
      $q.notify({ message: '添加成功', color: 'positive', position: 'top', timeout: 2000 });
      newPeerInput.value = '';
      await fetchPeers(); // 刷新在线节点列表
    } else {
      $q.notify({ message: res.msg || '添加失败', color: 'negative', position: 'top', timeout: 2000 });
    }
  } catch (e) {
    console.error('添加节点失败', e);
    $q.notify({ message: '添加失败', color: 'negative', position: 'top', timeout: 2000 });
  }
};

const toggleLanDiscovery = async (val) => {
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

onMounted(() => {
  document.title = '系统信息';
  fetchSearch();
  queryIpAddr();
  fetchLogs();
  fetchPeers();
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
