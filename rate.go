package miser

import (
	"fmt"
	"time"
)

const (
	//CAS重试最大次数
	maxCASAttempts = 10
)

type RateLimiter interface {
	RateLimit(key string, quantity int) (bool, RateLimitResult, error)
}

type RateLimitResult struct {
	//允许请求数
	Limit int

	//还剩下的令牌数
	Remaining int

	//某个计时周期内，还剩下的时间
	ResetAfter time.Duration

	//是允许下一个请求之前的时间
	//没有被限制时是-1
	RetryAfter time.Duration
}

type limitResult struct {
	limited bool
}

func (r *limitResult) Limited() bool { return r.limited }

type rateLimitResult struct {
	limitResult

	limit, remaining  int
	reset, retryAfter time.Duration
}

func (r *rateLimitResult) Limit() int                { return r.limit }
func (r *rateLimitResult) Remaining() int            { return r.remaining }
func (r *rateLimitResult) Reset() time.Duration      { return r.reset }
func (r *rateLimitResult) RetryAfter() time.Duration { return r.retryAfter }

// Rate表述请求速率，单位时间内允许的请求数量
type Rate struct {
	period time.Duration
	count  int
}

// RateQuota描述每个时间段允许的请求数。
// MaxRate指定请求的最大持续速率，并且必须大于0。
// MaxBurst定义将允许在单个突发中超过速率，并且必须大于或等于零
// --------------------!!!注意!!!--------------------
// Rate{PerSec(1), 0}表示1秒允许1个请求，Rate{PerSec(2), 0}表示0.5秒允许1个请求
// 所以，一般情况下MaxBurst必须大于0，预留足够的起始缓冲空间。
type RateQuota struct {
	MaxRate  Rate
	MaxBurst int
}

func PerSec(n int) Rate { return Rate{time.Second / time.Duration(n), n} }

func PerMin(n int) Rate { return Rate{time.Minute / time.Duration(n), n} }

func PerHour(n int) Rate { return Rate{time.Hour / time.Duration(n), n} }

func PerDay(n int) Rate { return Rate{24 * time.Hour / time.Duration(n), n} }

// cell-rate算法
type GCRARateLimiter struct {
	limit int

	// 可以理解为水桶size
	delayVariationTolerance time.Duration

	//令牌频率
	emissionInterval time.Duration

	store GCRAStore
}

func NewGCRARateLimiter(st GCRAStore, quota RateQuota) (*GCRARateLimiter, error) {
	if quota.MaxBurst < 0 {
		return nil, fmt.Errorf("Invalid RateQuota %#v. MaxBurst must be greater than zero.", quota)
	}
	if quota.MaxRate.period <= 0 {
		return nil, fmt.Errorf("Invalid RateQuota %#v. MaxRate must be greater than zero.", quota)
	}

	return &GCRARateLimiter{
		delayVariationTolerance: quota.MaxRate.period * (time.Duration(quota.MaxBurst) + 1),
		emissionInterval:        quota.MaxRate.period,
		limit:                   quota.MaxBurst + 1,
		store:                   st,
	}, nil
}

//RateLimit检查key是否已超过速率
func (g *GCRARateLimiter) RateLimit(key string, quantity int) (bool, RateLimitResult, error) {
	var tat, newTat, now time.Time
	var ttl time.Duration
	rlc := RateLimitResult{Limit: g.limit, RetryAfter: -1}
	limited := false

	i := 0
	for {
		var err error
		var tatVal int64
		var updated bool

		// tat 预计到达时间
		tatVal, now, err = g.store.GetWithTime(key)
		if err != nil {
			return false, rlc, err
		}

		if tatVal == -1 {
			tat = now
		} else {
			tat = time.Unix(0, tatVal)
		}

		increment := time.Duration(quantity) * g.emissionInterval
		if now.After(tat) {
			newTat = now.Add(increment)
		} else {
			newTat = tat.Add(increment)
		}

		// 如果下一个允许时间在将来，那么block
		allowAt := newTat.Add(-(g.delayVariationTolerance))
		if diff := now.Sub(allowAt); diff < 0 {
			if increment <= g.delayVariationTolerance {
				rlc.RetryAfter = -diff
			}
			ttl = tat.Sub(now)
			limited = true
			break
		}

		ttl = newTat.Sub(now)

		if tatVal == -1 {
			updated, err = g.store.SetIfNotExistsWithTTL(key, newTat.UnixNano(), ttl)
		} else {
			updated, err = g.store.CompareAndSwapWithTTL(key, tatVal, newTat.UnixNano(), ttl)
		}

		if err != nil {
			return false, rlc, err
		}
		if updated {
			break
		}

		i++
		if i > maxCASAttempts {
			return false, rlc, fmt.Errorf(
				"Failed to store updated rate limit data for key %s after %d attempts",
				key, i,
			)
		}
	}

	next := g.delayVariationTolerance - ttl
	if next > -g.emissionInterval {
		rlc.Remaining = int(next / g.emissionInterval)
	}
	rlc.ResetAfter = ttl

	return limited, rlc, nil
}
