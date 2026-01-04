package wecont

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
)

func (wc Wecont) startChild(programObj Program) (*exec.Cmd, error) {
	// 1. 删除旧文件（不检查错误，因为文件可能本就不存在）
	os.Remove(fmt.Sprintf("%s%s", programObj.Path, SocketAddr))

	cmd := SetAttributes(programObj.Path, programObj.FileName)

	cmd.Stdout = l.Info.Writer()
	cmd.Stderr = l.Error.Writer()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cmd start failed: %v", err)
	}

	programObj.PID = cmd.Process.Pid
	wc.Programs[programObj.ID] = programObj

	wc.SaveConfig(pID_path)

	return cmd, nil
}

func (p Program) sendMsg(cmd string) (string, error) {
	// 拨号连接 .sock 或管道
	conn, err := net.Dial(NetType, fmt.Sprintf("%s%s", p.Path, SocketAddr))
	if err != nil {
		return "", fmt.Errorf("link .sock failed: %v", err)
	}
	defer conn.Close()

	// 发送指令
	conn.Write([]byte(cmd + "\n"))

	// 读取回复
	reader := bufio.NewReader(conn)
	reply, err := reader.ReadString('\n')
	if err != nil {
		return reply, fmt.Errorf("read reply failed: %v", err)
	}
	return reply, nil

}

func killByPid(pid int) error {
	// FindProcess 在 Windows 上不会检查进程是否存在
	// 在 Unix 上，它也只是建立一个进程对象的引用
	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("no find PID: %d, %v", pid, err)
	}

	// Kill 会直接强制结束进程
	err = p.Kill()
	if err != nil {
		return fmt.Errorf("failed to force terminate the process: %v", err)
	}

	return nil
}
