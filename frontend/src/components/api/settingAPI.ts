import { commonAxios } from '../../boot/axios';
import { SettingInfo } from '../model/Setting';

export const GeMemeryLog = async () => {
  const res = await commonAxios().get('/api/logMemory');
  return res;
};

export const GetLocalLog = async () => {
  const res = await commonAxios().get('/api/localLog');
  return res;
};

export const ClearMemoryLog = async () => {
  const res = await commonAxios().post('/api/clearMemoryLog');
  return res && res.data;
};

export const ClearLocalLog = async () => {
  const res = await commonAxios().post('/api/clearLocalLog');
  return res && res.data;
};

export const GetSettingInfo = async () => {
  const res = await commonAxios().get('/api/settingInfo');
  return res;
};

export const PostSettingInfo = async (data: SettingInfo) => {
  const res = await commonAxios().post('/api/setting', data);
  return res && res.data;
};

export const GetIpAddr = async () => {
  const res = await commonAxios().get('/api/GetIpAddr');
  return res && res.data;
};

export const GetLanPeers = async () => {
  const res = await commonAxios().get('/api/lanPeers');
  return res && res.data;
};

export const GetLanPeersWithStats = async () => {
  const res = await commonAxios().get('/api/lanPeersWithStats');
  return res && res.data;
};

export const AddLanPeer = async (ip: string, port: string, filePort: string) => {
  const res = await commonAxios().post('/api/addPeer', { ip, port, filePort });
  return res && res.data;
};

export const PingHost = async (ip: string) => {
  const res = await commonAxios().get('/api/pingHost', { params: { ip } });
  return res && res.data;
};

export const RemoveLanPeer = async (id: string) => {
  const res = await commonAxios().post('/api/removePeer', { id });
  return res && res.data;
};

export const TogglePeer = async (id: string, disabled: boolean) => {
  const res = await commonAxios().post('/api/togglePeer', { id, disabled });
  return res && res.data;
};

export const DiscoverLanPeers = async (subnet: string) => {
  const res = await commonAxios().post('/api/discoverPeers', { subnet });
  return res && res.data;
};

export const CleanLanPeers = async () => {
  const res = await commonAxios().post('/api/cleanLanPeers');
  return res && res.data;
};

export const GetShutDown = async () => {
  const res = await commonAxios().get('/api/shutDown');
  return res as unknown;
};

export const AppShutDown = async () => {
  const res = await commonAxios().get('/api/close');
  return res && res.data;
};

export const AppRestart = async () => {
  const res = await commonAxios().get('/api/restart');
  return res && res.data;
};

export const GetServerPort = async () => {
  const res = await commonAxios().get('/api/serverPort');
  return res && res.data;
};

// 普通用户管理
export const GetUsers = async () => {
  const res = await commonAxios().get('/api/users');
  return res && res.data;
};

export const AddUser = async (username: string, password: string, expireDate = '') => {
  const res = await commonAxios().post('/api/user/add', {
    username,
    password,
    expireDate
  });
  return res && res.data;
};

export const DeleteUser = async (username: string) => {
  const res = await commonAxios().post('/api/user/delete', {
    username
  });
  return res && res.data;
};

// ── 权限管理 ──────────────────────────────────────────────────────

export const GetAllPermissions = async () => {
  const res = await commonAxios().get('/api/permissions');
  return res && res.data;
};

export const GetUserPermissions = async (username: string) => {
  const res = await commonAxios().get(`/api/user/${username}/permissions`);
  return res && res.data;
};

export const UpdateUserPermissions = async (username: string, permissions: string[]) => {
  const res = await commonAxios().post('/api/user/permissions', {
    username,
    permissions
  });
  return res && res.data;
};

// ── 角色管理 ──────────────────────────────────────────────────────

export const GetRoles = async () => {
  const res = await commonAxios().get('/api/roles');
  return res && res.data;
};

export const CreateRole = async (name: string, label: string, permissions: string[]) => {
  const res = await commonAxios().post('/api/roles', { name, label, permissions });
  return res && res.data;
};

export const UpdateRole = async (name: string, label: string, permissions: string[]) => {
  const res = await commonAxios().post(`/api/roles/${name}`, { name, label, permissions });
  return res && res.data;
};

export const DeleteRole = async (name: string) => {
  const res = await commonAxios().delete(`/api/roles/${name}`);
  return res && res.data;
};

export const UpdateUserRole = async (username: string, role: string) => {
  const res = await commonAxios().post('/api/user/role', { username, role });
  return res && res.data;
};
