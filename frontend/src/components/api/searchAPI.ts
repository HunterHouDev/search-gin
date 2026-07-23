import { commonAxios } from '../../boot/axios';
import { RouteParamValue } from 'vue-router';
import type { AxiosRequestConfig } from 'axios';

export const SearchAPI = async (params: object, signal?: AbortSignal) => {
  const config: AxiosRequestConfig = {};
  if (signal) config.signal = signal;
  const { data } = await commonAxios().post('/api/movieList', params, config);
  return data;
};

// 文件归属所需字段：Id 用于索引更新，Path 用于源节点直接操作磁盘，Host(NodeHost) 用于判定本机/远程转发
type FileItemLike = { Id: string; Path?: string; NodeHost?: string };

// 调用方可传：完整 item 对象（含 NodeHost 触发远程转发）/ 普通对象 / 字符串 id（按本机处理）
type OpItem = FileItemLike | Record<string, unknown> | string;

// 构造统一文件操作请求体 { id, path, host, ...extra }
const opBody = (data: OpItem, extra: Record<string, unknown> = {}) => {
  const item = (typeof data === 'string' ? { Id: data } : (data as FileItemLike));
  return {
    id: item.Id,
    path: item.Path || '',
    host: item.NodeHost || '',
    ...extra,
  };
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

export const FindFileInfo = async (data: OpItem) => {
  const res = await commonAxios().post(`/api/info`, opBody(data));
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

export const OpenFileFolder = async (data: OpItem) => {
  const res = await commonAxios().post(`/api/openFolder`, opBody(data));
  return res && res.data;
};

export const DeleteFile = async (data: OpItem) => {
  const res = await commonAxios().post(`/api/delete`, opBody(data));
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

export const TansferFileVcode = async (data: OpItem, vcode: string) => {
  const res = await commonAxios().post(`/api/tranferToMp4`, opBody(data, { xcode: vcode }));
  return res && res.data;
};

export const CutFile = async (data: OpItem, start: string, end: string) => {
  const res = await commonAxios().post(`/api/cutMovie`, opBody(data, { start, end }));
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

export const AddTag = async (data: OpItem, title: string) => {
  const res = await commonAxios().post(`/api/addFileTag`, opBody(data, { tag: title }));
  return res && res.data;
};

export const CloseTag = async (data: OpItem, title: string) => {
  const res = await commonAxios().post(`/api/clearFileTag`, opBody(data, { tag: title }));
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
  data: OpItem,
  type: string,
  start: string,
  downFlag: boolean
) => {
  const res = await commonAxios().post(
    `/api/cutImage`,
    opBody(data, { typeImage: type, start, downFlag: String(downFlag) })
  );
  return res && res.data;
};
