// 单端口服务：图片/文件流与 API 共用同一端口
// 默认使用当前页面 origin，可通过 setFileBaseUrl 覆盖（从 ControllerHost 配置读取）

let _fileBaseUrl: string | null = null;

export const setFileBaseUrl = (url: string) => {
  _fileBaseUrl = url;
};

const getFileBaseUrl = (): string => {
  return _fileBaseUrl || window.location.origin;
};

export const getPng = (Id: string) => {
  return `${getFileBaseUrl()}/api/stream/png/` + Id;
};

export const getJpg = (Id: string) => {
  return `${getFileBaseUrl()}/api/stream/jpg/` + Id;
};

export const getFileStream = (id: string) => {
  return `${getFileBaseUrl()}/api/stream/file/` + id;
};

export const getTempImage = (id: string) => {
  return `${getFileBaseUrl()}/api/stream/tempimage/` + id;
};

export const getActressImage = (actressUrl: string) => {
  // actressImgae 是 API 路由，在 10081 上
  return '/api/actressImgae/' + actressUrl;
};

export const getVideoSrt = (path: string) => {
  return '/api/stream/GetFileByPathUseEncode/' + encodeURI(path);
};

export const GetFileByPathUseEncode = (path: string) => {
  return `${getFileBaseUrl()}/api/stream/GetFileByPathUseEncode/` + encodeURI(path);
};
