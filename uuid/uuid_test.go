package uuid

import "testing"

func TestNew(t *testing.T) {
	// t.Fail()

	uuid := FromInt(4816090621160607225, 11900304877229632984)
	t.Log(uuid)

	uuid, _ = FromString("67847bc5-daf8-4265-b079-245f5286c3af")
	t.Log(uuid, uuid.MSB, uuid.LSB)

	uuid, _ = New()
	t.Log(uuid, uuid.MSB, uuid.LSB)
}
