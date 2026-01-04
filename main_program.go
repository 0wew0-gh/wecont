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
	pObj, ok := wcc.Get().Programs[programID]
	if !ok {
		return nil, fmt.Errorf("no find program")
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
		return fmt.Errorf("no find program")
	}

	pObj.sendMsg("STOP")

	pObj.PID = 0
	wc.Programs[programID] = pObj
	wcc.SaveConfig(pID_path)

	os.Remove(fmt.Sprintf("%s%s", pObj.Path, SocketAddr))

	return nil
}

func (wcc *WecontConfig) KillChild(programID string) error {
	wc := wcc.Get()
	pObj, ok := wc.Programs[programID]
	if !ok {
		return fmt.Errorf("no find program")
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
	wc.Programs[programID] = pObj
	wcc.SaveConfig(pID_path)

	os.Remove(fmt.Sprintf("%s%s", pObj.Path, SocketAddr))

	return nil
}

func (wcc *WecontConfig) ReStartChild(programID string) (*exec.Cmd, error) {
	pObj, ok := wcc.Get().Programs[programID]
	if !ok {
		return nil, fmt.Errorf("no find program")
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

	return wcc.startChild(pObj)
}

func (wcc *WecontConfig) SetStatus(programID string, status string) error {
	wc := wcc.Get()
	pObj, ok := wc.Programs[programID]
	if !ok {
		return fmt.Errorf("no find program")
	}

	pObj.Status = status
	wc.Programs[programID] = pObj

	return nil
}

func (wcc *WecontConfig) GetStatus(programID string) string {
	pObj, ok := wcc.Get().Programs[programID]
	if !ok {
		return ""
	}

	return pObj.Status
}

func (wcc *WecontConfig) SendMsg(id string, cmd string) (string, error) {
	p, ok := wcc.Get().Programs[id]
	if !ok {
		return "", fmt.Errorf("program not found")
	}

	return p.sendMsg(cmd)
}

// 获取进程ID
//
//	targetName	string	进程名称
//	targetPath	string	进程路径
func GetPidsByName(targetName string, targetPath string) ([]int32, error) {
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
			pids = append(pids, p.Pid)
		}
	}
	return pids, nil
}
