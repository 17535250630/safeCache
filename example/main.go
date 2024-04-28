package main

import (
	"safeCache"
)

type Enemy struct {
	Address string `json:"address,omitempty"`
	Old     int    `json:"old"`
}

func (self Enemy) CacheID() string {
	return self.Address
}

func main() {
	cache, err := safeCache.NewCacheManager[Enemy]("db/enemy", "enemy")
	if err != nil {
		panic(err.Error())
	}
	_ = cache
}
