package errors

import (
	"errors"
	"strings"
)

// DBError list
var DBError = map[int]string{
	// Duplicate Entry
	// 1062: "Error 1062",
	// Server shutdown
	1053: "Error 1053",
	// Err Disk Full
	1021: "Error 1021",
	// Err On Write
	1026: "Error 1026",
	// Cant find record
	1032: "Error 1032",
	// Out of Memory
	1037: "Error 1037",
	// Too many conns
	1040: "Error 1040",
	// Out of resource
	1041: "Error 1041",
	// Bad host error
	1042: "Error 1042",
	// Handshake Error
	1043: "Error 1043",
	// No DB Error
	1046: "Error 1046",
	// Err Unknown
	1047: "Error 1047",
	// Cant Change lock
	1150: "Error 1150",
	// Too Many Delayed Threads
	1151: "Error 1151",
	// Err Read From Pipe
	1154: "Error 1154",

	// ER_NET_FCNTL_ERROR
	1155: "Error 1155",
	// ER_NET_PACKETS_OUT_OF_ORDER
	1156: "Error 1156",
	// ER_NET_UNCOMPRESS_ERROR
	1157: "Error 1157",
	// ER_NET_READ_ERROR
	1158: "Error 1158",
	// ER_NET_READ_INTERRUPTED
	1159: "Error 1159",
	// ER_NET_ERROR_ON_WRITE
	1160: "Error 1160",
	// ER_NET_WRITE_INTERRUPTED
	1161: "Error 1161",

	// ER_SLAVE_THREAD
	1202: "Error 1202",
	// ER_TOO_MANY_USER_CONNECTIONS
	1203: "Error 1203",

	// ER_SLAVE_WAS_RUNNING
	1254: "Error 1254",
	// ER_SLAVE_WAS_RUNNING
	1255: "Error 1255",
	// ER_QUERY_INTERRUPTED
	1317: "Error 1317",
}

// NewError wraping error
func NewError(fileName, funcName, executMsg string, err error) error {
	if err != nil {
		return errors.New(fileName + ":" + funcName + ":" + executMsg + " => " + err.Error())
	}
	return nil
}

// IsDBError is a error checker
func IsDBError(err error) bool {
	for _, value := range DBError {
		if strings.Contains(err.Error(), value) {
			return true
		}
	}
	return false
}
