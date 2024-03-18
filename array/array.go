package array

import "github.com/uninus-opensource/uninus-go-architect-common/uuid"

func ContainsInt(array []int, value int) bool {
	for _, val := range array {
		if val == value {
			return true
		}
	}
	return false
}

// ContainsInt32 return true if array contains value, false otherwise
func ContainsInt32(array []int32, value int32) bool {
	for _, val := range array {
		if val == value {
			return true
		}
	}
	return false
}

// ContainsString return true if array contains value, false otherwise
func ContainsString(array []string, value string) bool {
	for _, val := range array {
		if val == value {
			return true
		}
	}
	return false
}

func AppendString(slice []string, i string) []string {
	if len(slice) < 1 {
		return append(slice, i)
	}

	for _, v := range slice {
		if v == i {
			return slice
		}
	}
	return append(slice, i)
}

func AppendInteger32(slice []int32, i int32) []int32 {
	if len(slice) < 1 {
		return append(slice, i)
	}

	for _, v := range slice {
		if v == i {
			return slice
		}
	}
	return append(slice, i)
}

func AppendInteger64(slice []int64, i int64) []int64 {
	if len(slice) < 1 {
		return append(slice, i)
	}

	for _, v := range slice {
		if v == i {
			return slice
		}
	}
	return append(slice, i)
}

func AppendUUID(slice []uuid.UUID, i uuid.UUID) []uuid.UUID {
	if len(slice) < 1 {
		return append(slice, i)
	}

	for _, v := range slice {
		if v == i {
			return slice
		}
	}
	return append(slice, i)
}
