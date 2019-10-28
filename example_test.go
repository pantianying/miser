package miser_test

import (
	"fmt"
	"github.com/pantianying/miser"
	"github.com/pantianying/miser/store/memstore"
	"log"
)

//example：如何使用rateLimiter
func ExampleRateLimiter() {
	store, err := memstore.New(65536)
	if err != nil {
		log.Fatal(err)
	}

	// Maximum burst of 5 which refills at 1 token per hour.
	quota := miser.RateQuota{MaxRate: miser.PerHour(1), MaxBurst: 5}

	rateLimiter, err := miser.NewGCRARateLimiter(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 20; i++ {
		bucket := fmt.Sprintf("by-order:%v", i/10)

		limited, result, err := rateLimiter.RateLimit(bucket, 1)
		if err != nil {
			log.Fatal(err)
		}

		if limited {
			fmt.Printf("Iteration %2v; bucket %v: FAILED. Rate limit exceeded.\n",
				i, bucket)
		} else {
			fmt.Printf("Iteration %2v; bucket %v: Operation successful (remaining=%v).\n",
				i, bucket, result.Remaining)
		}
	}
}
