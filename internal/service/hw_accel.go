package service

import (
	"os/exec"
	"runtime"
	"search-gin/pkg/consts"
	"search-gin/pkg/utils"
	"strings"
	"sync"
)

// hwAccel 硬件加速状态（惰性检测，首次转码时自动识别）
var hwAccel = struct {
	h264  string
	h265  string
	mode  string
	det   bool
	mu    sync.Mutex
	dec   string
	force bool
}{}

// detectHwAccel 检测平台上可用的最佳硬件编码器（惰性调用，首次转码时自动识别）
func (e *videoEncoder) detectHwAccel() {
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()

	if hwAccel.det && !hwAccel.force {
		return
	}
	forceDetect := hwAccel.force
	hwAccel.force = false

	if forceDetect {
		hwAccel.h264 = ""
		hwAccel.h265 = ""
		hwAccel.mode = ""
		hwAccel.dec = ""
	}

	ffmpegPath := e.ffmpegBinPath()
	cmd := exec.Command(ffmpegPath, "-encoders")
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		utils.InfoFormat("硬件加速检测失败(ffmpeg -encoders): %v", err)
		return
	}

	output := string(out)

	type hwEncoder struct {
		h264   string
		h265   string
		name   string
		decode string
	}
	encoders := []hwEncoder{
		{"h264_nvenc", "hevc_nvenc", "NVIDIA NVENC", "-hwaccel cuda -hwaccel_output_format cuda"},
		{"h264_amf", "hevc_amf", "AMD AMF", "-hwaccel dxva2"},
		{"h264_qsv", "hevc_qsv", "Intel QSV", "-hwaccel qsv -hwaccel_output_format qsv"},
		{"h264_vaapi", "hevc_vaapi", "VAAPI", "-hwaccel vaapi -hwaccel_output_format vaapi"},
		{"h264_videotoolbox", "hevc_videotoolbox", "VideoToolbox", "-hwaccel videotoolbox"},
	}

	for _, e := range encoders {
		h264Ok := strings.Contains(output, e.h264)
		h265Ok := strings.Contains(output, e.h265)
		if h264Ok && h265Ok {
			hwAccel.h264 = e.h264
			hwAccel.h265 = e.h265
			hwAccel.mode = e.name
			hwAccel.dec = e.decode
			hwAccel.det = true
			utils.InfoFormat("硬件加速检测成功: %s (h264=%s, h265=%s) 解码参数=%s", e.name, e.h264, e.h265, e.decode)
			return
		}
	}

	for _, e := range encoders {
		if strings.Contains(output, e.h264) {
			hwAccel.h264 = e.h264
			hwAccel.mode = e.name
			hwAccel.dec = e.decode
			hwAccel.det = true
			utils.InfoFormat("硬件加速部分检测成功(仅H264): %s", e.name)
			return
		}
	}

	utils.InfoFormat("未检测到任何硬件加速编码器，将使用软件编码")
	hwAccel.det = true
}

// getH264Encoder 获取当前应使用的 H264 编码器
func (e *videoEncoder) getH264Encoder() string {
	if consts.GetOSSetting().HardwareAcceleration {
		e.detectHwAccel()
		if hwAccel.h264 != "" {
			return hwAccel.h264
		}
	}
	return "libx264"
}

// getH265Encoder 获取当前应使用的 H265 编码器
func (e *videoEncoder) getH265Encoder() string {
	if consts.GetOSSetting().HardwareAcceleration {
		e.detectHwAccel()
		if hwAccel.h265 != "" {
			return hwAccel.h265
		}
	}
	return "libx265"
}

// GetHwAccelModeName 暴露硬件加速模式名称给外部
func GetHwAccelModeName() string {
	return hwAccel.mode
}

// getHwDecodeParams 获取硬件解码参数（在 -i 之前插入）
func (e *videoEncoder) getHwDecodeParams() string {
	if consts.GetOSSetting().HardwareAcceleration {
		e.detectHwAccel()
		if hwAccel.dec != "" {
			return hwAccel.dec
		}
	}
	return ""
}

// getHwQualityParam 获取硬件编码器的质量参数
func (e *videoEncoder) getHwQualityParam() string {
	if consts.GetOSSetting().HardwareAcceleration {
		e.detectHwAccel()
		if hwAccel.h264 != "" || hwAccel.h265 != "" {
			return "-q"
		}
	}
	return "-crf"
}

// HwAccelSettingChanged 检查硬件加速设置是否发生变化（与上次保存时不同）
var lastHwAccelSetting bool

func HwAccelSettingChanged() bool {
	current := consts.GetOSSetting().HardwareAcceleration
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()
	if lastHwAccelSetting != current {
		lastHwAccelSetting = current
		return true
	}
	return false
}

// ForceHwAccelDetect 强制下次转码时重新检测硬件加速
func ForceHwAccelDetect() {
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()
	hwAccel.force = true
	hwAccel.det = false
	utils.InfoFormat("硬件加速设置已更改，下次转码时将重新检测")
}
