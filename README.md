# gosrc 文件管理系统

功能:本地磁盘搜索系统

技术架构:
1. golang 后台逻辑
2. gin 前后端交互以及服务启动
3. sqlite 用于搜索一种方式 3W数据以上推荐
4. vue+element 前端开发 （运行使用打包后的文件）
5. ffmpeg 视频处理 转码   ffplay和ffprobe以待后续
6. Election桌面开发 半成品可忽略，好久没开发了
### 食用方式
```
1  代码库 下载viteApp 点击appVue.exe 即可运行
2. 自己DIY 下载代码库 安装GO环境与Node环境
  1、 主代码（GO）即本目录
  2、 前端代码 开发路径： vitehome（V3） vuehome（V2 弃用） ；打包后放到viteApp指定目录即可
  3、 main.go 为go启动文件  go run main.go 即可启动服务 内含go打包命令
    // 1 windows打包  go build -o viteApp/appVite.exe -ldflags  "-H=windowsgui" -tags=prod
    // 2 linux打包  go build -o viteApp/appVite.exe -ldflags  "-H=windowsgui" -tags=prod
  4、  winVite.sh windows 打包脚本 指定前后端打包并移动相关文件到viteApp目录
    参数  1 只打包前端 2 前端加后端一起 3 加election（好久没开发了）
```
