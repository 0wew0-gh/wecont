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
		pids, err := GetPidsByName([]GetProgramParams{{ID: pObj.ID, Name: pObj.FileName, Path: pObj.Path}})
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
	pObj.Message = ""
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

	findPIDs, err := GetPidsByName([]GetProgramParams{{ID: pObj.ID, Name: pObj.FileName, Path: pObj.Path}})
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
	pObj.Message = ""
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
		pids, err := GetPidsByName([]GetProgramParams{{ID: pObj.ID, Name: pObj.FileName, Path: pObj.Path}})
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
	pObj.Message = ""
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

func (wcc *WecontConfig) MonitorByPID(id []string) []ProgramInfo {
	if len(id) == 0 {
		return []ProgramInfo{}
	}
	wc := wcc.Get()

	gpp := []GetProgramParams{}
	for _, v := range id {
		pObj, ok := wc.Programs[v]
		if !ok {
			continue
		}

		pCmd, ok := wc.Cmd[v]
		if ok && pCmd == nil {
			pObj.PID = 0
			pObj.Status = STOP
			pObj.Message = ""
			wc.Programs[v] = pObj
			continue
		}
		gpp = append(gpp, GetProgramParams{
			ID:   pObj.ID,
			Name: pObj.FileName,
			Path: pObj.Path,
		})
	}
	if len(gpp) == 0 {
		wcc.UpdateProgram(wc.Programs)
		return nil
	}
	// return nil, fmt.Errorf("program has exited")

	pList, err := GetProcessByName(gpp)
	if err != nil {
		return nil
	}
	// if len(pList) == 0 {
	// 	return nil, fmt.Errorf("process not found")
	// }
	pInfoList := []ProgramInfo{}

	for _, p := range pList {
		pInfo := ProgramInfo{}

		temp, err := p.Name()
		if err == nil {
			pInfo.Name = temp
		}
		temp, err = p.Exe()
		if err == nil {
			pInfo.Path = temp
		}
		for _, v := range gpp {
			path := fmt.Sprintf("%s%s", v.Path, v.Name)
			if v.Name == pInfo.Name && path == pInfo.Path {
				pInfo.ID = v.ID
				break
			}
		}

		pInfo.PID = int(p.Pid)

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
	for _, v := range id {
		obj, ok := wc.Programs[v]
		if !ok {
			continue
		}
		pid := 0
		for _, p := range pInfoList {
			if p.ID == v {
				pid = p.PID
				break
			}
		}
		if pid > 0 {
			if obj.Status == STOP {
				obj.Status = RUN
			}
		} else {
			obj.Status = STOP
			obj.Message = ""
		}
		obj.PID = pid
		wc.Programs[v] = obj
	}
	wcc.UpdateProgram(wc.Programs)
	return pInfoList
}

func (wcc *WecontConfig) SendMsg(id string, cmd string) (string, error) {
	p, ok := wcc.Get().Programs[id]
	if !ok {
		return "", fmt.Errorf("program not found")
	}

	return p.sendMsg(cmd)
}

// 获取进程对象
func GetProcessByName(target []GetProgramParams) ([]*process.Process, error) {
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
		for _, v := range target {
			tPath := fmt.Sprintf("%s%s", v.Path, v.Name)
			// 匹配名称 (不区分大小写)
			if strings.EqualFold(name, v.Name) && strings.EqualFold(path, tPath) {
				pList = append(pList, p)
			}
		}
	}
	return pList, nil
}

// 获取进程ID
//
//	targetName	string	进程名称
//	targetPath	string	进程路径
func GetPidsByName(target []GetProgramParams) ([]int32, error) {
	var pids []int32

	pList, err := GetProcessByName(target)
	if err != nil {
		return nil, err
	}

	for _, v := range pList {
		pids = append(pids, v.Pid)
	}
	return pids, nil
}
