package safeCache

import (
	"encoding/json"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

type CacheData interface {
	CacheID() string
}

type CacheManager[T CacheData] struct {
	dbKey  string
	dbPath string
	dbLock *sync.Mutex
	cache  []T
}

func NewCacheManager[T CacheData](dbPath string, dbKey string) (*CacheManager[T], error) {
	mgr := &CacheManager[T]{
		dbKey:  dbKey,
		dbLock: &sync.Mutex{},
		dbPath: dbPath,
	}
	db, err := mgr.InitCacheFromDB()
	if err != nil {
		return nil, err
	}
	mgr.cache = db
	return mgr, nil
}

func (self *CacheManager[T]) Update(data T) {
	self.DeleteID(data.CacheID())
	self.PushCache(data)
}

func (self *CacheManager[T]) Has(id string) bool {
	for _, c := range self.cache {
		if c.CacheID() == id {
			return true
		}
	}
	return false
}

func (self *CacheManager[T]) DeleteID(id string) {
	self.dbLock.Lock()
	defer self.dbLock.Unlock()
	for i, c := range self.cache {
		if c.CacheID() == id {
			if i < len(self.cache)-1 {
				self.cache = append(self.cache[:i], self.cache[i+1:]...)
			} else if i == len(self.cache)-1 {
				self.cache = self.cache[:i]
			}
		}
	}
	self.SaveToDB()
}

func (self *CacheManager[T]) PushCache(data T) {
	self.dbLock.Lock()
	defer self.dbLock.Unlock()
	if self.Has(data.CacheID()) {
		return
	}
	self.cache = append(self.cache, data)
	self.SaveToDB()
}

func (self *CacheManager[T]) GetCache() []T {
	return self.cache
}

func (self *CacheManager[T]) SaveToDB() {
	db, err := badger.Open(badger.DefaultOptions(self.dbPath).WithLoggingLevel(badger.WARNING))
	if err != nil {
		panic(err.Error())
	}
	defer func(db *badger.DB) {
		er := db.Close()
		if er != nil {
			return
		}
	}(db)
	self.SaveCacheInfoToDB(db, self.cache)
}

func (self *CacheManager[T]) InitCacheFromDB() ([]T, error) {
	db, err := badger.Open(badger.DefaultOptions(self.dbPath).WithLoggingLevel(badger.WARNING))
	if err != nil {
		return nil, err
	}
	defer func(db *badger.DB) {
		er := db.Close()
		if er != nil {
			return
		}
	}(db)
	return self.GetCacheInfosFromDB(db)
}

func (self *CacheManager[T]) GetCacheInfosFromDB(db *badger.DB) ([]T, error) {
	var cachedData []T
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		cachedValue, err := txn.Get([]byte(self.dbKey))
		if err != nil {
			return nil
		}
		var value []byte
		data, _ := cachedValue.ValueCopy(value)
		err = json.Unmarshal(data, &cachedData)
		if err != nil {
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cachedData, nil
}

func (self *CacheManager[T]) SaveCacheInfoToDB(db *badger.DB, data []T) {
	err := db.Update(func(txn *badger.Txn) error {
		v, err := json.Marshal(data)
		if err != nil {
			return err
		}
		err = txn.Set([]byte(self.dbKey), v)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err.Error())
	}
}
