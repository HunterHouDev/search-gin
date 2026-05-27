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

export const GetShutDown = async () => {
  const res = await commonAxios().get('/api/shutDown');
  return res as unknown;
};

export const AppShutDown = async () => {
  const res = await commonAxios().get('/api/close');
  return res && res.data;
};

// 用户管理API
export const GetUsers = async () => {
  const res = await commonAxios().get('/api/users');
  return res && res.data;
};

export const AddUser = async (username: string, password: string, role: string = 'user', expireDate: string = '') => {
  const res = await commonAxios().post('/api/user/add', {
    username,
    password,
    role,
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

export const ChangePassword = async (username: string, oldPassword: string, newPassword: string) => {
  const res = await commonAxios().post('/api/user/changePassword', {
    username,
    oldPassword,
    newPassword
  });
  return res && res.data;
};
