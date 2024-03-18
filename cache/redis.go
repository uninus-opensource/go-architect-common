package cache

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
	uer "github.com/uninus-opensource/uninus-go-architect-common/errors"
)

const (
	Key       = `cache-key`
	fileName  = `redis.go`
	pipeIsNil = `pipeline is nil`
)

type redisHashCache struct {
	client      *redis.Client
	closeClient bool
	prefix      string
}

// NewRedisHashCache returns new redis hash cache
func NewRedisHashCache(url, prefix string) HashCache {
	cli := redis.NewClient(&redis.Options{Addr: url})
	return &redisHashCache{client: cli,
		closeClient: true,
		prefix:      prefix}
}

// NewRedisSentinelHashCache returns new redis hash cache
func NewRedisSentinelHashCache(master, prefix string, sentinels []string) HashCache {
	cli := redis.NewFailoverClient(&redis.FailoverOptions{MasterName: master, SentinelAddrs: sentinels})
	return &redisHashCache{
		client:      cli,
		closeClient: true,
		prefix:      prefix}
}

// NewSharedHashCache return new shared redis hash cache
func NewSharedHashCache(cli *redis.Client, closeClient bool, prefix string) HashCache {
	return &redisHashCache{
		client:      cli,
		closeClient: closeClient,
		prefix:      prefix}
}

func (rhc *redisHashCache) Keys() []string {
	pref := fmt.Sprintf("%s*", rhc.prefix)
	keys := rhc.client.Keys(pref)
	return keys.Val()
}

func (rhc *redisHashCache) MSet(key string, values map[string]interface{}) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.HMSet(realKey, values).Err()
}

// BatchMSet, please add key in map[string]interface{} for key. if not, never add
// example : Prefix(coba), []map[string]interface{} (with data)
// values[0]  : [0]map[data]olgi-1
// values[0]  : [0]map[Key]key-for-insert (key can using constanta "Key" at redis.go)
// so this will insert to redis
// Prefix : coba:key-for-insert
// Data   : data with values olgi-1
func (rhc *redisHashCache) BatchMSet(req []map[string]interface{}) error {
	const (
		funcName = `BatchMSet`
	)

	pipe := rhc.client.Pipeline()
	defer pipe.Close()
	for _, v := range req {
		if val, ok := v[Key]; ok {
			realKey := fmt.Sprintf("%s:%v", rhc.prefix, val)
			delete(v, Key)
			err := pipe.HMSet(realKey, v).Err()
			if err != nil {
				return err
			}
		}
	}

	_, err := pipe.Exec()
	if err != nil {
		return err
	}

	return nil
}

func (rhc *redisHashCache) MGet(key string) (map[string]string, error) {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	result := rhc.client.HGetAll(realKey)
	return result.Result()
}

func (rhc *redisHashCache) BatchMGet(keys ...string) (resp []map[string]string, err error) {
	const (
		funcName = `BatchMGet`
	)
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, "%s")
	realKeys := []string{}
	for _, v := range keys {
		realKeys = append(realKeys, fmt.Sprintf(realKey, v))
	}

	pipe := rhc.client.Pipeline()
	defer pipe.Close()
	for _, v := range realKeys {
		err := pipe.HGetAll(v).Err()
		if err != nil {
			return resp, err
		}
	}

	cmds, err := pipe.Exec()
	if err != nil {
		return resp, err
	}

	for _, v := range cmds {
		cd, err := v.(*redis.StringStringMapCmd).Result()
		if err != nil {
			continue
		}
		if len(cd) < 1 {
			continue
		}
		resp = append(resp, cd)
	}
	return resp, nil
}

func (rhc *redisHashCache) Del(key string) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.Del(realKey).Err()
}

func (rhc *redisHashCache) Dels(keys ...string) error {
	const funcName = "Dels"
	realKeys := make([]string, len(keys))
	for _, k := range keys {
		realKey := fmt.Sprintf("%s:%s", rhc.prefix, k)
		realKeys = append(realKeys, realKey)
	}
	err := rhc.client.Del(realKeys...).Err()
	if err != nil {
		return err
	}
	return nil
}

func (rhc *redisHashCache) BatchMDel(keys ...string) error {
	const funcName = "BatchMDel"
	realKeys := make([]string, len(keys))
	for _, k := range keys {
		realKey := fmt.Sprintf("%s:%s", rhc.prefix, k)
		realKeys = append(realKeys, realKey)
	}
	pipe := rhc.client.Pipeline()
	defer pipe.Close()
	err := pipe.Del(realKeys...).Err()
	if err != nil {
		return err
	}
	return nil
}

