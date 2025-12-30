package wecont

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
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
		return nil, fmt.Errorf("cmd start failed: %v", err)
	}

	// 1. 删除旧文件（不检查错误，因为文件可能本就不存在）
	os.Remove(SocketAddr)

	if programObj.PID > 0 {
		err := killByPid(programObj.PID)
		if err != nil {
			l.Error.Printf("kill program %d failed: %v\n", programObj.PID, err)
		}
		os.Remove(subPID)
	}

	programObj.PID = cmd.Process.Pid
	wc.Programs[programID] = programObj

	wc.SaveConfig(pID_path)

	return cmd, nil
}

func (wc Wecont) StopChild(programID string) error {
	programObj, ok := wc.Programs[programID]
	if !ok {
		return fmt.Errorf("no find program")
	}

	programObj.sendMsg("STOP")

	pid := programObj.PID

	programObj.PID = 0
	wc.Programs[programID] = programObj
	wc.SaveConfig(pID_path)

	findPIDs, err := getPidsByName(programObj.Name, programObj.Path)
	if err != nil {
		return err
	}

	for _, v := range findPIDs {
		if v == int32(pid) {
			l.Info.Printf("kill program %d\n", v)
			killByPid(int(v))
		}
	}

	return nil
}

func (wc Wecont) SendMsg(id string, cmd string) (string, error) {
	p, ok := wc.Programs[id]
	if !ok {
		return "", fmt.Errorf("program not found")
	}

	return p.sendMsg(cmd)
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
	return reply, fmt.Errorf("read reply failed: %v", err)

}

func getPidsByName(targetName string, targetPath string) ([]int32, error) {
	var pids []int32

	// 获取所有进程列表
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("get pid faild:%v", err)
	}

	for _, p := range processes {
		// 获取进程名称
		name, err := p.Name()
		if err != nil {
			continue // 忽略权限不足或已退出的进程
		}

		path, err := p.Exe()
		if err != nil {
			continue
		}

		// 匹配名称 (不区分大小写)
		if strings.EqualFold(name, targetName) && strings.EqualFold(path, targetPath) {
			pids = append(pids, p.Pid)
		}
	}
	return pids, nil
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
