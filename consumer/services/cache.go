package services

import (
	"github.com/patrickmn/go-cache"
	"time"
)

var _cache = cache.New(15*time.Minute, 5*time.Minute)

func SetValue(key string, value string) {
	_cache.Set(key, &value, cache.DefaultExpiration)
}

func GetValue(key string) (*string, bool) {
	value, found := _cache.Get(key)

	if found {
		return value.(*string), true
	} else {
		return nil, false
	}
}
