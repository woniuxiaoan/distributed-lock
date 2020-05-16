package wratelimiter

import (
	"testing"
	"time"
)

func TestLimiterRequest(t *testing.T) {
	uniqueKey := "qrain|get|files"
	setting := &LimiterSetting{
		UniqueKey:           uniqueKey,
		Max:                  50,
		RedisAddr:            "localhost:6379",
		RedisPwd:             "",
		DurationMicrosecondsPerToken: 10000,
		RefreshDurationSeconds: 60,
		ScriptReloadSeconds: 10,
		RedisClusterMod: true,
	}
	limiter := NewLimiter(setting)

	for i := 0; i<50; i++ {
		if tokens, err := limiter.RequestTokens(5); err != nil {
			t.Logf("第%d次, request token failed: %v\n", i+1, err)
		} else {
			t.Logf("第%d次, succeed,剩余%v个令牌", i+1, tokens)
		}
		time.Sleep(time.Millisecond * 10)
	}
}
