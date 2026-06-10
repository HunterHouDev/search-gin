// 文件/图片流服务端口（与 API 端口 10081 分离）
const FILE_PORT = '10082';

// 动态获取文件流基础 URL：使用当前页面的 hostname，替换端口为 10082
// 这样无论是 localhost、127.0.0.1、局域网 IP 还是域名都能正确访问
const getFileBaseUrl = (): string => {
  return `${window.location.protocol}//${window.location.hostname}:${FILE_PORT}`;
};

export const getPng = (Id: string) => {
  return `${getFileBaseUrl()}/api/png/` + Id;
};

export const getJpg = (Id: string) => {
  return `${getFileBaseUrl()}/api/jpg/` + Id;
};

export const getFileStream = (id: string) => {
  return `${getFileBaseUrl()}/api/file/` + id;
};

export const getTempImage = (id: string) => {
  return `${getFileBaseUrl()}/api/tempimage/` + id;
};

export const getActressImage = (actressUrl: string) => {
  // actressImgae 是 API 路由，在 10081 上
  return `/api/actressImgae/` + actressUrl;
};

export const getVideoSrt = (path: string) => {
  return `/api/GetFileByPathUseEncode/` + encodeURI(path);
};

export const GetFileByPathUseEncode = (path: string) => {
  return `${getFileBaseUrl()}/api/GetFileByPathUseEncode/` + encodeURI(path);
};
