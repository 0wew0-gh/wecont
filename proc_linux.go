//go:build !windows

package wecont

import (
	"fmt"
	"os/exec"
	"syscall"
)

func SetAttributes(path string, fileName string) *exec.Cmd {
	cmd := exec.Command(fmt.Sprintf("%s%s", path, fileName))
	cmd.Dir = path
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新会话脱离父进程
	}
	return cmd
}
