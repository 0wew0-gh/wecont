package wecont

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

var (
	pID_path string
	l        Logger
)

func Init(path string, infoLog *log.Logger, debugLog *log.Logger, errLog *log.Logger) (*WecontConfig, error) {
	db, err := badger_Link(path)
	if err != nil {
		return &WecontConfig{}, err
	}
	bDB := BadgerDB{DB: db}
	programsIDs := bDB.ReadIDList()

	wcc, _ := bDB.ReadConfig(programsIDs)

	wc := wcc.Get()

	wc.B.DB = db
	l = Logger{Info: infoLog, Debug: debugLog, Error: errLog}
	pID_path = path
	return wcc, nil
}

// 启动子程序
func (wcc *WecontConfig) StartChild(programID string) (*exec.Cmd, error) {
	wc := wcc.Get()
	pObj, ok := wc.Programs[programID]
	if !ok {
		return nil, fmt.Errorf("program not found")
	}
	cmdObj, ok := wc.Cmd[programID]
	if ok && cmdObj != nil {
		// 获取运行中进程状态
		return nil, fmt.Errorf("program has started, pid: %d", cmdObj.Process.Pid)
	}

	if pObj.PID > 0 {
		pids, err := GetPidsByName(pObj.FileName, pObj.Path)
		killPids := []int{}
		if err != nil {
			killPids = append(killPids, pObj.PID)
		} else {
			for _, v := range pids {
				if int(v) == pObj.PID {
					killPids = append(killPids, pObj.PID)
				}
			}
		}
		if len(killPids) > 0 {
			return nil, fmt.Errorf("program has started")
		}
	}

	return wcc.startChild(pObj)
}

func (wcc *WecontConfig) StopChild(programID string) error {
	wc := wcc.Get()
	pObj, ok := wc.Programs[programID]
	if !ok {
		return fmt.Errorf("program not found")
	}

	_, err := pObj.sendMsg("STOP")
	if err != nil {
		cmdObj, ok := wc.Cmd[programID]
		if !ok {
			return fmt.Errorf("program has exited")
		}
		if cmdObj == nil {
			return fmt.Errorf("program has exited")
		}
		err := cmdObj.Process.Signal(os.Interrupt)
		if err != nil {
			err = cmdObj.Process.Kill()
		}
		if err != nil {
			return err
		}

	}

	pObj.PID = 0
	pObj.Status = STOP
	wc.Programs[programID] = pObj
	wcc.SaveConfig(pID_path)
	wcc.UpdateProgram(wc.Programs)

	os.Remove(fmt.Sprintf("%s%s", pObj.Path, SocketAddr))

	return nil
}

func (wcc *WecontConfig) KillChild(programID string) error {
	wc := wcc.Get()
	pObj, ok := wc.Programs[programID]
	if !ok {
		return fmt.Errorf("program not found")
	}

	pid := pObj.PID

	findPIDs, err := GetPidsByName(pObj.FileName, pObj.Path)
	if err != nil {
		return err
	}

	for _, v := range findPIDs {
		if v == int32(pid) {
			l.Info.Printf("kill program %d\n", v)
			killByPid(int(v))
		}
	}

	pObj.PID = 0
	pObj.Status = STOP
	wc.Programs[programID] = pObj
	wcc.SaveConfig(pID_path)
	wcc.UpdateProgram(wc.Programs)

	os.Remove(fmt.Sprintf("%s%s", pObj.Path, SocketAddr))

	return nil
}

func (wcc *WecontConfig) ReStartChild(programID string) (*exec.Cmd, error) {
	wc := wcc.Get()
	pObj, ok := wc.Programs[programID]
	if !ok {
		return nil, fmt.Errorf("program not found")
	}
	cmdObj, ok := wc.Cmd[programID]
	if ok && cmdObj != nil {
		err := cmdObj.Process.Signal(os.Interrupt)
		if err != nil {
			cmdObj.Process.Kill()
		}
		pObj.PID = 0
		cmdObj = nil
	}

	if pObj.PID > 0 {
		pids, err := GetPidsByName(pObj.FileName, pObj.Path)
		killPids := []int{}
		if err != nil {
			killPids = append(killPids, pObj.PID)
		} else {
			for _, v := range pids {
				if int(v) == pObj.PID {
					killPids = append(killPids, pObj.PID)
				}
			}
		}
		for _, v := range killPids {
			killByPid(v)
		}
	}
	pObj.PID = 0
	pObj.Status = STOP
	wc.Programs[programID] = pObj
	wcc.UpdateProgram(wc.Programs)

	return wcc.startChild(pObj)
}

