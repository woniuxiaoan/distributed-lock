package wlock

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/redis.v5"
	"math/rand"
	"strconv"
	"time"
)

const (
	GlobalLockKey = "DISTRIBUTED_LOCK_KEY"
	UnlockScript  = `
local distributed_lock_key = KEYS[1]
local finger_print = KEYS[2]

if redis.call("get", distributed_lock_key) == finger_print then
  return redis.call("del", distributed_lock_key)
end

return 0
`
)

type DistributedLock struct {
	RedisClient    *redis.Client
	FingerPrint    string
	ExpireDuration time.Duration
	Rand           *rand.Rand
}

var (
	fingerPrint string
	scriptHash  string
)

func getFingerPrint() string {
	if len(fingerPrint) != 0 {
		return fingerPrint
	}
	fingerPrint = strconv.FormatUint(rand.New(rand.NewSource(time.Now().UnixNano())).Uint64(), 10)
	return fingerPrint
}

func NewDistributedLock(redisAddr string, expireDuration time.Duration) (*DistributedLock, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := redisClient.Ping().Err(); err != nil {
		logrus.Error("fail to init redisclient")
		return nil, err
	}

	go func() {
		ticker := time.Tick(time.Second * 10)
		for {
			<-ticker
			redisClient.ScriptLoad(UnlockScript)
		}
	}()

	return &DistributedLock{
		RedisClient:    redisClient,
		ExpireDuration: expireDuration,
		FingerPrint:    getFingerPrint(),
	}, nil
}

func (d *DistributedLock) Lock() bool {
	res, err := d.RedisClient.SetNX(GlobalLockKey, d.FingerPrint, d.ExpireDuration).Result()
	if err != nil {
		logrus.Error(err)
		return false
	}
	return res
}

func (d *DistributedLock) Unlock() bool {
	res, err := d.RedisClient.EvalSha(d.getScriptHash(), []string{GlobalLockKey, d.FingerPrint}).Result()
	if err != nil {
		logrus.Error(err)
		return false
	}

	return res.(int64) == 1
}

func (d *DistributedLock) getScriptHash() string {
	if len(scriptHash) != 0 {
		return scriptHash
	}

	scriptHash, _ = d.RedisClient.ScriptLoad(UnlockScript).Result()
	return scriptHash
}
