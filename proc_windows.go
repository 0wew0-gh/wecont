//go:build windows

package wecont

import (
	"fmt"
	"os/exec"
	"syscall"
)

func SetAttributes(path string, fileName string) *exec.Cmd {
	cmd := exec.Command(fmt.Sprintf("%s%s", path, fileName))
	cmd.Dir = path
	const DETACHED_PROCESS = 0x00000008
	const CREATE_NEW_PROCESS_GROUP = 0x00000200

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP,
	}
	return cmd
}
