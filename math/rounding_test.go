package math

import (
	"testing"
)

func TestRound10Int64(t *testing.T) {
	// t.Fail()

	var toRounding int64

	toRounding = 432

	t.Log("Round Up int64 : ", RoundUp10Int64(toRounding))

	t.Log("Round Down int64  : ", RoundDown10Int64(toRounding))
}

func TestRoundUpFloat64(t *testing.T) {
	// t.Fail()

	var toRounding float64

	toRounding = 425.00

	t.Log("Round Up Float64 : ", RoundUpFloat64ToInt32(toRounding))
}

func TestRound1000Int32(t *testing.T) {
	//t.Fail()

	var toRounding int32

	toRounding = 500

	t.Log("Round Up int32 : ", RoundUp1000Int32(toRounding))

	t.Log("Round Down int32  : ", RoundDown1000Int32(toRounding))
}
