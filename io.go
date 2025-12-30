package wecont

import (
	"fmt"
	"io"
	"os"
)

// MARK: 创建文件夹
func checkFolder(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return fmt.Errorf("%s:%v", path, err)
		}
	}
	return nil
}

// MARK: 保存文件
func saveFile(path string, name string, content []byte) error {
	if len(content) == 0 {
		return fmt.Errorf("content is nil")
	}
	if path != "" {
		err := checkFolder(path)
		if err != nil {
			return err
		}
	}

	filePath := fmt.Sprintf("%s%s", path, name)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		f, err = os.Create(filePath)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return err
}

func openFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
