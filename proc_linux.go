//go:build !windows

package wecont

import (
	"os/exec"
	"syscall"
)

func SetAttributes(path string) *exec.Cmd {
	cmd := exec.Command(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新会话脱离父进程
	}
	return cmd
}
