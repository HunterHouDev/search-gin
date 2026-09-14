import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ref } from 'vue';
import type { QVueGlobals } from 'quasar';

// useLinkPlayback 通过 searchAPI 提交服务端下载任务，
// 这里只关心提交上去的播放列表文本（多源合并后的结果）
const mockHlsDownloadAPI = vi.fn();
const mockTransferTasksInfo = vi.fn();

vi.mock('src/components/api/searchAPI', () => ({
  HlsDownloadAPI: (...args: unknown[]) => mockHlsDownloadAPI(...args),
  HlsCancelAPI: vi.fn(),
  DelTransferTasksInfo: vi.fn(),
  TransferTasksInfo: (...args: unknown[]) => mockTransferTasksInfo(...args),
}));

interface HlsDownloadPayload {
  playlist: string;
  sourceUrl: string;
  fileName: string;
  dir: string;
}

/** 拼 m3u8 文本 */
function playlist(lines: string[]): string {
  return lines.join('\n') + '\n';
}

const URL_A = 'http://a.test/index.m3u8';
const URL_B = 'http://b.test/index.m3u8';

/** TS 源：2 个分片，总时长 8s */
const HLS_A = playlist([
  '#EXTM3U',
  '#EXT-X-VERSION:3',
  '#EXT-X-TARGETDURATION:4',
  '#EXTINF:4.0,',
  'http://a.test/a1.ts',
  '#EXTINF:4.0,',
  'http://a.test/a2.ts',
  '#EXT-X-ENDLIST',
]);

/** TS 源：1 个分片，时长 6s（用于校验合并后的目标时长取最大值） */
const HLS_B = playlist([
  '#EXTM3U',
  '#EXT-X-VERSION:3',
  '#EXT-X-TARGETDURATION:6',
  '#EXTINF:6.0,',
  'http://b.test/b1.ts',
  '#EXT-X-ENDLIST',
]);

/** fMP4 源：各自带初始化段 */
const HLS_FMP4_A = playlist([
  '#EXTM3U',
  '#EXT-X-VERSION:7',
  '#EXT-X-MAP:URI="http://a.test/initA.mp4"',
  '#EXTINF:4.0,',
  'http://a.test/a1.m4s',
  '#EXT-X-ENDLIST',
]);

const HLS_FMP4_B = playlist([
  '#EXTM3U',
  '#EXT-X-VERSION:7',
  '#EXT-X-MAP:URI="http://b.test/initB.mp4"',
  '#EXTINF:4.0,',
  'http://b.test/b1.m4s',
  '#EXT-X-ENDLIST',
]);

/** AES-128 源：密钥未显式给出 IV，IV 需按 #EXT-X-MEDIA-SEQUENCE + 分片序号推导 */
const HLS_ENCRYPTED = playlist([
  '#EXTM3U',
  '#EXT-X-VERSION:3',
  '#EXT-X-MEDIA-SEQUENCE:10',
  '#EXT-X-KEY:METHOD=AES-128,URI="http://a.test/key.bin"',
  '#EXTINF:4.0,',
  'http://a.test/a1.ts',
  '#EXTINF:4.0,',
  'http://a.test/a2.ts',
  '#EXT-X-ENDLIST',
]);

function stubFetch(map: Record<string, string>) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => ({
      ok: true,
      status: 200,
      text: async () => map[url] ?? '',
    })),
  );
}

/** 动态引入：保证 vi.mock 的工厂在 mock 变量初始化之后再执行 */
async function setup() {
  const { useLinkPlayback } = await import('./useLinkPlayback');
  const $q = { notify: vi.fn() } as unknown as QVueGlobals;
  const api = useLinkPlayback($q, {
    magnetURI: ref(''),
    submitMagnet: () => undefined,
    getVideoEl: () => null,
    getVolume: () => 0.8,
    onPlay: () => undefined,
  });
  // 输入框绑定的是 activeLinkValue，先切到分片 tab 再写入地址
  api.linkTab.value = 'hls';
  return api;
}

function payloadOf(callIndex = 0): HlsDownloadPayload {
  return mockHlsDownloadAPI.mock.calls[callIndex]?.[0] as HlsDownloadPayload;
}

