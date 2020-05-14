package wratelimiter

import (
	"testing"
	"time"
)

func TestLimiterRequest(t *testing.T) {
	uniqueKey := "qrain|get|files"
	setting := &LimiterSetting{
		UniqueKey:           uniqueKey,
		Max:                  5,
		Duration:             time.Second * 10,
		ScriptReloadDuration: time.Second * 5,
		RedisAddr:            "localhost:6379",
		RedisPwd:             "",
	}
	limiter := NewLimiter(setting)

	if tokens, err := limiter.RequestTokens(1); err != nil {
		t.Logf("request token failed: %v\n", err)
	} else {
		t.Logf("succeed,剩余%v个令牌", tokens)
	}
}
