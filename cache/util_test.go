package cache

import "testing"

func TestSsWantSetNull(t *testing.T) {
	var dt interface{} = 99
	val := isWantSetNull(dt)
	t.Log(val)
}
