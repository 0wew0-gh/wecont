package wecont

import (
	"log"

	"github.com/dgraph-io/badger/v4"
)

const (
	NetType    = "unix"
	SocketAddr = "sub_program.sock"

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
	IsNull   bool
	DB       *badger.DB
	Programs map[string]Program `json:"programs"`
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