func (rhc *redisHashCache) Set(key, field string, value string) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.HSet(realKey, field, value).Err()
}

func (rhc *redisHashCache) Get(key, field string) (string, error) {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.HGet(realKey, field).Result()
}

func (rhc *redisHashCache) Hincrby(key, field string, incre int64) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.HIncrBy(realKey, field, incre).Err()
}

func (rhc *redisHashCache) Expireat(key string, time time.Time) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.ExpireAt(realKey, time).Err()
}

func (rhc *redisHashCache) Expire(key string, time time.Duration) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.Expire(realKey, time).Err()
}

func (rhc *redisHashCache) MExists(key string, field string) (bool, error) {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	cek, err := rhc.client.HExists(realKey, field).Result()
	return cek, err
}

func (rhc *redisHashCache) SetNX(vehicleID string) (bool, error) {
	duration := time.Duration(60 * time.Second)
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, vehicleID)
	cek, err := rhc.client.SetNX(realKey, "1", duration).Result()
	if !cek && err == nil {
		return cek, errors.New(fmt.Sprintf("Key Already Exist : %s\n", realKey))
	}
	return cek, err
}

func (rhc *redisHashCache) MSetNX(keys string, duration time.Duration, value string) (bool, error) {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, keys)
	cek, err := rhc.client.SetNX(realKey, value, duration).Result()
	if !cek && err == nil {
		return cek, errors.New(fmt.Sprintf("Key Already Exist : %s\n", realKey))
	}
	return cek, err
}

func (rhc *redisHashCache) ClearSetNX(vehicleID string) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, vehicleID)
	return rhc.client.Del(realKey).Err()
}

func (rhc *redisHashCache) Close() error {
	if rhc.closeClient {
		if err := rhc.client.Close(); err != nil {
			log.Println(err.Error())
			return err
		}
	}
	return nil
}

// ScanKeys is scan all keys with count (default is 100).
// this will return list of keys and error
func (rhc *redisHashCache) ScanKeys() ([]string, error) {
	var cursor uint64
	var keys []string
	var err error

	realKey := fmt.Sprintf("%s:*", rhc.prefix)
	for {
		var ky []string
		ky, cursor, err = rhc.client.Scan(cursor, realKey, DefaultScanCount).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, ky...)

		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func (rhc *redisHashCache) LPush(key string, values interface{}) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.LPush(realKey, values).Err()
}

func (rhc *redisHashCache) RPush(key string, values interface{}) error {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.RPush(realKey, values).Err()
}

func (rhc *redisHashCache) LRange(key string, from, to int64) ([]string, error) {
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, key)
	return rhc.client.LRange(realKey, from, to).Result()
}

// GetPipeliner ...
func (rhc *redisHashCache) GetPipeline() redis.Pipeliner {
	return rhc.client.Pipeline()
}

// DelPipeline ...
func (rhc *redisHashCache) DelPipeline(pipe redis.Pipeliner, keys []string) error {
	const (
		funcName = `DelPipeline`
	)

	if pipe == nil {
		return uer.NewError(fileName, funcName, "", fmt.Errorf(pipeIsNil))
	}

	newKeys := []string{}
	realKey := fmt.Sprintf("%s:%s", rhc.prefix, "%s")
	for _, v := range keys {
		newKeys = append(newKeys, fmt.Sprintf(realKey, v))
	}
	err := pipe.Del(realKey).Err()
	if err != nil {
		return uer.NewError(fileName, funcName, "pipe.Del", err)
	}

	return nil
}

// DelPipelineWithCustomPrefixKey ...
func (rhc *redisHashCache) DelPipelineWithCustomPrefixKey(pipe redis.Pipeliner, keys []string) error {
	const (
		funcName = `DelPipelineWithCustomPrefixKey`
	)

	if pipe == nil {
		return uer.NewError(fileName, funcName, "", fmt.Errorf(pipeIsNil))
	}

	err := pipe.Del(keys...).Err()
	if err != nil {
		return uer.NewError(fileName, funcName, "pipe.HMSet", err)
	}

	return nil
}

