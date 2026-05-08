const isMobile = () => {
  return (
    navigator.userAgent.indexOf('Mobile') > -1 ||
    navigator.userAgent.indexOf('Android') > -1 ||
    navigator.userAgent.indexOf('iPhone') > -1
  );
};

// 判断当前是否为Electron环境
const isElectron = () => {
  // 通过检测是否存在Electron特有的process对象和versions属性来判断
  return (
    navigator.userAgent.indexOf('Electron') > -1
  );
};

export { isMobile, isElectron };
