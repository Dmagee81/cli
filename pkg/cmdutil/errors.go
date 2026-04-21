package cmdutil

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2/terminal"
)

// FlagErrorf returns a new FlagError that wraps an error produced by
// fmt.Errorf(format, args...).
func FlagErrorf(format string, args ...interface{}) error {
	return FlagErrorWrap(fmt.Errorf(format, args...))
}

// FlagErrorWrap returns a new FlagError that wraps the specified error.
func FlagErrorWrap(err error) error { return &FlagError{err} }

// A *FlagError indicates an error processing command-line flags or other arguments.
// Such errors cause the application to display the usage message.
type FlagError struct {
	// Note: not struct{error}: only *FlagError should satisfy error.
	err error
}

func (fe *FlagError) Error() string {
	return fe.err.Error()
}

func (fe *FlagError) Unwrap() error {
	return fe.err
}

// silentError triggers exit code 1 without any error messaging.
type silentError struct{}

func (silentError) Error() string { return "SilentError" }

// cancelError signals user-initiated cancellation.
type cancelError struct{}

func (cancelError) Error() string { return "CancelError" }

// pendingError signals nothing failed but something is pending.
type pendingError struct{}

func (pendingError) Error() string { return "PendingError" }

// SilentError is an error that triggers exit code 1 without any error messaging.
// Backed by a distinct concrete type so it is distinguishable from other sentinels
// in telemetry (via %T) while remaining usable with errors.Is.
var SilentError error = silentError{}

// CancelError signals user-initiated cancellation. Backed by a distinct concrete
// type so it is distinguishable from other sentinels in telemetry (via %T) while
// remaining usable with errors.Is.
var CancelError error = cancelError{}

// PendingError signals nothing failed but something is pending. Backed by a distinct
// concrete type so it is distinguishable from other sentinels in telemetry (via %T)
// while remaining usable with errors.Is.
var PendingError error = pendingError{}

func IsUserCancellation(err error) bool {
	return errors.Is(err, CancelError) || errors.Is(err, terminal.InterruptErr)
}

func MutuallyExclusive(message string, conditions ...bool) error {
	numTrue := 0
	for _, ok := range conditions {
		if ok {
			numTrue++
		}
	}
	if numTrue > 1 {
		return FlagErrorf("%s", message)
	}
	return nil
}

type NoResultsError struct {
	message string
}

func (e NoResultsError) Error() string {
	return e.message
}

func NewNoResultsError(message string) NoResultsError {
	return NoResultsError{message: message}
}
