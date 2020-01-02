package miser

import (
	"github.com/pantianying/miser/store/memstore"
	"log"
	"strconv"
	"testing"
	"time"
)

func TestFilter(t *testing.T) {
	store, err := memstore.New(65536)
	if err != nil {
		log.Fatal(err)
	}
	f := NewFilter(store)
	keyPrefix := "pantianying:"
	for i := 1; i < 10; i++ {
		// 每过1/5秒会有1个令牌生成
		quota := RateQuota{MaxRate: PerSec(5), MaxBurst: 1}
		key := keyPrefix + strconv.Itoa(i)
		e := f.AddKey(key, quota)
		if e != nil {
			t.Error(e)
		}
	}
	for i := 1; i < 10; i++ {
		j := 1
		for j < 20 {
			key := keyPrefix + strconv.Itoa(i)
			b, e := f.RateLimit(key)
			if e != nil {
				panic(e)
			}
			if b {
				t.Logf("limit false key:%v,count:%v,limit:%v", key, j, i)
			} else {
				t.Logf("limit ok key:%v,count:%v,limit:%v", key, j, i)
			}
			j++
			time.Sleep(100 * time.Millisecond)
		}

	}

}
