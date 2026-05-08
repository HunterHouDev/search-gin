import { commonAxios } from '../../boot/axios';
import { RouteParamValue } from 'vue-router';

export const SearchAPI = async (params: object) => {
  const { data } = await commonAxios().post('/api/movieList', params);
  return data;
};

export const RefreshAPI = async (BaseDir: string) => {
  if (BaseDir && BaseDir.length > 0) {
    const params = encodeURI(BaseDir);
    return RefreshTargetAPI(params);
  }
  const res = await commonAxios().get(`/api/refreshIndex`);
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

export const QueryDirImageBase64 = async (data: string, sort: string) => {
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
  const res = await commonAxios().get(`/api/delete/${data}`);
  return res && res.data;
};

export const FilesMerge = async (data: object) => {
  const res = await commonAxios().post('/api/mergeFiles', data);
  return res && res.data;
};

export const SyncFileInfo = async (data: object) => {
  const res = await commonAxios().post('/api/sync', data);
  return res && res.data;
};

export const TransferTasksInfo = async () => {
  const res = await commonAxios().get('/api/transferTasks');
  return res && res.data;
};

export const DelTransferTasksInfo = async (create: string) => {
  const res = await commonAxios().get(`/api/delTransferTasks/${create}`);
  return res && res.data;
};

export const TansferFile = async (data: string) => {
  const res = await commonAxios().get(`/api/tranferToMp4/${data}`);
  return res && res.data;
};

export const MergeSrt = async (data: string) => {
  const res = await commonAxios().get(`/api/mergeSrt/${data}`);
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
  const res = await commonAxios().get(`/api/setMovieType/${data}/${movieType}`);
  return res && res.data;
};

export const DownImageList = async (data: string) => {
  const res = await commonAxios().get(`/api/imageList/${data}`);
  return res && res.data;
};

export const HeartBeatQuery = async () => {
  const res = await commonAxios().get('/api/heartBeat');
  return res && res.data;
};

export const AddTag = async (clickId: string, title: string) => {
  const res = await commonAxios().get(`/api/file/addTag/${clickId}/${title}`);
  return res && res.data;
};

export const CloseTag = async (id: string, title: string) => {
  const res = await commonAxios().get(`/api/file/clearTag/${id}/${title}`);
  return res && res.data;
};

export const FileRename = async (data: unknown) => {
  const res = await commonAxios().post('/api/file/rename', data);
  return res && res.data;
};

export const MoveFile = async (data: unknown) => {
  const res = await commonAxios().post('/api/file/move', data);
  return res && res.data;
}

export const OpenFolerByPath = async (data: unknown) => {
  const res = await commonAxios().post('/api/OpenFolerByPath', data);
  return res && res.data;
};
export const DeleteFolerByPath = async (data: unknown) => {
  const res = await commonAxios().post('/api/DeleteFolerByPath', data);
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
