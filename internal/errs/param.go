package errs

import "fmt"

var errorInvalidParamFmt = "invalid request params: %s %v"
var errorRecordNotFoundFmt = "%s not found by %s"
var errorMissingParamFmt = "missing required param: %s"

func NewInvalidParamErr(name string, value interface{}) error {
	return fmt.Errorf(errorInvalidParamFmt, name, value)
}

func NewRecordNotFoundErr(name string, value interface{}) error {
	return fmt.Errorf(errorRecordNotFoundFmt, name, value)
}

func NewMissingParamError(name string) error {
	return fmt.Errorf(errorMissingParamFmt, name)
}
