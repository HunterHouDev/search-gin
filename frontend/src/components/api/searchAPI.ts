import { commonAxios } from '../../boot/axios';
import { RouteParamValue } from 'vue-router';
import type { AxiosRequestConfig } from 'axios';

export const SearchAPI = async (params: object, signal?: AbortSignal) => {
  const config: AxiosRequestConfig = {};
  if (signal) config.signal = signal;
  const { data } = await commonAxios().post('/api/movieList', params, config);
  return data;
};

export const RefreshAPI = async (BaseDir: string) => {
  if (BaseDir && BaseDir.length > 0) {
    const params = encodeURI(BaseDir);
    return RefreshTargetAPI(params);
  }
  const res = await commonAxios().get('/api/refreshIndex');
  return res && res.data;
};

export const RefreshTargetAPI = async (params: string) => {
  const res = await commonAxios().get(`/api/refreshTargetIndex/${params}`);
  return res && res.data;
};

export const FindFileInfo = async (data: string | RouteParamValue[]) => {
  const res = await commonAxios().get(`/api/info/${data}`);
  return res && res.data;
};

export const QueryDirImages = async (data: string, sort: string) => {
  const res = await commonAxios().get(`/api/dir/${data}/${sort}`);
  return res;
};

export const DeleteFileByPathUseEncode = async (path: string) => {
  const res = await commonAxios().get(
    `/api/DeleteFileByPathUseEncode/${encodeURI(path)}`
  );
  return res;
};

export const PlayMovie = async (data: string) => {
  const res = await commonAxios().get(`/api/play/${data}`);
  return res && res.data;
};

export const OpenFileFolder = async (data: string) => {
  const res = await commonAxios().get(`/api/openFolder/${data}`);
  return res && res.data;
};

export const DeleteFile = async (data: string) => {
  const res = await commonAxios().delete(`/api/delete/${data}`);
  return res && res.data;
};

export const FilesMerge = async (data: object) => {
  const res = await commonAxios().post('/api/mergeFiles', data);
  return res && res.data;
};

export const TransferTasksInfo = async () => {
  const res = await commonAxios().get('/api/transferTasks');
  return res && res.data;
};

export const DelTransferTasksInfo = async (taskID: string) => {
  const res = await commonAxios().get(`/api/delTransferTasks/${taskID}`);
  return res && res.data;
};

export const ClearCompletedTasks = async () => {
  const res = await commonAxios().post('/api/clearCompletedTasks');
  return res && res.data;
};

export const ClearFailedTasks = async () => {
  const res = await commonAxios().post('/api/clearFailedTasks');
  return res && res.data;
};

export const ClearAllTasks = async () => {
  const res = await commonAxios().post('/api/clearAllTasks');
  return res && res.data;
};

export const TansferFileVcode = async (data: string, vcode: string) => {
  const res = await commonAxios().get(`/api/tranferToMp4/${data}/${vcode}`);
  return res && res.data;
};

export const CutFile = async (id: string, start: string, end: string) => {
  const res = await commonAxios().get(`/api/cutMovie/${id}/${start}/${end}`);
  return res && res.data;
};

export const ResetMovieType = async (data: string, movieType: string) => {
  const res = await commonAxios().post(`/api/setMovieType/${data}/${movieType}`);
  return res && res.data;
};

export const HeartBeatQuery = async () => {
  const res = await commonAxios().get('/api/heartBeat');
  return res && res.data;
};

export const IndexHealthQuery = async () => {
  const res = await commonAxios().get('/api/indexHealth');
  return res && res.data;
};

export const AddTag = async (clickId: string, title: string) => {
  const res = await commonAxios().post(`/api/addFileTag/${clickId}/${title}`);
  return res && res.data;
};

export const CloseTag = async (id: string, title: string) => {
  const res = await commonAxios().post(`/api/clearFileTag/${id}/${title}`);
  return res && res.data;
};

export const FileRename = async (data: unknown) => {
  const res = await commonAxios().post('/api/renameFile', data);
  return res && res.data;
};

export const MoveFile = async (data: unknown) => {
  const res = await commonAxios().post('/api/moveFile', data);
  return res && res.data;
}

export const OpenFolderByPath = async (data: unknown) => {
  const res = await commonAxios().post('/api/OpenFolderByPath', data);
  return res && res.data;
};
export const DeleteFolderByPath = async (data: unknown) => {
  const res = await commonAxios().post('/api/DeleteFolderByPath', data);
  return res && res.data;
};

// 查询单任务日志
export const GetTaskLogAPI = async (taskID: string) => {
  const res = await commonAxios().get(`/api/taskLog/${taskID}`);
  return res && res.data;
};

export const CutImage = async (
  id: string,
  type: string,
  start: string,
  downFlag: boolean
) => {
  const res = await commonAxios().get(
    `/api/cutImage/${id}/${type}/${downFlag}/${start}`
  );
  return res && res.data;
};
