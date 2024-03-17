package cache

import (
	"fmt"
	"time"

	"github.com/uninus-opensource/go-architect-common/uuid"
)

func AddToRedisMapIsExist(rdsDatamap map[string]interface{}, value interface{}, redisFieldConst []string) map[string]interface{} {
	if value == nil {
		return rdsDatamap
	}
	if !isStringsMoreNumber(redisFieldConst, 1) {
		return rdsDatamap
	}
	switch value.(type) {
	case bool:
		if value.(bool) {
			rdsDatamap[redisFieldConst[0]] = value
		}
	case uuid.UUID:
		if !value.(uuid.UUID).IsEmpty() {
			if isStringsMoreNumber(redisFieldConst, 2) {
				rdsDatamap[redisFieldConst[0]] = value.(uuid.UUID).MSB
				rdsDatamap[redisFieldConst[1]] = value.(uuid.UUID).LSB
			}
		}
	case int32:
		if value.(int32) != 0 {
			rdsDatamap[redisFieldConst[0]] = value
		}
	case float64:
		if value.(float64) != 0 {
			rdsDatamap[redisFieldConst[0]] = value
		}
	case int64:
		if value.(int64) != 0 {
			rdsDatamap[redisFieldConst[0]] = value
		}
	case time.Time:
		if value.(time.Time).Unix() > 0 {
			rdsDatamap[redisFieldConst[0]] = value.(time.Time).Unix()
		}
	case string:
		if value.(string) != "" {
			rdsDatamap[redisFieldConst[0]] = value
		}
	default:
		fmt.Printf("error : Failed store to redis : value[%v] , field[%v]\n", value, redisFieldConst)
	}
	return rdsDatamap
}

func isWantSetNull(val interface{}) interface{} {
	if fmt.Sprint(val) == "999" {
		return 0
	}
	return val
}

// IsStringsMoreNumber .. // TODO
func isStringsMoreNumber(arrs []string, num int) bool {
	if len(arrs) >= num {
		return true
	}
	return false
}

// GenerateToStrings ... //TODO
func GenerateToStrings(vals ...string) []string {
	return vals
}
