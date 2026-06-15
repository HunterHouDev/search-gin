
let _fileBaseUrl: string | null = null;

export const setFileBaseUrl = (url: string) => {
  _fileBaseUrl = url;
};

const getFileBaseUrl = (): string => {
  return _fileBaseUrl || window.location.origin;
};

export const getAuthorImage = (authorUrl: string) => {
  return '/api/authorImage/' + authorUrl;
};

export const getVideoSrt = (path: string) => {
  return '/api/stream/GetFileByPathUseEncode/' + encodeURI(path);
};

export const GetFileByPathUseEncode = (path: string) => {
  return `${getFileBaseUrl()}/api/stream/GetFileByPathUseEncode/` + encodeURI(path);
};
