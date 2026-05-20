import { useSystemProperty } from '../../../stores/System';
import { computed } from 'vue';

const systemProperty = useSystemProperty();
const settingInfo = computed(() => {
  return systemProperty.SettingInfo;
});

let ImageHost = '';
let StreamHost = '';

export const getPng = (Id: string) => {
  const url = '/api/png/' + Id;
  if (ImageHost.length > 0) {
    return ImageHost + url;
  }
  if (settingInfo.value.ImageHost) {
    if (systemProperty.isElectron) {
      ImageHost = 'http://localhost:10081';
      return ImageHost + url;
    }
    if (settingInfo.value.ImageHost.indexOf(':') == 0) {
      ImageHost =
        window.location.protocol +
        '//' +
        window.location.hostname +
        settingInfo.value.ImageHost;
      return ImageHost + url;
    }
    ImageHost = settingInfo.value.ImageHost;
    return ImageHost + url;
  }
  return url;
};
export const getJpg = (Id: string) => {
  const url = '/api/jpg/' + Id;
  if (ImageHost.length > 0) {
    return ImageHost + url;
  }
  if (settingInfo.value.ImageHost) {
    if (systemProperty.isElectron) {
      ImageHost = 'http://localhost:10081';
      return ImageHost + url;
    }
    if (settingInfo.value.ImageHost.indexOf(':') == 0) {
      ImageHost =
        window.location.protocol +
        '//' +
        window.location.hostname +
        settingInfo.value.ImageHost;
      return ImageHost + url;
    }
    ImageHost = settingInfo.value.ImageHost;
    return ImageHost + url;
  }
  return url;
  // return settingInfo.value.ImageHost + '/api/jpg/' + Id;
};

export const getFileStream = (id: string) => {
  const url = '/api/file/' + id;
  
  // Electron 环境直接使用 localhost:10081
  if (systemProperty.isElectron) {
    return 'http://localhost:10081' + url;
  }
  
  // Web 环境优先使用相对路径（走当前端口，由 Quasar 代理转发）
  if (!settingInfo.value.StreamHost) {
    return url;
  }
  
  // 自定义 StreamHost 配置
  if (settingInfo.value.StreamHost.indexOf(':') == 0) {
    return window.location.protocol + '//' + window.location.hostname + settingInfo.value.StreamHost + url;
  }
  return settingInfo.value.StreamHost + url;
};

export const getTempImage = (id: string) => {
  // return settingInfo.value.StreamHost + '/api/tempimage/' + id;
  const url = '/api/tempimage/' + id;
  if (StreamHost.length > 0) {
    return StreamHost + url;
  }
  if (settingInfo.value.StreamHost) {
    if (systemProperty.isElectron) {
      ImageHost = 'http://localhost:10081';
      return ImageHost + url;
    }
    if (settingInfo.value.StreamHost.indexOf(':') == 0) {
      StreamHost =
        window.location.protocol +
        '//' +
        window.location.hostname +
        settingInfo.value.StreamHost;
      return StreamHost + url;
    }
    StreamHost = settingInfo.value.StreamHost  || '';
    return StreamHost + url;
  }
  return url;
};

export const getActressImage = (actressUrl: string) => {
  // return settingInfo.value.ImageHost + '/api/actressImgae/' + actressUrl;
  const url = '/api/actressImgae/' + actressUrl;
  if (ImageHost.length > 0) {
    return ImageHost + url;
  }
  if (settingInfo.value.ImageHost) {
    if (systemProperty.isElectron) {
      ImageHost = 'http://localhost:10081';
      return ImageHost + url;
    }
    if (settingInfo.value.ImageHost.indexOf(':') == 0) {
      ImageHost =
        window.location.protocol +
        '//' +
        window.location.hostname +
        settingInfo.value.ImageHost;
      return ImageHost + url;
    }
    ImageHost = settingInfo.value.ImageHost;
    return ImageHost + url;
  }
  return url;
};

export const getVideoSrt = (path: string) => {
  const url = '/api/GetFileByPathUseEncode/' + encodeURI(path);
  return url;
};

export const GetFileByPathUseEncode = (path: string) => {
  const url = '/api/GetFileByPathUseEncode/' + encodeURI(path);
  if (StreamHost.length > 0) {
    return StreamHost + url;
  }
  if (settingInfo.value.StreamHost) {
    if (systemProperty.isElectron) {
      ImageHost = 'http://localhost:10081';
      return ImageHost + url;
    }
    if (settingInfo.value.StreamHost.indexOf(':') == 0) {
      StreamHost =
        window.location.protocol +
        '//' +
        window.location.hostname +
        settingInfo.value.StreamHost;
      return StreamHost + url;
    }
    StreamHost = settingInfo.value.ImageHost || '';
    return StreamHost + url;
  }
  return url;
};
