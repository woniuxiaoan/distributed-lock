package wratelimiter

import (
	"errors"
	"gopkg.in/redis.v5"
	"time"
)

const Script = `
local token_num_key = KEYS[1]
local token_num_last_updated_time_key = KEYS[2]
local token_required_time = tonumber(ARGV[1])
local token_num_capacity = tonumber(ARGV[2])
local duration = tonumber(ARGV[3])
local token_num_required = tonumber(ARGV[4])
local token_generated_rate = tonumber(ARGV[5])
local token_num_last_updated_time = tonumber(redis.call("get", token_num_last_updated_time_key))

if token_num_last_updated_time == nil then
  token_num_last_updated_time = token_required_time
end

local new_token = math.min(token_num_capacity, (token_required_time-token_num_last_updated_time) * token_generated_rate)

local token_num = tonumber(redis.call("get", token_num_key))
if token_num == nil then
  token_num = 0
end

token_num = math.min(token_num_capacity, token_num+new_token)

local allowed = token_num_required <= token_num

if allowed then
  token_num = token_num - token_num_required
end

token_num_last_updated_time = token_required_time

local ttl = math.floor(2*duration/1000000000)

redis.call("setex", token_num_key, ttl, token_num)
redis.call("setex", token_num_last_updated_time_key, ttl, token_num_last_updated_time)

if allowed then
  return token_num
end

return -1
`

var hashScript string

type Limiter struct {
	uniqueKey      string
	max            int64
	duration       time.Duration
	redisClient    *redis.Client
	rate           float32
	reloadDuration time.Duration
}

type LimiterSetting struct {
	UniqueKey            string
	Max                  int64
	Duration             time.Duration
	ScriptReloadDuration time.Duration
	RedisAddr            string
	RedisPwd             string
}

func NewLimiter(setting *LimiterSetting) *Limiter {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     setting.RedisAddr,
		Password: setting.RedisPwd,
	})

	go func() {
		ticker := time.Tick(setting.ScriptReloadDuration)
		for {
			<-ticker
			redisClient.ScriptLoad(Script)
		}
	}()

	return &Limiter{
		uniqueKey:   setting.UniqueKey,
		max:         setting.Max,
		duration:    setting.Duration,
		redisClient: redisClient,
		rate:        (float32(setting.Max) * float32(time.Millisecond)) / float32(setting.Duration),
	}
}

func (l *Limiter) GenerateKeys() []string {
	return []string{
		l.uniqueKey + "|" + "token_num",
		l.uniqueKey + "|" + "token_num_last_updated_key",
	}
}

func (l *Limiter) RequestTokens(tokeNums int64) (int64, error) {
	if len(hashScript) == 0 {
		res, err := l.redisClient.ScriptLoad(Script).Result()
		if err != nil {
			return -1, err
		}
		hashScript = res
	}

	requiredTime := time.Now().UnixNano() / int64(time.Millisecond)
	res, err := l.redisClient.EvalSha(hashScript, l.GenerateKeys(), requiredTime, l.max, int64(l.duration), tokeNums, l.rate).Result()
	if err != nil {
		return -1, err
	}

	if res.(int64) == -1 {
		return -1, errors.New("令牌不够")
	}

	return res.(int64), nil
}
