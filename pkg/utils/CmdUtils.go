package utils

import (
	"os/exec"
	"runtime"
)

// ExecCmdStart 使用系统 start 命令打开路径
// 调用方必须先通过 ValidatePath 校验路径合法性
func ExecCmdStart(path string) int {
	return ExecCmd(path, "start")
}

// ExecCmd 执行系统命令
// 调用方必须确保 path 已通过 ValidatePath 校验
func ExecCmd(path string, cmdType string) int {
	InfoFormat("%v %v", cmdType, path)
	cmd := exec.Command("cmd", "/C", cmdType, "", path)
	if runtime.GOOS == "windows" {
		FixOnWin(cmd)
	}
	cmdErr := cmd.Start()
	if cmdErr != nil {
		InfoFormat("%v", cmdErr)
		return 0
	}
	InfoFormat("ExecCmdSuccess:%s , %s", cmdType, path)
	return 1
}
