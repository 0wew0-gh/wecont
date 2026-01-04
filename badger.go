package wecont

import (
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

type BadgerDB struct {
	DB *badger.DB
}

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

func (b BadgerDB) ReadIDList() []string {
	val, err := b.Read([]byte("programs"))
	if err != nil {
		return []string{}
	}
	valStr := string(val)
	return strings.Split(valStr, ",")
}
func (b BadgerDB) Read(key []byte) ([]byte, error) {
	value := []byte{}
	err := b.DB.View(func(txn *badger.Txn) error {
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

func (b BadgerDB) Write(key []byte, value []byte) error {
	err := b.DB.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	return err
}
