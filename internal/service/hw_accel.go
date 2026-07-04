package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"search-gin/pkg/utils"
	"strings"
	"sync"
	"sync/atomic"
)

// hwAccel 当前选中的硬件加速编码器状态
var hwAccel = struct {
	h264  string
	h265  string
	mode  string
	det   bool
	mu    sync.Mutex
	dec   string
	force bool
}{}

// hwAccelOption 一个硬件加速方案的完整信息
type hwAccelOption struct {
	name        string // 显示名称，如 "NVIDIA NVENC"
	h264Encoder string // h264_nvenc
	h265Encoder string // hevc_nvenc
	decode      string // -hwaccel cuda -hwaccel_output_format cuda
}

// availableHwAccels 扫描到的全部可用硬件加速方案列表（启动时填充）
var availableHwAccels atomic.Value // stores []hwAccelOption

// 所有候选硬件加速方案的定义（按优先级排序）
var hwAccelCandidates = []hwAccelOption{
	{"NVIDIA NVENC", "h264_nvenc", "hevc_nvenc", "-hwaccel cuda -hwaccel_output_format cuda"},
	{"AMD AMF", "h264_amf", "hevc_amf", "-hwaccel dxva2"},
	{"Intel QSV", "h264_qsv", "hevc_qsv", "-hwaccel qsv -hwaccel_output_format qsv"},
	{"VAAPI", "h264_vaapi", "hevc_vaapi", "-hwaccel vaapi -hwaccel_output_format vaapi"},
	{"VideoToolbox", "h264_videotoolbox", "hevc_videotoolbox", "-hwaccel videotoolbox"},
}

// InitHwAccelDetection 启动时扫描全部可用硬件加速方案
// 无论 HardwareAcceleration 是否启用，均执行扫描，以便前端展示可用选项
func InitHwAccelDetection() {
	scanAvailableHwAccels()
	utils.InfoFormat("硬件加速检测完成，可用方案: %v", GetAvailableHwAccelModes())
}

// scanAvailableHwAccels 调用 ffmpeg -encoders 扫描全部可用硬件加速方案
func scanAvailableHwAccels() {
	ffmpegPath := ffmpegBinPath()
	cmd := exec.Command(ffmpegPath, "-encoders")
	if runtime.GOOS == "windows" {
		utils.FixOnWin(cmd)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		utils.InfoFormat("扫描硬件加速失败(ffmpeg -encoders): %v", err)
		availableHwAccels.Store([]hwAccelOption{})
		return
	}
	output := string(out)

	var results []hwAccelOption
	for _, c := range hwAccelCandidates {
		h264Ok := strings.Contains(output, c.h264Encoder)
		h265Ok := strings.Contains(output, c.h265Encoder)
		if !h264Ok && !h265Ok {
			continue
		}
		if !verifyHwDeviceAvailable(c.decode) {
			utils.InfoFormat("硬件加速跳过 %s：驱动不可用", c.name)
			continue
		}
		results = append(results, c)
		utils.InfoFormat("硬件加速可用: %s", c.name)
	}
	if results == nil {
		results = []hwAccelOption{}
	}
	availableHwAccels.Store(results)
}

// GetAvailableHwAccelModes 返回可用硬件加速方案的显示名称列表（含 "auto"）
func GetAvailableHwAccelModes() []string {
	modes := []string{"auto"}
	list, ok := availableHwAccels.Load().([]hwAccelOption)
	if !ok {
		return modes
	}
	for _, opt := range list {
		modes = append(modes, opt.name)
	}
	return modes
}

// detectHwAccel 检测并选择硬件加速编码器
// 逻辑：
//  1. 调用 scanAvailableHwAccels 刷新可用方案
//  2. 如果用户的 HardwareAccelMode 指定了具体方案，选择该方案
//  3. 否则按优先级自动选择第一个可用方案
func detectHwAccel() {
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()

	if hwAccel.det && !hwAccel.force {
		return
	}
	hwAccel.force = false

	// 重置当前选中
	hwAccel.h264 = ""
	hwAccel.h265 = ""
	hwAccel.mode = ""
	hwAccel.dec = ""

	// 刷新可用方案列表
	scanAvailableHwAccels()

	list, ok := availableHwAccels.Load().([]hwAccelOption)
	if !ok || len(list) == 0 {
		utils.InfoFormat("未检测到任何可用的硬件加速方案，将使用软件编码")
		hwAccel.det = true
		return
	}

	selectedMode := GetOSSetting().HardwareAccelMode

	// 用户选择了特定方案 → 匹配
	if selectedMode != "" && selectedMode != "auto" {
		for _, opt := range list {
			if opt.name == selectedMode {
				applyHwAccelOption(opt)
				utils.InfoFormat("使用用户指定的硬件加速模式: %s", selectedMode)
				hwAccel.det = true
				return
			}
		}
		utils.InfoFormat("用户指定的硬件加速模式 %s 已不可用，回退自动选择", selectedMode)
	}

	// 自动选择：取第一个（优先级最高）
	opt := list[0]
	applyHwAccelOption(opt)
	utils.InfoFormat("自动选择硬件加速模式: %s", opt.name)
	hwAccel.det = true
}

