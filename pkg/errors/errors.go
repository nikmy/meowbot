package errors

import (
	"errors"
	"fmt"
)

var (
	Is = errors.Is
	As = errors.As
)

const cantPrefix = "can't"

func Collapse(errs []error) error {
	return errors.Join(errs...)
}

func Error(msg string) error {
	return fmt.Errorf(msg)
}

func Errorf(msgFormat string, args ...any) error {
	return fmt.Errorf(msgFormat, args...)
}

func Fail(whatFailed string) error {
	return fmt.Errorf("%s %s", cantPrefix, whatFailed)
}

func Failf(whatFailedFormat string, args ...any) error {
	return fmt.Errorf(cantPrefix+whatFailedFormat, args...)
}

func Wrap(err error, wrapper string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", wrapper, err)
}

func Wrapf(err error, wrapperFormat string, args ...any) error {
	if err == nil {
		return nil
	}
	wrapper := fmt.Sprintf(wrapperFormat, args...)
	return Wrap(err, wrapper)
}

func WrapFail(err error, whatFailed string) error {
	if err == nil {
		return nil
	}
	return Wrapf(err, "%s %s", cantPrefix, whatFailed)
}

func WrapFailf(err error, whatFailedFormat string, args ...any) error {
	if err == nil {
		return nil
	}
	return Wrapf(err, cantPrefix+whatFailedFormat, args...)
}
