package cache

import (
	"io"
	"time"

	"github.com/go-redis/redis"
)

const (
	// DefaultScanCount is default count when using scan in redis
	DefaultScanCount = 100
)

//HashCache is hash map cache
type HashCache interface {
	io.Closer
	Keys() []string
	MSet(string, map[string]interface{}) error
	// BatchMSet, please add key in map[string]interface{} for key. if not, never add
	// example : Prefix(coba), []map[string]interface{} (with data)
	// values[0]  : [0]map[data]olgi-1
	// values[0]  : [0]map[Key]key-for-insert (key can using constanta "Key" at redis.go)
	// so this will insert to redis
	// Prefix : coba:key-for-insert
	// Data   : data with values olgi-1
	BatchMSet([]map[string]interface{}) error
	MGet(string) (map[string]string, error)
	BatchMGet(...string) ([]map[string]string, error)
	Del(string) error
	BatchMDel(...string) error
	Dels(...string) error
	Set(string, string, string) error
	Get(string, string) (string, error)
	Hincrby(string, string, int64) error
	Expireat(string, time.Time) error
	Expire(string, time.Duration) error
	MExists(string, string) (bool, error)
	SetNX(string) (bool, error)
	MSetNX(string, time.Duration, string) (bool, error)
	ClearSetNX(string) error
	// ScanKeys is scan all keys with count (default is 100).
	// this will return list of keys and error
	ScanKeys() ([]string, error)
	// LPush is push to redis with key and values
	// the data at redis will ASC. if data, FRIST insert will be on top.
	// ex : data = [1,2,3,4]
	// if that looping, the first data is 1.
	// so, at redis will like this.
	// Redis : 1
	//         2
	//         3
	//         4
	LPush(key string, values interface{}) error
	// RPush is push to redis with key and values
	// the data at redis will DESC. if data, LAST insert will be on top.
	// ex : data = [1,2,3,4]
	// if that looping, the first data is 1.
	// so, at redis will like this.
	// Redis : 4
	//         3
	//         2
	//         1
	RPush(key string, values interface{}) error
	LRange(key string, from, to int64) ([]string, error)

	// getter & setter pipeline
	GetPipeline() redis.Pipeliner

	// customParam
	DelPipeline(pipe redis.Pipeliner, keys []string) error
	DelPipelineWithCustomPrefixKey(pipe redis.Pipeliner, keys []string) error
	HsetPipelineWithCustomPrefixKey(pipe redis.Pipeliner, customParamKeys KeyValues) error
	HsetPipeline(pipe redis.Pipeliner, params KeyValues) error
	HmsetPipelineWithCustomPrefixKey(pipe redis.Pipeliner, customParamKeys KeyMapValues) error
	HmsetPipeline(pipe redis.Pipeliner, params KeyMapValues) error
}

//PubSubRedis is
type PubSubRedis interface {
	io.Closer
	Publish(string, string) error
	SubscribePubSub(string) *redis.PubSub
	Channel(*redis.PubSub) <-chan *redis.Message
	ClosePubSub(*redis.PubSub) error
}
