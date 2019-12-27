package miser

import "sync"

//filter分装了对GCRARateLimiter使用，用户可以实时控制流量

type filter struct {
	st GCRAStore

	// map[{限流key}]*GCRARateLimiter
	keyM sync.Map

	// 限流回调函数
	f func()
}

func NewFilter(store GCRAStore) *filter {
	return &filter{
		st: store,
	}
}
func (f *filter) AddKey(key string, quota RateQuota) {
	//todo
}
func (f *filter) UpdateKey(key string, quota RateQuota) {
	//todo
}
func (f *filter) RateLimit(key string) bool {
	//todo
}
