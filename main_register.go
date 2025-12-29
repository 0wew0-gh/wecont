package wecont

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/kagurazakayashi/libNyaruko_Go/nyacrypt"
)

func ReadConfig() (Wecont, error) {
	scStr, err := openFile(subPID)
	if err != nil {
		return Wecont{}, err
	}
	var wc Wecont
	var configObj Programs
	err = json.Unmarshal([]byte(scStr), &configObj)
	if err != nil {
		return wc, err
	}

	wc.Programs = make(map[string]Program)
	for _, v := range configObj {
		wc.Programs[v.ID] = v
	}

	return wc, nil
}

func (wc Wecont) SaveConfig() error {
	var configObj Programs
	for _, v := range wc.Programs {
		configObj = append(configObj, v)
	}
	sort.Sort(configObj)

	configBytes, err := json.Marshal(wc.Programs)
	if err != nil {
		return err
	}

	return saveFile("", subPID, configBytes)
}

func (wc Wecont) RegisterProgram(c Config) (Wecont, error) {
	tn := time.Now()
	id := nyacrypt.MD5String(fmt.Sprintf("%s-%s", c.Path, c.Name), fmt.Sprintf("%v", tn))
	newP := Program{Name: c.Name, Path: c.Path, Status: STOP, Created: tn.UnixNano(), ID: id}

	wc.Programs[id] = newP

	err := wc.SaveConfig()
	return wc, err
}

func (wc Wecont) RemoveProgram(id string) (Wecont, error) {
	p, ok := wc.Programs[id]
	if !ok {
		return wc, fmt.Errorf("program not found")
	}
	err := killByPid(p.PID)
	if err != nil {
		return wc, err
	}
	delete(wc.Programs, id)
	err = wc.SaveConfig()
	return wc, err
}
