package uuid

import (
	uuid "github.com/satori/go.uuid"
	bin "github.com/uninus-opensource/go-architect-common/math"
)


var (
	Empty = UUID{}
)

type UUID struct {
	MSB uint64
	LSB uint64
}

func New() (UUID, error) {
	id := uuid.NewV4()

	msb := bin.BytesToUint64(id[:8])
	lsb := bin.BytesToUint64(id[8:])

	return UUID{
		MSB: msb,
		LSB: lsb,
	}, nil
}

func FromInt(msb, lsb uint64) UUID {
	return UUID{MSB: msb, LSB: lsb}
}

func FromString(hex string) (UUID, error) {
	id, err := uuid.FromString(hex)
	if err != nil {
		return Empty, err
	}

	msb := bin.BytesToUint64(id[:8])
	lsb := bin.BytesToUint64(id[8:])

	return UUID{MSB: msb, LSB: lsb}, nil
}

func (id UUID) String() string {
	msb := bin.Uint64ToBytes(id.MSB)
	lsb := bin.Uint64ToBytes(id.LSB)
	uid, err := uuid.FromBytes(append(msb, lsb...))
	if err != nil {
		return ""
	}

	return uid.String()
}


// IsEmpty memeriksa apakah UUID tidak memiliki nilai atau memiliki nilai
// return nilai true jika tidak memiliki nilai atau NULL
// return nilai false jika memiliki
func (id UUID) IsEmpty() bool {
	if id == Empty || id.MSB == 0 || id.LSB == 0 {
		return true
	}
	return false
}

func IsSetUUID(val UUID, defvals ...UUID) UUID {
	if val.IsEmpty() {
		for _, v := range defvals {
			if !v.IsEmpty() {
				return v
			}
		}
	}
	return val
}

