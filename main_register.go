package wecont

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kagurazakayashi/libNyaruko_Go/nyacrypt"
)

func ReadConfig(db *badger.DB, programsIDs []string) (Wecont, error) {
	var configObj Programs
	for _, v := range programsIDs {
		val, err := badger_Read(db, []byte(v))
		if err != nil {
			continue
		}
		var temp Program
		err = json.Unmarshal(val, &temp)
		if err != nil {
			continue
		}
		configObj = append(configObj, temp)
	}

	var wc Wecont
	wc.Programs = make(map[string]Program)
	for _, v := range configObj {
		wc.Programs[v.ID] = v
	}

	return wc, nil
}

func (wc Wecont) SaveConfig(path string) error {
	if wc.DB == nil {
		db, err := badger_Link(path)
		if err != nil {
			return err
		}
		wc.DB = db
	}

	var configObj Programs
	for _, v := range wc.Programs {
		configObj = append(configObj, v)
	}
	sort.Sort(configObj)

	idList := []string{}
	errList := []error{}
	for i := 0; i < configObj.Len(); i++ {
		obj := configObj[i]
		idList = append(idList, obj.ID)
		configBytes, err := json.Marshal(configObj[i])
		if err != nil {
			errList = append(errList, fmt.Errorf("[%s]json.Marshal error: %+v", obj.ID, err))
			continue
		}
		err = badger_Write(wc.DB, []byte(idList[i]), configBytes)
		if err != nil {
			errList = append(errList, fmt.Errorf("[%s]badger write error: %+v", obj.ID, err))
			continue
		}
	}
	err := badger_Write(wc.DB, []byte("programs"), []byte(strings.Join(idList, ",")))
	if err != nil {
		errList = append(errList, fmt.Errorf("[programs]badger write error: %+v", err))
	}
	if len(errList) == 0 {
		return nil
	}
	errStr := ""
	for _, v := range errList {
		errStr += v.Error() + "\n"
	}
	return fmt.Errorf("%+v", errStr)
}

func (wc Wecont) RegisterProgram(c Config) (Wecont, string, error) {
	if c.Name == "" || c.FileName == "" || c.Path == "" {
		return wc, "", fmt.Errorf("invalid config")
	}
	for _, v := range wc.Programs {
		if v.Name == c.Name {
			return wc, "", fmt.Errorf("name already exists")
		}
		if v.FileName == c.FileName && v.Path == c.Path {
			return wc, "", fmt.Errorf("program already exists")
		}
	}

	filePath := fmt.Sprintf("%s%s", c.Path, c.Name)
	absPath, err := filepath.Abs(c.Path)
	if err != nil {
		return wc, "", fmt.Errorf("get abs path: %+v", err)
	}
	if !strings.HasSuffix(absPath, string(os.PathSeparator)) {
		absPath += string(os.PathSeparator)
	}

	tn := time.Now()
	id := nyacrypt.MD5String(filePath, fmt.Sprintf("%v", tn))
	newP := Program{Name: c.Name, FileName: c.FileName, Path: absPath, Status: STOP, Created: tn.UnixNano(), ID: id}

	wc.Programs[id] = newP

	err = wc.SaveConfig(pID_path)
	return wc, id, err
}

func (wc Wecont) RemoveProgram(id string) (Wecont, error) {
	p, ok := wc.Programs[id]
	if !ok {
		return wc, fmt.Errorf("program not found")
	}
	err := killByPid(p.PID)
	if err != nil {
		findPIDs, err := GetPidsByName(p.FileName, p.Path)
		if err != nil {
			return wc, err
		}

		for _, v := range findPIDs {
			if v == int32(p.PID) {
				l.Info.Printf("kill program %d\n", v)
				killByPid(int(v))
			}
		}
	}
	delete(wc.Programs, id)
	err = wc.SaveConfig(pID_path)
	return wc, err
}
