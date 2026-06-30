export class SettingInfo {
  AdminPassword = '';
  Tags: string[] = [];
  ImageTypes: string[] = [];
  DocsTypes: string[] = [];
  VideoTypes: string[] = [];
  DirsLib: string[] = [];
  Dirs: string[] = [];
  Types: string[] = ['mp4', 'jpg'];
  MovieTypes: string[] = ['骑兵', '步兵', '国产', '漫动'];
  TagsLib: string[] = [];
  Pages: string[] = [];

  EnableTimeScan = true;
  CutThenDelete = false;
  SystemPlayer = 'ffplay';
  SystemPlayerVolumn = '30';
  SystemPlayerWidth = '1280';
  HardwareAcceleration = false;
  HardwareAccelMode = '';

  EnableLanDiscovery: boolean | null = null;
  NodeName: string | undefined;
  DiscoveryPeers: string[] = [];

  ControllerHost: string | undefined;
  FileHost: string | undefined;
  BaseUrl: string | undefined;
  ImageUrl: string | undefined;
  Remark: string | undefined;
}
