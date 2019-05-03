package wratelimiter

import (
	"fmt"
	"gopkg.in/redis.v5"
	"testing"
	"time"
)

func TestLimiterRequest(t *testing.T) {
	uniqueKey := "qrain|get|files"
	redisClient := redis.NewClient(&redis.Options{
		Password: "",
		Addr:     "localhost:6379",
	})

	limiter := NewLimiter(uniqueKey, 1, time.Second*10, redisClient)
	res, err := limiter.RequestTokens(1)

	if err != nil {
		fmt.Println("Error!!, ", err)
	} else {
		fmt.Println(res)
	}

	time.Now().UnixNano()
}
