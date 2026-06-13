import { commonAxios } from '../../boot/axios';
import { SettingInfo } from '../model/Setting';

export const GeMemeryLog = async () => {
  const res = await commonAxios().get('/api/logMemery');
  return res;
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

export const AddLanPeer = async (ip: string, port: string, filePort: string) => {
  const res = await commonAxios().post('/api/addPeer', { ip, port, filePort });
  return res && res.data;
};

export const PingHost = async (ip: string) => {
  const res = await commonAxios().get('/api/pingHost', { params: { ip } });
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

