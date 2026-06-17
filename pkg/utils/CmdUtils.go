package utils

import (
	"os/exec"
	"runtime"
)

func ExecCmdStart(path string) int {
	return ExecCmd(path, "start")
}

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
