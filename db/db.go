package db

import (
	"encoding/json"

	"github.com/xuperchain/xupercore/lib/storage/kvdb"
	"github.com/xuperchain/xupercore/lib/storage/kvdb/leveldb"
)

type DB struct {
	db kvdb.Database
}

func New() *DB {
	param := &kvdb.KVParameter{
		DBPath:                "./data",
		KVEngineType:          kvdb.KVEngineTypeLDB,
		StorageType:           kvdb.StorageTypeSingle,
		MemCacheSize:          128,
		FileHandlersCacheSize: 1024,
	}
	ldb, err := leveldb.NewKVDBInstance(param)
	if err != nil {
		panic(err)
	}
	db := &DB{
		db: ldb,
	}
	return db
}

func (db *DB) Close() {
	db.db.Close()
}

func (db *DB) Set(key, value []byte) (err error) {
	return db.db.Put(key, value)
}

func (db *DB) Get(key []byte) (value []byte, err error) {
	return db.db.Get(key)
}
func (db *DB) Del(key []byte) error {
	return db.db.Delete(key)
}

func (db *DB) SetLatestHeight(sideChainName string, height int64) (err error) {
	v, _ := json.Marshal(height)
	return db.db.Put([]byte(sideChainName+"_latestheight"), v)
}

func (db *DB) GetLatestHeight(sideChainName string) (int64, error) {
	key := []byte(sideChainName + "_latestheight")
	value, err := db.db.Get(key)
	if err != nil {
		return 0, err
	}
	h := int64(0)
	err = json.Unmarshal(value, &h)
	return h, err

}