func applyHwAccelOption(opt hwAccelOption) {
	hwAccel.h264 = opt.h264Encoder
	hwAccel.h265 = opt.h265Encoder
	hwAccel.mode = opt.name
	hwAccel.dec = opt.decode
}

// getH264Encoder 获取当前应使用的 H264 编码器
func getH264Encoder() string {
	if GetOSSetting().HardwareAcceleration {
		detectHwAccel()
		hwAccel.mu.Lock()
		defer hwAccel.mu.Unlock()
		if hwAccel.h264 != "" {
			return hwAccel.h264
		}
	}
	return "libx264"
}

// getH265Encoder 获取当前应使用的 H265 编码器
func getH265Encoder() string {
	if GetOSSetting().HardwareAcceleration {
		detectHwAccel()
		hwAccel.mu.Lock()
		defer hwAccel.mu.Unlock()
		if hwAccel.h265 != "" {
			return hwAccel.h265
		}
	}
	return "libx265"
}

// GetHwAccelModeName 暴露当前选中的硬件加速模式名称给外部
func GetHwAccelModeName() string {
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()
	return hwAccel.mode
}

// getHwDecodeParams 获取硬件解码参数（在 -i 之前插入）
func getHwDecodeParams() string {
	if GetOSSetting().HardwareAcceleration {
		detectHwAccel()
		hwAccel.mu.Lock()
		defer hwAccel.mu.Unlock()
		if hwAccel.dec != "" {
			return hwAccel.dec
		}
	}
	return ""
}

// getHwQualityParam 获取硬件编码器的质量参数
func getHwQualityParam() string {
	if GetOSSetting().HardwareAcceleration {
		detectHwAccel()
		hwAccel.mu.Lock()
		defer hwAccel.mu.Unlock()
		if hwAccel.h264 != "" || hwAccel.h265 != "" {
			return "-q"
		}
	}
	return "-crf"
}

// HwAccelSettingChanged 检查硬件加速开关是否发生变化（与上次保存时不同）
var lastHwAccelSetting bool

func HwAccelSettingChanged() bool {
	current := GetOSSetting().HardwareAcceleration
	hwAccel.mu.Lock()
	defer hwAccel.mu.Unlock()
	if lastHwAccelSetting != current {
		lastHwAccelSetting = current
		return true
	}
	return false
}

// HwAccelModeChanged 检查硬件加速模式选择是否发生变化
var lastHwAccelMode string

func HwAccelModeChanged() bool {
	current := GetOSSetting().HardwareAccelMode
	if lastHwAccelMode != current {
		lastHwAccelMode = current
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

// verifyHwDeviceAvailable 运行时验证硬件设备驱动是否实际可用
func verifyHwDeviceAvailable(decodeParams string) bool {
	switch {
	case strings.Contains(decodeParams, "cuda"):
		if runtime.GOOS == "windows" {
			return dllExists("nvcuda.dll")
		}
		if runtime.GOOS == "linux" {
			return utils.ExistsFiles("/dev/nvidiactl")
		}
		return false
	case strings.Contains(decodeParams, "dxva2"):
		if runtime.GOOS != "windows" {
			return false
		}
		return dllExists("amfrt64.dll")
	case strings.Contains(decodeParams, "qsv"):
		if runtime.GOOS == "windows" {
			return dllExists("libmfxhw64.dll")
		}
		return utils.ExistsFiles("/dev/dri/renderD128")
	case strings.Contains(decodeParams, "vaapi"):
		return runtime.GOOS == "linux" && utils.ExistsFiles("/dev/dri/renderD128")
	case strings.Contains(decodeParams, "videotoolbox"):
		return runtime.GOOS == "darwin"
	}
	return true
}

// dllExists 检查 Windows System32 下驱动 DLL 是否存在
func dllExists(name string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	sysDir := os.Getenv("SystemRoot")
	if sysDir == "" {
		sysDir = "C:\\Windows"
	}
	return utils.ExistsFiles(filepath.Join(sysDir, "System32", name))
}
