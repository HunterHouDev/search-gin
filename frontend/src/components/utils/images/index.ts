import { useSystemProperty } from '../../../stores/System';

const systemProperty = useSystemProperty();

// 文件/图片流服务端口（与 API 端口分离）
const FILE_PORT = '10082';

export const getPng = (Id: string) => {
  if (systemProperty.isElectron) {
    return `http://localhost:${FILE_PORT}/api/png/` + Id;
  }
  return `/api/png/` + Id;
};

export const getJpg = (Id: string) => {
  if (systemProperty.isElectron) {
    return `http://localhost:${FILE_PORT}/api/jpg/` + Id;
  }
  return `/api/jpg/` + Id;
};

export const getFileStream = (id: string) => {
  if (systemProperty.isElectron) {
    return `http://localhost:${FILE_PORT}/api/file/` + id;
  }
  return `/api/file/` + id;
};

export const getTempImage = (id: string) => {
  if (systemProperty.isElectron) {
    return `http://localhost:${FILE_PORT}/api/tempimage/` + id;
  }
  return `/api/tempimage/` + id;
};

export const getActressImage = (actressUrl: string) => {
  if (systemProperty.isElectron) {
    return `http://localhost:10081/api/actressImgae/` + actressUrl;
  }
  return `/api/actressImgae/` + actressUrl;
};

export const getVideoSrt = (path: string) => {
  return `/api/GetFileByPathUseEncode/` + encodeURI(path);
};

export const GetFileByPathUseEncode = (path: string) => {
  if (systemProperty.isElectron) {
    return `http://localhost:${FILE_PORT}/api/GetFileByPathUseEncode/` + encodeURI(path);
  }
  return `/api/GetFileByPathUseEncode/` + encodeURI(path);
};
