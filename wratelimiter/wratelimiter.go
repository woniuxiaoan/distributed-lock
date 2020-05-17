package wratelimiter

import (
	"errors"
	"fmt"
	"gopkg.in/redis.v5"
	"math/rand"
	"strconv"
	"time"
)

const Script = `
local function trim(origin, tag)
   local result = origin
   if string.len(tag) ~= 0 then
       local begin, _ = string.find(origin, tag)
       if begin ~= nil then
         result = string.sub(origin, 1, begin - 1)
       end
   end
   return result
end

local hash_tag = ARGV[6]
local token_num_key = KEYS[1]
local token_num_last_updated_time_key = KEYS[2]
local token_required_time = tonumber(trim(ARGV[1],hash_tag))
local token_num_capacity = tonumber(trim(ARGV[2],hash_tag))
local ttl = tonumber(trim(ARGV[3],hash_tag))
local token_num_required = tonumber(trim(ARGV[4],hash_tag))
local token_generated_rate = tonumber(trim(ARGV[5],hash_tag))
local token_num_last_updated_time = tonumber(redis.call("get", token_num_last_updated_time_key))

if token_num_last_updated_time == nil then
  token_num_last_updated_time = token_required_time
end

local new_token = math.min(token_num_capacity, (token_required_time-token_num_last_updated_time) / token_generated_rate)

local token_num = tonumber(redis.call("get", token_num_key))
if token_num == nil then
  token_num = token_num_capacity
end

token_num = math.min(token_num_capacity, token_num+new_token)

local allowed = token_num >= 1

if allowed then
  token_num = token_num - token_num_required
end

token_num_last_updated_time = token_required_time

redis.call("setex", token_num_key, ttl, token_num)
redis.call("setex", token_num_last_updated_time_key, ttl, token_num_last_updated_time)

if allowed then
  return {token_num, 1}
end

return {token_num, 0}
`

var hashScript string

type limiter struct {
	uniqueKey      string
	max            int64
	redisClient    *redis.Client
	rate           int64
	reloadDurationSeconds int
	refreshDurationSeconds int
	hashTag string
	rand *rand.Rand
}

type RateLimiter interface {
	RequestTokens(tokenNumsRequired int64) (int64, error)
}

type LimiterSetting struct {
	UniqueKey            string
	Max                  int64
	DurationMicrosecondsPerToken     int64
	ScriptReloadSeconds int
	RefreshDurationSeconds int
	RedisAddr            string
	RedisPwd             string
	RedisClusterMod bool
}

func NewLimiter(setting *LimiterSetting) RateLimiter {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     setting.RedisAddr,
		Password: setting.RedisPwd,
	})

	go func() {
		ticker := time.Tick(time.Second * time.Duration(setting.ScriptReloadSeconds))
		for {
			<-ticker
			redisClient.ScriptLoad(Script)
		}
	}()

	limiter := &limiter{
		uniqueKey:   setting.UniqueKey,
		max:         setting.Max,
		redisClient: redisClient,
		rate:        setting.DurationMicrosecondsPerToken,
		refreshDurationSeconds: setting.RefreshDurationSeconds,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if setting.RedisClusterMod {
		limiter.hashTag = fmt.Sprintf("|{%s}", limiter.generateFingerPrint())
	}

	return limiter
}

func (l *limiter) generateFingerPrint() string {
	return strconv.FormatUint(l.rand.Uint64(), 10)
}

func (l *limiter) generateKeys() []string {
	keys := []string{
		l.uniqueKey + "|" + "token_num" ,
		l.uniqueKey + "|" + "token_num_last_updated_key",
	}

	if len(l.hashTag) != 0 {
		for i := 0; i<len(keys); i++ {
			keys[i] = fmt.Sprintf("%v%s", keys[i], l.hashTag)
		}
	}

	return keys
}

func (l *limiter) RequestTokens(tokeNumsRequired int64) (int64, error) {
	if tokeNumsRequired > l.max {
		return -1, fmt.Errorf("允许请求最大值为%d", l.max)
	}

	if len(hashScript) == 0 {
		res, err := l.redisClient.ScriptLoad(Script).Result()
		if err != nil {
			return -1, err
		}
		hashScript = res
	}

	requiredTime := time.Now().UnixNano() / int64(time.Microsecond)
	interfaces, err := l.redisClient.EvalSha(hashScript, l.generateKeys(), l.generateArgs(requiredTime, tokeNumsRequired)...).Result()
	if err != nil {
		return -1, err
	}

	res, ok := interfaces.([]interface{})
	if !ok {
		return -1, errors.New("transfer interfaces failed")
	}

	tokensLeft, pass := res[0].(int64), res[1].(int64) == 1
	if !pass {
		return tokensLeft, errors.New("令牌不足")
	}

	return tokensLeft, nil
}

func (l *limiter) generateArgs(requiredTime int64, tokenNumsRequired int64) []interface{} {
	args := []interface{}{
		requiredTime,
		l.max,
		l.refreshDurationSeconds,
		tokenNumsRequired,
		l.rate,
		l.hashTag,
	}

	if len(l.hashTag) != 0 {
		for i := 0; i < len(args) - 1; i++ {
			args[i] = fmt.Sprintf("%v%s", args[i], l.hashTag)
		}
	}

	return args
}
