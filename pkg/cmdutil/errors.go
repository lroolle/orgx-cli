package cmdutil

import (
	"errors"
	"fmt"
)

var SilentError = errors.New("SilentError")
var CancelError = errors.New("CancelError")

type FlagError struct {
	Err error
}

func (e *FlagError) Error() string {
	return e.Err.Error()
}

func (e *FlagError) Unwrap() error {
	return e.Err
}

func FlagErrorf(format string, args ...interface{}) error {
	return &FlagError{Err: fmt.Errorf(format, args...)}
}

func FlagErrorWrap(err error) error {
	return &FlagError{Err: err}
}

func IsFlagError(err error) bool {
	var fe *FlagError
	return errors.As(err, &fe)
}
