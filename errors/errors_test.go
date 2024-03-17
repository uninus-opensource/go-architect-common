package errors

import (
	ers "errors"
	"testing"
)

func TestNewError(t *testing.T) {
	//t.Fail()
	var err error = ers.New("coba error")
	fileName := "error.go"
	funcName := "TestNewError"
	t.Log(NewError(fileName, funcName, "log", err))
}
