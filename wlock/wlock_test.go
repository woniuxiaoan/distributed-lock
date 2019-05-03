package wlock

import (
	"log"
	"testing"
	"time"
)

func TestDistributedLock(t *testing.T) {
	lock, err := NewDistributedLock("localhost:6379", time.Second*10)
	defer lock.RedisClient.Close()

	if err != nil {
		log.Println(err)
		return
	}

	for i := 0; i <= 100; i++ {
		go func(lock *DistributedLock, i int) {
			ticker := time.Tick(time.Second * 1)
			for {
				<-ticker
				if lock.Lock() {
					log.Printf("Goroutine %d get lock", i)
					lock.Unlock()
					return
				}
			}
		}(lock, i)
	}

	<-time.Tick(time.Hour * 1)
}
