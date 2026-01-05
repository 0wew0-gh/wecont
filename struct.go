package wecont

import (
	"log"
	"os/exec"
	"sync/atomic"

	"github.com/dgraph-io/badger/v4"
)

const (
	NetType    = "unix"
	SocketAddr = "wecont_link.sock"

	subPID = "pid"
)
const (
	RUN     = "run"
	WARNING = "warning"
	FAULT   = "fault"
	STOP    = "stop"
)

type Config struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	FileName string `json:"file_name"`
}

type Wecont struct {
	B        BadgerDB
	Cmd      map[string]*exec.Cmd `json:"cmd"`
	Programs map[string]Program   `json:"programs"`
}

type WecontConfig struct {
	value atomic.Pointer[Wecont]
}

func (cm *WecontConfig) Get() *Wecont {
	return cm.value.Load()
}

func copyProgram(p map[string]Program) map[string]Program {
	newP := make(map[string]Program)

	for k, v := range p {
		newP[k] = v
	}
	return newP
}
func copyCmd(p map[string]*exec.Cmd) map[string]*exec.Cmd {
	newP := make(map[string]*exec.Cmd)

	for k, v := range p {
		newP[k] = v
	}
	return newP
}

// 更新配置：需要修改 Map 时，必须全量替换
func (cm *WecontConfig) Update(db *badger.DB, mapC map[string]*exec.Cmd, mapP map[string]Program) {
	oldCfg := cm.Get()
	// 1. 创建新副本
	newCfg := &Wecont{
		B:        oldCfg.B,
		Cmd:      copyCmd(mapC),
		Programs: copyProgram(mapP),
	}
	if db != nil {
		if newCfg.B.DB != nil {
			newCfg.B.DB.Close()
		}
		newCfg.B.DB = db
	}

	// 3. 原子替换
	cm.value.Store(newCfg)
}

func (cm *WecontConfig) UpdateDB(db *badger.DB) {
	wc := cm.Get()
	if db != nil {
		if wc.B.DB != nil {
			wc.B.DB.Close()
		}
		wc.B.DB = db
	}
	cm.value.Store(wc)
}

func (cm *WecontConfig) UpdateCmd(p map[string]*exec.Cmd) {
	oldCfg := cm.Get()
	// 1. 创建新副本
	newCfg := &Wecont{
		B:        oldCfg.B,
		Cmd:      copyCmd(p),
		Programs: oldCfg.Programs,
	}
	cm.value.Store(newCfg)
}

func (cm *WecontConfig) UpdateProgram(p map[string]Program) {
	oldCfg := cm.Get()
	// 1. 创建新副本
	newCfg := &Wecont{
		B:        oldCfg.B,
		Cmd:      oldCfg.Cmd,
		Programs: copyProgram(p),
	}
	cm.value.Store(newCfg)
}

type Program struct {
	ID       string `json:"id"`
	PID      int    `json:"pid"`
	Name     string `json:"name"`
	FileName string `json:"file_name"`
	Path     string `json:"path"`
	Created  int64  `json:"created"`
	Status   string `json:"status"`
}
type Programs []Program

func (a Programs) Len() int           { return len(a) }
func (a Programs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Programs) Less(i, j int) bool { return a[i].Created < a[j].Created }

type ByNameASC struct {
	Programs
}

func (c ByNameASC) Less(i, j int) bool {
	return c.Programs[i].Name < c.Programs[j].Name
}

type ByNameDESC struct {
	Programs
}

func (c ByNameDESC) Less(i, j int) bool {
	return c.Programs[i].Name > c.Programs[j].Name
}

type Logger struct {
	Info  *log.Logger
	Debug *log.Logger
	Error *log.Logger
}

type ProgramInfo struct {
	Name   string
	Path   string
	CPU    float64
	Memory float64 // 字节 (Bytes)
	IO     IOCount
}

type IOCount struct {
	ReadCount  uint64 // 读操作次数
	WriteCount uint64 // 写操作次数
	ReadBytes  uint64 // 读取的总字节数
	WriteBytes uint64 // 写入的总字节数
}
