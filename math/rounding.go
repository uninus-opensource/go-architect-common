package math

import "math"

func RoundUp10Int(toRound int) int {
	if toRound%10 == 0 {
		return toRound
	}
	return (10 - toRound%10) + toRound
}

func RoundDown10Int(toRound int) int {
	return toRound - toRound%10
}

func RoundUp10Int32(toRound int32) int32 {
	if toRound%10 == 0 {
		return toRound
	}
	return (10 - toRound%10) + toRound
}

func RoundDown10Int32(toRound int32) int32 {
	return toRound - toRound%10
}

func RoundUp10Int64(toRound int64) int64 {
	if toRound%10 == 0 {
		return toRound
	}
	return (10 - toRound%10) + toRound
}

func RoundDown10Int64(toRound int64) int64 {
	return toRound - toRound%10
}

func RoundUpFloat64ToInt32(toRound float64) int32 {
	if math.Mod(toRound, 1.0) == 0 {
		return int32(toRound)
	}
	return int32(toRound) + 1
}

func RoundUp1000Int32(toRound int32) int32 {
	if toRound%1000 == 0 {
		return toRound
	}
	return (1000 - toRound%1000) + toRound
}

func RoundDown1000Int32(toRound int32) int32 {
	return toRound - toRound%1000
}
