package wecont

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
)

var (
	pID_path string
	l        Logger
)

func Init(path string, infoLog *log.Logger, debugLog *log.Logger, errLog *log.Logger) (Wecont, error) {
	wc, err := ReadConfig(path)
	if err != nil {
		wc = Wecont{
			IsNull:   false,
			Programs: make(map[string]Program),
		}
	}
	wc.IsNull = false
	l = Logger{Info: infoLog, Debug: debugLog, Error: errLog}
	pID_path = path
	return wc, nil
}

// 启动子程序
func (wc Wecont) StartChild(programID string) (*exec.Cmd, error) {
	programObj, ok := wc.Programs[programID]
	if !ok {
		return nil, fmt.Errorf("no find program")
	}

	cmd := SetAttributes(programObj.Path)

	cmd.Stdout = l.Info.Writer()
	cmd.Stderr = l.Error.Writer()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cmd start failed: %s", err)
	}

	// 1. 删除旧文件（不检查错误，因为文件可能本就不存在）
	os.Remove(SocketAddr)

	if programObj.PID > 0 {
		err := killByPid(programObj.PID)
		if err != nil {
			l.Error.Printf("kill program %d failed: %s\n", programObj.PID, err)
		}
		os.Remove(subPID)
	}

	programObj.PID = cmd.Process.Pid
	wc.Programs[programID] = programObj

	return cmd, nil
}

func (wc Wecont) SendMsg(id string, cmd string) (string, error) {
	p, ok := wc.Programs[id]
	if !ok {
		return "", fmt.Errorf("program not found")
	}

	// 拨号连接 .sock 或管道
	conn, err := net.Dial(NetType, fmt.Sprintf("%s%s", p.Path, SocketAddr))
	if err != nil {
		return "", fmt.Errorf("link .sock failed: %s", err)
	}
	defer conn.Close()

	// 发送指令
	conn.Write([]byte(cmd + "\n"))

	// 读取回复
	reader := bufio.NewReader(conn)
	reply, err := reader.ReadString('\n')
	return reply, fmt.Errorf("read reply failed: %s", err)
}

func killByPid(pid int) error {
	// FindProcess 在 Windows 上不会检查进程是否存在
	// 在 Unix 上，它也只是建立一个进程对象的引用
	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("no find PID: %d, %s", pid, err)
	}

	// Kill 会直接强制结束进程
	err = p.Kill()
	if err != nil {
		return fmt.Errorf("failed to force terminate the process: %v", err)
	}

	return nil
}