describe('useLinkPlayback 多源分片合并', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTransferTasksInfo.mockResolvedValue({ Data: { tasks: [] } });
    mockHlsDownloadAPI.mockResolvedValue({ Code: 200 });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('一次粘贴多个地址时按顺序合并，下载提交的播放列表带不连续标记', async () => {
    stubFetch({ [URL_A]: HLS_A, [URL_B]: HLS_B });
    const app = await setup();

    app.activeLinkValue.value = `${URL_A}\n${URL_B}`;
    await app.parseHls();

    expect(app.hlsSourceCount.value).toBe(2);
    expect(app.hlsKeptCount.value).toBe(3);
    expect(app.hlsSegmentGroups.value.map((g) => g.segments.length)).toEqual([
      2, 1,
    ]);
    // 解析成功后清空输入框，便于继续粘贴下一个地址
    expect(app.activeLinkValue.value).toBe('');

    await app.downloadHls();

    const payload = payloadOf();
    expect(payload.playlist).toContain('#EXT-X-DISCONTINUITY');
    expect(payload.playlist.indexOf('http://a.test/a2.ts')).toBeLessThan(
      payload.playlist.indexOf('http://b.test/b1.ts'),
    );
    // 目标时长按合并后的最大分片重算（6s > 4s）
    expect(payload.playlist).toContain('#EXT-X-TARGETDURATION:6');
    expect(payload.playlist).not.toContain('#EXT-X-TARGETDURATION:4');
    expect(payload.fileName.endsWith('.ts')).toBe(true);
  });

  it('重复解析同一地址只刷新原源，不会重复追加分片', async () => {
    stubFetch({ [URL_A]: HLS_A, [URL_B]: HLS_B });
    const app = await setup();

    app.activeLinkValue.value = URL_A;
    await app.parseHls();
    expect(app.hlsSourceCount.value).toBe(1);

    app.activeLinkValue.value = URL_A;
    await app.parseHls();

    expect(app.hlsSourceCount.value).toBe(1);
    expect(app.hlsKeptCount.value).toBe(2);
  });

  it('调整源顺序后，下载提交的分片顺序随之改变', async () => {
    stubFetch({ [URL_A]: HLS_A, [URL_B]: HLS_B });
    const app = await setup();

    app.activeLinkValue.value = `${URL_A} ${URL_B}`;
    await app.parseHls();

    const secondSourceId = app.hlsSourceList.value[1].source.id;
    app.moveHlsSource(secondSourceId, -1);

    expect(app.hlsSourceList.value[0].source.url).toBe(URL_B);

    await app.downloadHls();
    const payload = payloadOf();
    expect(payload.playlist.indexOf('http://b.test/b1.ts')).toBeLessThan(
      payload.playlist.indexOf('http://a.test/a1.ts'),
    );
  });

  it('移除某个源时同时移除它名下的分片', async () => {
    stubFetch({ [URL_A]: HLS_A, [URL_B]: HLS_B });
    const app = await setup();

    app.activeLinkValue.value = `${URL_A} ${URL_B}`;
    await app.parseHls();

    app.removeHlsSource(app.hlsSourceList.value[0].source.id);

    expect(app.hlsSourceCount.value).toBe(1);
    expect(app.hlsKeptCount.value).toBe(1);
    expect(app.hlsSegments.value[0].url).toBe('http://b.test/b1.ts');
  });

  it('删掉的分片不会进入合并播放列表，恢复后重新出现', async () => {
    stubFetch({ [URL_A]: HLS_A, [URL_B]: HLS_B });
    const app = await setup();

    app.activeLinkValue.value = `${URL_A} ${URL_B}`;
    await app.parseHls();

    const firstSegment = app.hlsSegments.value[0];
    app.removeHlsSegment(firstSegment.id);
    expect(app.hlsKeptCount.value).toBe(2);

    // 恢复：被删的分片重新回到列表
    app.restoreHlsSegments();
    expect(app.hlsKeptCount.value).toBe(3);

    // 再次删除后提交下载：提交给服务端的播放列表不含被删的分片
    app.removeHlsSegment(firstSegment.id);
    await app.downloadHls();
    expect(payloadOf().playlist).not.toContain('http://a.test/a1.ts');
  });

  it('fMP4 多源：后缀取 mp4，每个源自己的初始化段排在该源分片之前', async () => {
    stubFetch({ [URL_A]: HLS_FMP4_A, [URL_B]: HLS_FMP4_B });
    const app = await setup();

    app.activeLinkValue.value = `${URL_A} ${URL_B}`;
    await app.parseHls();
    await app.downloadHls();

    const payload = payloadOf();
    expect(payload.fileName.endsWith('.mp4')).toBe(true);
    expect(payload.playlist.indexOf('http://a.test/initA.mp4')).toBeLessThan(
      payload.playlist.indexOf('http://a.test/a1.m4s'),
    );
    // 第二个源的初始化段必须排在 DISCONTINUITY 之后、它自己的分片之前
    const discontinuity = payload.playlist.indexOf('#EXT-X-DISCONTINUITY');
    expect(payload.playlist.indexOf('http://b.test/initB.mp4')).toBeGreaterThan(
      discontinuity,
    );
    expect(payload.playlist.indexOf('http://b.test/initB.mp4')).toBeLessThan(
      payload.playlist.indexOf('http://b.test/b1.m4s'),
    );
  });

  it('AES-128 缺省 IV 时按源内序号写出显式 IV（合并后序号已位移）', async () => {
    stubFetch({ [URL_A]: HLS_ENCRYPTED, [URL_B]: HLS_B });
    const app = await setup();

    app.activeLinkValue.value = `${URL_A} ${URL_B}`;
    await app.parseHls();
    await app.downloadHls();

    const text = payloadOf().playlist;
    // #EXT-X-MEDIA-SEQUENCE:10 → 第 1、2 个分片的 IV 依次为 0x0a、0x0b
    expect(text).toContain(
      '#EXT-X-KEY:METHOD=AES-128,URI="http://a.test/key.bin",IV=0x0000000000000000000000000000000a',
    );
    expect(text).toContain(
      '#EXT-X-KEY:METHOD=AES-128,URI="http://a.test/key.bin",IV=0x0000000000000000000000000000000b',
    );
    // 第二个源没有加密，进入该源前要显式清除密钥
    expect(text).toContain('#EXT-X-KEY:METHOD=NONE');
  });
});

