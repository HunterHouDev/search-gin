# gosrc 文件管理系统

功能:本地磁盘搜索系统

技术架构:
1. golang 后台逻辑
2. gin 前后端交互以及服务启动
3. vue前端开发 （运行使用打包后的文件）
4. ffmpeg 视频处理 转码   ffplay

### 使用方式
```
  根目录 
1 web系统：执行打包脚本 sh buildQuasar.sh 2， 生成qapp文件夹（可移动）点击exe启动WEB服务，端口10081可访问系统
2 桌面系统：执行打包脚本 sh buildQuasar.sh 4， 生成PC打包目录electron_quasar\dist\electron\Packaged\文件搜索系统-win32-x64 点击【文件搜索系统.exe】启动桌面软件
```
