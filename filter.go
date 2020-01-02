package miser

import (
	"sync"
)

//filter分装了对GCRARateLimiter使用，用户可以实时控制流量

type Filter struct {
	st GCRAStore

	// map[{限流key}]*GCRARateLimiter
	keyM sync.Map

	// 限流回调函数
	f func(key string)
}

func NewFilter(store GCRAStore) *Filter {
	return &Filter{
		st: store,
	}
}

// 对外暴露的限流方法，false为通过
func (f *Filter) RateLimit(key string) (bool, error) {
	if l, ok := f.GetRateLimiter(key); ok {
		b, _, e := l.RateLimit(key, 1)
		if e == nil {
			if b && f.f != nil {
				go f.f(key)
			}
			return b, nil
		}
		return false, e
	}
	//默认通过
	return false, nil
}

func (f *Filter) AddKey(key string, quota RateQuota) error {
	rateLimiter, err := NewGCRARateLimiter(f.st, quota)
	if err != nil {
		return err
	}
	f.keyM.Store(key, rateLimiter)
	return nil
}
func (f *Filter) UpdateKey(key string, quota RateQuota) error {
	rateLimiter, err := NewGCRARateLimiter(f.st, quota)
	if err != nil {
		return err
	}
	f.keyM.Store(key, rateLimiter)
	return nil
}
func (f *Filter) DeleteKey(key string) {
	f.keyM.Delete(key)
}
func (f *Filter) GetRateLimiter(key string) (*GCRARateLimiter, bool) {
	l, ok := f.keyM.Load(key)
	if !ok {
		return nil, false
	}
	return l.(*GCRARateLimiter), true
}
func (f *Filter) Clean() {
	f.keyM = sync.Map{}
}
