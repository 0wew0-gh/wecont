package wecont

import (
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

func badger_Link(path string) (*badger.DB, error) {
	// 1. 打开数据库（如果文件夹不存在会自动创建）
	// 在闪存上使用时，建议开启压缩
	opts := badger.DefaultOptions(fmt.Sprintf("%s%s", path, subPID)).
		WithCompression(options.ZSTD). // 减少闪存写入压力
		WithLoggingLevel(badger.ERROR)
		// WithNumLevelZeroTables(5) // L0 层触发合并的文件数量
	opts.SyncWrites = false // 不再每次写入都强制落盘，由操作系统缓冲，大幅延长闪存寿命

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %v", err)
	}

	return db, nil
}

func badger_ReadIDList(db *badger.DB) []string {
	val, err := badger_Read(db, []byte("programs"))
	if err != nil {
		return []string{}
	}
	valStr := string(val)
	return strings.Split(valStr, ",")
}
func badger_Read(db *badger.DB, key []byte) ([]byte, error) {
	value := []byte{}
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		// 处理读取到的数据
		return item.Value(func(val []byte) error {
			value = val
			return nil
		})
	})
	return value, err
}

func badger_Write(db *badger.DB, key []byte, value []byte) error {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	return err
}