func (wcc *WecontConfig) SetStatus(programID string, status string) error {
	wc := wcc.Get()
	pObj, ok := wc.Programs[programID]
	if !ok {
		return fmt.Errorf("program not found")
	}

	pObj.Status = status
	wc.Programs[programID] = pObj
	wcc.UpdateProgram(wc.Programs)

	return nil
}

func (wcc *WecontConfig) GetStatus(programID string) string {
	pObj, ok := wcc.Get().Programs[programID]
	if !ok {
		return ""
	}

	return pObj.Status
}

func (wcc *WecontConfig) SetMessage(programID string, message string) error {
	wc := wcc.Get()
	pObj, ok := wc.Programs[programID]
	if !ok {
		return fmt.Errorf("program not found")
	}

	pObj.Message = message
	wc.Programs[programID] = pObj
	wcc.UpdateProgram(wc.Programs)

	return nil
}

func (wcc *WecontConfig) GetMessage(programID string) string {
	pObj, ok := wcc.Get().Programs[programID]
	if !ok {
		return ""
	}

	return pObj.Message
}

func (wcc *WecontConfig) MonitorByPID(id string) ([]ProgramInfo, error) {
	wc := wcc.Get()
	pObj, ok := wc.Programs[id]
	if !ok {
		return nil, fmt.Errorf("program not found")
	}

	pCmd, ok := wc.Cmd[id]
	if ok && pCmd == nil {
		return nil, fmt.Errorf("program has exited")
	}

	pList, err := GetProcessByName(pObj.FileName, pObj.Path)
	if err != nil {
		return nil, fmt.Errorf("get process failed: %v", err)
	}
	if len(pList) == 0 {
		return nil, fmt.Errorf("process not found")
	}
	pInfoList := []ProgramInfo{}

	for _, p := range pList {
		pInfo := ProgramInfo{
			Name: pObj.FileName,
			Path: pObj.Path,
		}
		pCPUpercent, err := p.CPUPercent()
		if err == nil {
			pInfo.CPU = pCPUpercent
		}

		pMemInfo, err := p.MemoryInfo()
		if err == nil {
			pInfo.Memory = float64(pMemInfo.RSS)
		}

		pIO, err := p.IOCounters()
		if err == nil {
			pInfo.IO.ReadBytes = pIO.ReadBytes
			pInfo.IO.ReadCount = pIO.ReadCount
			pInfo.IO.WriteBytes = pIO.WriteBytes
			pInfo.IO.WriteCount = pIO.WriteCount
		}
		pInfoList = append(pInfoList, pInfo)
	}
	return pInfoList, nil
}

func (wcc *WecontConfig) SendMsg(id string, cmd string) (string, error) {
	p, ok := wcc.Get().Programs[id]
	if !ok {
		return "", fmt.Errorf("program not found")
	}

	return p.sendMsg(cmd)
}

// 获取进程对象
func GetProcessByName(targetName string, targetPath string) ([]*process.Process, error) {
	pList := []*process.Process{}

	// 获取所有进程列表
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	for _, p := range processes {
		// 获取进程名称
		name, err := p.Name()
		if err != nil {
			// l.Error.Printf("get process [name] failed: %v\n", err)
			continue // 忽略权限不足或已退出的进程
		}

		path, err := p.Exe()
		if err != nil {
			// l.Error.Printf("get process [path] failed: %v\n", err)
			continue
		}

		tPath := fmt.Sprintf("%s%s", targetPath, targetName)
		// 匹配名称 (不区分大小写)
		if strings.EqualFold(name, targetName) && strings.EqualFold(path, tPath) {
			pList = append(pList, p)
		}
	}
	return pList, nil
}

// 获取进程ID
//
//	targetName	string	进程名称
//	targetPath	string	进程路径
func GetPidsByName(targetName string, targetPath string) ([]int32, error) {
	var pids []int32

	pList, err := GetProcessByName(targetName, targetPath)
	if err != nil {
		return nil, err
	}

	for _, v := range pList {
		pids = append(pids, v.Pid)
	}
	return pids, nil
}