describe('useLinkPlayback 分片链接多行输入', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTransferTasksInfo.mockResolvedValue({ Data: { tasks: [] } });
    mockHlsDownloadAPI.mockResolvedValue({ Code: 200 });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  /** 各行的文本（行序号由下标 +1 得到） */
  function rowTexts(rows: { value: string }[]): string[] {
    return rows.map((row) => row.value);
  }

  it('初始只有一行，第一行填入地址后自动生成下一行空行', async () => {
    const app = await setup();
    expect(rowTexts(app.hlsUrlRows.value)).toEqual(['']);

    app.updateHlsUrlRow(app.hlsUrlRows.value[0].id, URL_A);

    expect(rowTexts(app.hlsUrlRows.value)).toEqual([URL_A, '']);
    // 行文本即解析依据（排序号按行序展示）
    expect(app.activeLinkValue.value).toBe(URL_A);
    expect(app.canSubmitLink.value).toBe(true);
  });

  it('一行里粘贴多个地址（空格 / 换行分隔）自动拆成多行，顺序保持不变', async () => {
    const app = await setup();
    const ids = [app.hlsUrlRows.value[0].id];

    app.updateHlsUrlRow(ids[0], `${URL_A} ${URL_B}\nhttp://c.test/index.m3u8`);

    expect(rowTexts(app.hlsUrlRows.value)).toEqual([
      URL_A,
      URL_B,
      'http://c.test/index.m3u8',
      '',
    ]);
    // 多行文本按行序拼成解析用文本，同时保持原行的稳定 id（光标不丢）
    expect(app.activeLinkValue.value.split('\n')).toEqual([
      URL_A,
      URL_B,
      'http://c.test/index.m3u8',
    ]);
    expect(app.hlsUrlRows.value[0].id).toBe(ids[0]);
  });

  it('清空某一行后该行消失（序号不留空档），末尾始终保留空行', async () => {
    const app = await setup();
    const firstRowId = app.hlsUrlRows.value[0].id;
    app.updateHlsUrlRow(firstRowId, URL_A);
    const secondRowId = app.hlsUrlRows.value[1].id;
    app.updateHlsUrlRow(secondRowId, URL_B);
    expect(rowTexts(app.hlsUrlRows.value)).toEqual([URL_A, URL_B, '']);

    // 清空第 1 行：该行保留（正在输入的那一行不重建，避免焦点丢失），多余空行被丢弃
    app.updateHlsUrlRow(firstRowId, '');
    expect(rowTexts(app.hlsUrlRows.value)).toEqual(['', URL_B, '']);

    // 再清空剩下的内容行，回到单行空输入
    app.updateHlsUrlRow(secondRowId, '');
    expect(rowTexts(app.hlsUrlRows.value)).toEqual(['']);
  });

  it('解析成功后输入行被重置为单行空输入', async () => {
    stubFetch({ [URL_A]: HLS_A, [URL_B]: HLS_B });
    const app = await setup();

    app.updateHlsUrlRow(app.hlsUrlRows.value[0].id, URL_A);
    app.updateHlsUrlRow(app.hlsUrlRows.value[1].id, URL_B);
    await app.parseHls();

    expect(app.hlsSourceCount.value).toBe(2);
    expect(rowTexts(app.hlsUrlRows.value)).toEqual(['']);
    expect(app.canSubmitLink.value).toBe(false);
  });

  it('清空按钮把输入行恢复成单行', async () => {
    const app = await setup();
    app.updateHlsUrlRow(app.hlsUrlRows.value[0].id, URL_A);
    app.updateHlsUrlRow(app.hlsUrlRows.value[1].id, URL_B);

    app.clearHlsUrlRows();

    expect(rowTexts(app.hlsUrlRows.value)).toEqual(['']);
    expect(app.activeLinkValue.value).toBe('');
  });
});

