### 限流器 长期维护

### how to use it

```
    store, err := memstore.New(65535)
	if err != nil {
		log.Fatal(err)
	}
    
    f := NewFilter(store)
    
    quota := RateQuota{MaxRate: PerSec(5), MaxBurst: 1}
    key := "limit key"
    f.AddKey(key, quota)

    // when request :
    if limited, e := f.RateLimit(key);e == nil{
        if limited {
            // be limited
        } else {
            // adopt
        }
    }
    
```