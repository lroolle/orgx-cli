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

// FixError pairs a failure with the command that fixes it, so the
// error envelope can carry a machine-actionable "fix" instead of
// prose an agent has to mine.
type FixError struct {
	Err error
	Fix string // a runnable command or concrete next step
}

func (e *FixError) Error() string {
	return e.Err.Error() + " — " + e.Fix
}

func (e *FixError) Unwrap() error {
	return e.Err
}

// WithFix attaches a fix to an error; nil stays nil.
func WithFix(err error, fix string) error {
	if err == nil {
		return nil
	}
	return &FixError{Err: err, Fix: fix}
}

// ErrorEnvelope is the JSON error shape (orgx.error.v1), written to
// stderr when --json was requested: stdout stays data-only.
type ErrorEnvelope struct {
	Kind  string `json:"kind"`
	Error struct {
		Message string `json:"message"`
		Fix     string `json:"fix,omitempty"`
	} `json:"error"`
}

func NewErrorEnvelope(err error) ErrorEnvelope {
	env := ErrorEnvelope{Kind: "orgx.error.v1"}
	var fe *FixError
	if errors.As(err, &fe) {
		env.Error.Message = fe.Err.Error()
		env.Error.Fix = fe.Fix
		return env
	}
	env.Error.Message = err.Error()
	return env
}
