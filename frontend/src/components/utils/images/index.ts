// 保留函数：getTempImage / getAuthorImage / getVideoSrt / GetFileByPathUseEncode
// getPng / getJpg / getFileStream 已废弃，改为从 movie 对象直接读取
// streamUrl / pngUrl / jpgUrl 字段

let _fileBaseUrl: string | null = null;

export const setFileBaseUrl = (url: string) => {
  _fileBaseUrl = url;
};

const getFileBaseUrl = (): string => {
  return _fileBaseUrl || window.location.origin;
};

export const getTempImage = (id: string) => {
  return `${getFileBaseUrl()}/api/stream/tempimage/` + id;
};

export const getAuthorImage = (actressUrl: string) => {
  return '/api/authorImage/' + actressUrl;
};

export const getVideoSrt = (path: string) => {
  return '/api/stream/GetFileByPathUseEncode/' + encodeURI(path);
};

export const GetFileByPathUseEncode = (path: string) => {
  return `${getFileBaseUrl()}/api/stream/GetFileByPathUseEncode/` + encodeURI(path);
};
