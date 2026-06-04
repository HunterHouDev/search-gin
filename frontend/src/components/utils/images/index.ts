import { useSystemProperty } from '../../../stores/System';

const systemProperty = useSystemProperty();

export const getPng = (Id: string) => {
  if (systemProperty.isElectron) {
    return 'http://localhost:10081/api/png/' + Id;
  }
  return '/api/png/' + Id;
};

export const getJpg = (Id: string) => {
  if (systemProperty.isElectron) {
    return 'http://localhost:10081/api/jpg/' + Id;
  }
  return '/api/jpg/' + Id;
};

export const getFileStream = (id: string) => {
  if (systemProperty.isElectron) {
    return 'http://localhost:10081/api/file/' + id;
  }
  return '/api/file/' + id;
};

export const getTempImage = (id: string) => {
  if (systemProperty.isElectron) {
    return 'http://localhost:10081/api/tempimage/' + id;
  }
  return '/api/tempimage/' + id;
};

export const getActressImage = (actressUrl: string) => {
  if (systemProperty.isElectron) {
    return 'http://localhost:10081/api/actressImgae/' + actressUrl;
  }
  return '/api/actressImgae/' + actressUrl;
};

export const getVideoSrt = (path: string) => {
  return '/api/GetFileByPathUseEncode/' + encodeURI(path);
};

export const GetFileByPathUseEncode = (path: string) => {
  if (systemProperty.isElectron) {
    return 'http://localhost:10081/api/GetFileByPathUseEncode/' + encodeURI(path);
  }
  return '/api/GetFileByPathUseEncode/' + encodeURI(path);
};
