
export const getAuthorImage = (name: string) => {
  return '/api/authorImage/' + name;
};

export const getVideoSrt = (path: string) => {
  return '/api/stream/GetFileByPathUseEncode/' + encodeURI(path);
};
