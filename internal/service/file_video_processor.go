package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
	"time"
)

// encoder 视频转码/处理服务
type videoEncoder struct{}

// TransferFormatter 视频转码格式化
func (e *videoEncoder) TransferFormatter(task model.TransferTaskModel) utils.Result {
	switch task.VCode {
	case "h264":
		return e.transferWithEncoder(task, e.getH264Encoder(), "23")
	case "h265":
		return e.transferWithEncoder(task, e.getH265Encoder(), "28")
	default:
		return e.transferFormatWithCopy(task)
	}
}

// transferWithEncoder 通用转码函数（合并原来的 TransferFormatter264/265）
func (e *videoEncoder) transferWithEncoder(task model.TransferTaskModel, encoder, crf string) utils.Result {
	from := task.Path
	suffix := utils.GetSuffix(task.Path)

	if suffix == task.To {
		if suffix == "mp4" {
			task.To = "mov"
		} else {
			task.To = "mp4"
		}
	}

	dest := strings.ReplaceAll(task.Path, "."+suffix, "."+task.To)
	decodeParams := e.getHwDecodeParams()
	qualityParam := e.getHwQualityParam()

	args := make([]string, 0, 10)
	if decodeParams != "" {
		args = append(args, strings.Fields(decodeParams)...)
	}
	args = append(args, "-i", from, "-c:v", encoder, qualityParam, crf, dest)

	res := e.ffmpegExec(args, task.CreateTime)

	if res.IsSuccess() {
		e.cleanupSourceIfNeeded(task.Path)
	}

	return res
}

// cleanupSourceIfNeeded 如果配置了转码后删除源文件，则执行删除
func (e *videoEncoder) cleanupSourceIfNeeded(path string) {
	if GetOSSetting().CutThenDelete {
		if err := os.Remove(path); err != nil {
			utils.InfoFormat("删除源文件失败: %s, 错误: %v", path, err)
		}
	}
}

// transferFormatWithCopy 以 copy 方式转码（不重新编码）
func (e *videoEncoder) transferFormatWithCopy(task model.TransferTaskModel) utils.Result {
	from := task.Path
	suffix := utils.GetSuffix(task.Path)

	if suffix == task.To {
		if suffix == "mp4" {
			task.To = "mov"
		} else {
			task.To = "mp4"
		}
	}

	dest := strings.ReplaceAll(task.Path, "."+suffix, "."+task.To)
	args := []string{"-i", from, "-vcodec", "copy", dest}
	res := e.ffmpegExec(args, task.CreateTime)

	if res.IsSuccess() {
		e.cleanupSourceIfNeeded(task.Path)
	}

	return res
}

// MergeFiles 合并文件
func (e *videoEncoder) MergeFiles(task model.TransferTaskModel) utils.Result {
	args := []string{"-f", "concat", "-safe", "0", "-i", task.ConcatFile, "-vcodec", "copy", task.Dest}
	res := e.ffmpegExec(args, task.CreateTime)

	if res.IsSuccess() && task.DeleteSource {
		e.cleanupSourceIfNeeded(task.Path)
	}

	return res
}

// CutFormatter 视频剪辑格式化
func (e *videoEncoder) CutFormatter(task model.TransferTaskModel) utils.Result {
	from := task.Path
	suffix := utils.GetSuffix(task.Path)

	toSuffix := "mkv"
	if suffix == "mkv" {
		toSuffix = "mp4"
	}

	dest := strings.ReplaceAll(task.Path, "."+suffix, "."+toSuffix)
	args := []string{"-i", from, "-ss", task.Start, "-t", task.End, "-c", "copy", dest}
	res := e.ffmpegExec(args, task.CreateTime)

	if res.IsSuccess() && GetOSSetting().CutThenDelete {
		e.cleanupSourceIfNeeded(task.Path)
	}

	return res
}

// CutImage 视频截图
func (e *videoEncoder) CutImage(path string, typeImage string, start string) utils.Result {
	res := utils.NewSuccess()

	isSnapshot := false
	if !strings.EqualFold(typeImage, "Png") && !strings.EqualFold(typeImage, "Jpg") {
		isSnapshot = true
		typeImage = "Jpg"
	}

	dest := strings.TrimSuffix(path, filepath.Ext(path))
	if isSnapshot {
		dest += time.Now().Format("-20060102150405")
	}
	dest += "." + strings.ToLower(typeImage)

	args := []string{"-y", "-ss", start}

	decodeParams := e.getHwDecodeParams()
	if decodeParams != "" {
		args = append(args, strings.Fields(decodeParams)...)
	}

	args = append(args, "-i", path,
		"-f", "image2",
		"-vframes", "1",
		"-an",
		"-vcodec", "mjpeg",
		dest,
	)

	ffmpegPath := e.ffmpegBinPath()
	cmd := exec.Command(ffmpegPath, args...)
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}

	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		utils.InfoFormat("视频截图失败，输出: %v, 错误: %v", string(out), cmdErr)
		res = utils.NewFailByMsg("截图转换失败")

		if utils.ExistsFiles(dest) {
			res.Data = utils.ImageToString(dest)
		}
		return res
	}

	res.Data = utils.ImageToString(dest)
	return res
}

// ffmpegBinPath 获取 ffmpeg 二进制路径
func (e *videoEncoder) ffmpegBinPath() string {
	if WorkDir != "" {
		return filepath.Join(WorkDir, "ffmpeg.exe")
	}
	return "ffmpeg.exe"
}

// ffmpegExec 执行ffmpeg命令
func (e *videoEncoder) ffmpegExec(args []string, thisNow time.Time) utils.Result {
	TransferTaskMutex.Lock()
	task, exists := TransferTask[thisNow]
	if !exists {
		TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务不存在")
	}

	ffmpegPath := e.ffmpegBinPath()

	task.SetStatus("执行中")
	task.CreateTime = time.Now()
	task.Command = ffmpegPath + " " + strings.Join(args, " ")
	TransferTask[thisNow] = task
	TransferTaskMutex.Unlock()

	utils.InfoFormat("执行命令: %v", task.Command)

	cmd := exec.Command(ffmpegPath, args...)
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}

	out, cmdErr := cmd.CombinedOutput()

	TransferTaskMutex.Lock()
	task.SetLog(string(out))
	task.FinishTime = time.Now()

	if cmdErr != nil {
		task.SetStatus("执行失败")
		TransferTask[thisNow] = task
		TransferTaskMutex.Unlock()

		utils.InfoFormat("命令执行失败: %v, 错误: %v, 参数: %v", string(out), cmdErr, args)
		return utils.NewFailByMsg("转换失败")
	}

	task.SetStatus("成功")
	TransferTask[thisNow] = task
	TransferTaskMutex.Unlock()

	return utils.NewSuccessByMsg("转换成功")
}
