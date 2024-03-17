package math

import (
	"time"
)

func IsSetString(val string, defvals ...string) string {
	if val == "" {
		for _, v := range defvals {
			if v != "" {
				return v
			}
		}
	}
	return val
}

func IsSetInt32(val int32, defvals ...int32) int32 {
	if val == 0 {
		for _, v := range defvals {
			if v != 0 {
				return v
			}
		}
	}
	return val
}

func IsSetInt(val int, defvals ...int) int {
	if val == 0 {
		for _, v := range defvals {
			if v != 0 {
				return v
			}
		}
	}
	return val
}

func IsSetInt64(val int64, defvals ...int64) int64 {
	if val == 0 {
		for _, v := range defvals {
			if v != 0 {
				return v
			}
		}
	}
	return val
}

func IsSetFloat64(val float64, defvals ...float64) float64 {
	if val == 0 {
		for _, v := range defvals {
			if v != 0 {
				return v
			}
		}
	}
	return val
}

func IsSetFloat32(val float32, defvals ...float32) float32 {
	if val == 0 {
		for _, v := range defvals {
			if v != 0 {
				return v
			}
		}
	}
	return val
}

func IsSetTime(val time.Time, defvals ...time.Time) time.Time {
	var NullTime time.Time
	if val == NullTime {
		for _, v := range defvals {
			if v != NullTime {
				return v
			}
		}
	}
	return val
}