describe('useLinkPlayback 提交下载后重置本地状态', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTransferTasksInfo.mockResolvedValue({ Data: { tasks: [] } });
    mockHlsDownloadAPI.mockResolvedValue({ Code: 200 });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  /** 服务端任务列表里的一条分片下载任务 */
  function serverTask(overrides: Partial<Record<string, unknown>> = {}) {
    return {
      ID: 'task-1',
      Type: '分片下载',
      Name: 'index.ts',
      Path: 'D:/media/index.ts',
      URL: URL_A,
      Segments: 0,
      TotalSegments: 3,
      Progress: 0,
      Size: 0,
      Duration: '00:16',
      Status: '执行中',
      Log: '',
      CreateTime: '2026-01-01T00:00:00Z',
      ...overrides,
    };
  }

  it('提交成功后清空分片列表与输入行，下载列表保留', async () => {
    stubFetch({ [URL_A]: HLS_A, [URL_B]: HLS_B });
    const app = await setup();

    app.activeLinkValue.value = `${URL_A}\n${URL_B}`;
    await app.parseHls();
    expect(app.hlsKeptCount.value).toBe(3);

    // 解析后用户又填了一行待处理的地址
    app.updateHlsUrlRow(app.hlsUrlRows.value[0].id, URL_A);
    expect(app.hlsUrlRows.value.map((row) => row.value)).toEqual([URL_A, '']);

    mockTransferTasksInfo.mockResolvedValue({
      Data: { tasks: [serverTask()] },
    });
    await app.downloadHls();

    expect(app.hlsParsed.value).toBe(false);
    expect(app.hlsSourceCount.value).toBe(0);
    expect(app.hlsKeptCount.value).toBe(0);
    expect(app.hlsUrlRows.value.map((row) => row.value)).toEqual(['']);
    // 下载列表是服务端任务的镜像，与本地解析结果无关，必须保留
    expect(app.hlsDownloadList.value).toHaveLength(1);
    expect(app.hlsDownloadList.value[0].id).toBe('task-1');

    app.cleanup();
  });

  it('提交失败时保留分片列表，便于直接重试', async () => {
    stubFetch({ [URL_A]: HLS_A });
    mockHlsDownloadAPI.mockResolvedValue({ Code: 400, Message: '创建失败' });
    const app = await setup();

    app.activeLinkValue.value = URL_A;
    await app.parseHls();
    await app.downloadHls();

    expect(app.hlsKeptCount.value).toBe(2);
    expect(app.hlsSourceCount.value).toBe(1);

    app.cleanup();
  });
});
