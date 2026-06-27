package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
	"time"
)

// TransferFormatter 视频转码格式化
func TransferFormatter(task model.TransferTaskModel) utils.Result {
	switch task.VCode {
	case "h264":
		return transferWithEncoder(task, getH264Encoder(), "23")
	case "h265":
		return transferWithEncoder(task, getH265Encoder(), "28")
	default:
		return transferFormatWithCopy(task)
	}
}

// transferWithEncoder 通用转码函数（合并原来的 TransferFormatter264/265）
func transferWithEncoder(task model.TransferTaskModel, encoder, crf string) utils.Result {
	from := task.Path
	suffix := utils.GetSuffix(task.Path)

	if suffix == task.To {
		if suffix == "mp4" {
			task.To = "mov"
		} else {
			task.To = "mp4"
		}
	}

	dest := replaceSuffix(task.Path, suffix, task.To)
	decodeParams := getHwDecodeParams()
	qualityParam := getHwQualityParam()

	args := make([]string, 0, 10)
	if decodeParams != "" {
		args = append(args, strings.Fields(decodeParams)...)
	}
	args = append(args, "-i", from, "-c:v", encoder, qualityParam, crf, dest)

	res := ffmpegExec(args, task.CreateTime)

	if res.IsSuccess() {
		cleanupSourceIfNeeded(task.Path)
	}

	return res
}

func replaceSuffix(path, oldSuffix, newSuffix string) string {
	// 只替换文件名的最后一个后缀（扩展名），不替换路径中间的重名字符串
	if extLen := len(oldSuffix); extLen > 0 && len(path) > extLen && path[len(path)-extLen-1] == '.' {
		return path[:len(path)-extLen-1] + "." + newSuffix
	}
	return path + "." + newSuffix
}
func cleanupSourceIfNeeded(path string) {
	if GetOSSetting().CutThenDelete {
		if err := os.Remove(path); err != nil {
			utils.InfoFormat("删除源文件失败: %s, 错误: %v", path, err)
		}
	}
}

// transferFormatWithCopy 以 copy 方式转码（不重新编码）
func transferFormatWithCopy(task model.TransferTaskModel) utils.Result {
	from := task.Path
	suffix := utils.GetSuffix(task.Path)

	if suffix == task.To {
		if suffix == "mp4" {
			task.To = "mov"
		} else {
			task.To = "mp4"
		}
	}

	dest := replaceSuffix(task.Path, suffix, task.To)
	args := []string{"-i", from, "-c", "copy", dest}
	res := ffmpegExec(args, task.CreateTime)

	if res.IsSuccess() {
		cleanupSourceIfNeeded(task.Path)
	}

	return res
}

// MergeFiles 合并文件
func MergeFiles(task model.TransferTaskModel) utils.Result {
	args := []string{"-f", "concat", "-safe", "0", "-i", task.ConcatFile, "-c", "copy", task.Dest}
	res := ffmpegExec(args, task.CreateTime)

	// 清理临时合并列表文件
	if task.ConcatFile != "" && utils.ExistsFiles(task.ConcatFile) {
		if err := os.Remove(task.ConcatFile); err != nil {
			utils.InfoFormat("删除合并临时文件失败: %s, 错误: %v", task.ConcatFile, err)
		}
	}

	if res.IsSuccess() && task.DeleteSource {
		cleanupSourceIfNeeded(task.Path)
	}

	return res
}

// CutFormatter 视频剪辑格式化
func CutFormatter(task model.TransferTaskModel) utils.Result {
	from := task.Path
	suffix := utils.GetSuffix(task.Path)

	toSuffix := task.To
	if toSuffix == "" {
		// 兜底：未指定 To 时按原逻辑
		toSuffix = "mkv"
		if suffix == "mkv" {
			toSuffix = "mp4"
		}
	}

	dest := replaceSuffix(task.Path, suffix, toSuffix)
	args := []string{"-i", from, "-ss", task.Start, "-to", task.End, "-c", "copy", dest}
	res := ffmpegExec(args, task.CreateTime)

	if res.IsSuccess() && GetOSSetting().CutThenDelete {
		cleanupSourceIfNeeded(task.Path)
	}

	return res
}

// CutImage 视频截图
func CutImage(path string, typeImage string, start string) utils.Result {
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

	decodeParams := getHwDecodeParams()
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

	ffmpegPath := ffmpegBinPath()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}

	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		utils.InfoFormat("视频截图失败，输出: %v, 错误: %v", string(out), cmdErr)
		return utils.NewFailByMsg("截图转换失败")
	}

	res.Data = utils.ImageToString(dest)
	return res
}

// ffmpegBinPath 获取 ffmpeg 二进制路径
func ffmpegBinPath() string {
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	if GetWorkDir() != "" {
		return filepath.Join(GetWorkDir(), name)
	}
	return name
}

// updateTaskStatus 集中管理任务状态变更
func updateTaskStatus(key time.Time, status, log string) {
	TransferTaskMutex.Lock()
	defer TransferTaskMutex.Unlock()
	t, ok := TransferTask[key]
	if !ok {
		return
	}
	t.Status = status
	t.FinishTime = time.Now()
	if log != "" {
		t.Log = log
	}
	TransferTask[key] = t
	wakeTaskScheduler()
}

// ffmpegRun 纯执行层：只跑 ffmpeg 命令，不关心任务状态
func ffmpegRun(ctx context.Context, args []string) ([]byte, error) {
	ffmpegPath := ffmpegBinPath()
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}
	return cmd.CombinedOutput()
}

// ffmpegExec 编排层：管理任务生命周期 + 执行 ffmpeg + 回写结果
func ffmpegExec(args []string, taskKey time.Time) utils.Result {
	TransferTaskMutex.Lock()
	task, exists := TransferTask[taskKey]
	if !exists {
		TransferTaskMutex.Unlock()
		return utils.NewFailByMsg("任务不存在")
	}
	task.Status = model.StatusExecuting
	task.Command = ffmpegBinPath() + " " + strings.Join(args, " ")
	TransferTask[taskKey] = task
	TransferTaskMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	out, cmdErr := ffmpegRun(ctx, args)

	if cmdErr != nil {
		updateTaskStatus(taskKey, model.StatusFailed, string(out))
		utils.InfoFormat("命令执行失败: %v, 错误: %v, 参数: %v", string(out), cmdErr, args)
		return utils.NewFailByMsg("转换失败")
	}

	updateTaskStatus(taskKey, model.StatusCompleted, "")
	return utils.NewSuccessByMsg("转换成功")
}
