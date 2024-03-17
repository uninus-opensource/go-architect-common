package math

import (
	"strconv"
	"time"
)

func ConvertStringToInt32(rdsData string) int32 {
	return int32(ConvertStringToInt64(rdsData))
}

func ConvertStringToTime(rdsData string) time.Time {
	return time.Unix(ConvertStringToInt64(rdsData), 0)
}

func ConvertStringToInt64(rdsData string) int64 {
	dt, _ := strconv.ParseInt(rdsData, 10, 64)
	return dt
}

func ConvertStringToUint64(rdsData string) uint64 {
	dt, _ := strconv.ParseUint(rdsData, 10, 64)
	return dt
}

func ConvertStringToFloat64(rdsData string) float64 {
	dt, _ := strconv.ParseFloat(rdsData, 64)
	return dt
}

func ConvertStringToArrByte(rdsData string) []byte {
	return []byte(rdsData)
}

func ConvertStringToBoolRedis(rdsData string) bool {
	dt, _ := strconv.ParseBool(rdsData)
	return dt
}
