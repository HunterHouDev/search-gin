
export const getAuthorImage = (name: string) => {
  return '/api/authorImage/' + name;
};

/**
 * 字幕流地址。
 *
 * 必须用 encodeURIComponent（不是 encodeURI）：文件名里的 `#` / `?` / `&` / `+`
 * 在 encodeURI 下不会被转义，`#` 之后会被浏览器当成片段直接截掉，
 * 表现为字幕（乃至整个地址）404。服务端按路径语义解码，能还原这些字符。
 */
export const getVideoSrt = (path: string) => {
  return '/api/stream/GetFileByPathUseEncode/' + encodeURIComponent(path);
};
