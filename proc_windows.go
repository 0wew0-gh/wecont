//go:build windows

package wecont

import (
	"os/exec"
	"syscall"
)

func SetAttributes(path string) *exec.Cmd {
	cmd := exec.Command(path)
	const DETACHED_PROCESS = 0x00000008
	const CREATE_NEW_PROCESS_GROUP = 0x00000200

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP,
	}
	return cmd
}