// HsetPipeline ...
func (rhc *redisHashCache) HsetPipeline(pipe redis.Pipeliner, params KeyValues) error {
	const (
		funcName = `HsetPipeline`
	)

	if pipe == nil {
		return uer.NewError(fileName, funcName, "", fmt.Errorf(pipeIsNil))
	}

	realKey := fmt.Sprintf("%s:%s", rhc.prefix, "%s")
	for _, v := range params {
		err := rhc.client.HSet(fmt.Sprintf(realKey, v.Key), v.Field, v.Value).Err()
		if err != nil {
			return uer.NewError(fileName, funcName, "pipe.HSet", err)
		}
	}

	return nil
}

// HsetPipelineWithCustomPrefixKey ...
func (rhc *redisHashCache) HsetPipelineWithCustomPrefixKey(pipe redis.Pipeliner, customParamKeys KeyValues) error {
	const (
		funcName = `HsetPipelineWithCustomPrefixKey`
	)

	if pipe == nil {
		return uer.NewError(fileName, funcName, "", fmt.Errorf(pipeIsNil))
	}

	for _, v := range customParamKeys {
		err := pipe.HSet(v.Key, v.Field, v.Value).Err()
		if err != nil {
			return uer.NewError(fileName, funcName, "pipe.HMSet", err)
		}
	}

	return nil
}

// HmsetPipelineWithCustomPrefixKey ...
func (rhc *redisHashCache) HmsetPipelineWithCustomPrefixKey(pipe redis.Pipeliner, customParamKeys KeyMapValues) error {
	const (
		funcName = `HmsetPipelineWithCustomPrefixKey`
	)

	if pipe == nil {
		return uer.NewError(fileName, funcName, "", fmt.Errorf(pipeIsNil))
	}

	for _, v := range customParamKeys {
		err := pipe.HMSet(v.Key, v.Values).Err()
		if err != nil {
			return uer.NewError(fileName, funcName, "pipe.HMSet", err)
		}
	}

	return nil
}

// HmsetPipeline ...
func (rhc *redisHashCache) HmsetPipeline(pipe redis.Pipeliner, params KeyMapValues) error {
	const (
		funcName = `HmsetPipeline`
	)

	if pipe == nil {
		return uer.NewError(fileName, funcName, "", fmt.Errorf(pipeIsNil))
	}

	realKey := fmt.Sprintf("%s:%s", rhc.prefix, "%s")
	for _, v := range params {
		err := pipe.HMSet(fmt.Sprintf(realKey, v.Key), v.Values).Err()
		if err != nil {
			return uer.NewError(fileName, funcName, "pipe.HMSet", err)
		}
	}

	return nil
}

type redisPubSub struct {
	client *redis.Client
	prefix string
}

type RedisMessage struct {
	Pubsub  *redis.PubSub
	Message <-chan *redis.Message
}

// NewRedisPubSub is
func NewRedisPubSub(url, prefix string) PubSubRedis {
	return &redisPubSub{client: redis.NewClient(&redis.Options{Addr: url}),
		prefix: prefix}
}

// NewRedisSentinelPubSub returns new redis publish subcriber
func NewRedisSentinelPubSub(master, prefix string, sentinels []string) PubSubRedis {
	return &redisPubSub{
		client: redis.NewFailoverClient(&redis.FailoverOptions{MasterName: master, SentinelAddrs: sentinels}),
		prefix: prefix}
}

func (rps *redisPubSub) Publish(topic string, message string) error {
	realTopic := fmt.Sprintf("%s:%s", rps.prefix, topic)
	return rps.client.Publish(realTopic, message).Err()
}

func (rps *redisPubSub) SubscribePubSub(topic string) *redis.PubSub {
	realTopic := fmt.Sprintf("%s:%s", rps.prefix, topic)

	return rps.client.Subscribe(realTopic)
}

func (rps *redisPubSub) Channel(red *redis.PubSub) <-chan *redis.Message {
	return red.Channel()
}

func (rps *redisPubSub) ClosePubSub(red *redis.PubSub) error {
	return red.Close()
}

func (rps *redisPubSub) Close() error {
	return rps.client.Close()
}

// func (rm *RedisMessage) Channel() interface{} {
// 	chn := rm.Pubsub.Channel()
// 	return &RedisMessage{Message: chn}
// }

// func (rm *RedisMessage) Close() error {
// 	err := rm.Pubsub.Close()
// 	return err
// }
