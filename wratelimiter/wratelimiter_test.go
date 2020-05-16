package wratelimiter

import (
	"testing"
)

func TestLimiterRequest(t *testing.T) {
	uniqueKey := "qrain|get|files"
	setting := &LimiterSetting{
		UniqueKey:           uniqueKey,
		Max:                  10,
		RedisAddr:            "localhost:6379",
		RedisPwd:             "",
		DurationMicrosecondsPerToken: 1000000,
		RefreshDurationSeconds: 60,
		ScriptReloadSeconds: 10,
		RedisClusterMod: true,
	}
	limiter := NewLimiter(setting)

	if tokens, err := limiter.RequestTokens(6); err != nil {
		t.Logf("request token failed: %v\n", err)
	} else {
		t.Logf("succeed,剩余%v个令牌", tokens)
	}
}
