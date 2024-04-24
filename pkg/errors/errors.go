package errors

import (
	"errors"
	"fmt"
)

var (
	Is = errors.Is
	As = errors.As
)

const cannotPrefix = "cannot"

func Join(errs []error) error {
	return errors.Join(errs...)
}

func Error(msg string, args ...any) error {
	return fmt.Errorf(msg, args...)
}

func Fail(whatFailed string, args ...any) error {
	return fmt.Errorf(cannotPrefix+whatFailed, args...)
}

func Wrap(err error, wrapper string, args ...any) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", fmt.Sprintf(wrapper, args...), err)
}

func WrapFail(err error, whatFailed string, args ...any) error {
	if err == nil {
		return nil
	}
	wrapper := fmt.Sprintf(whatFailed, args...)
	return Wrap(err, "%s %s", cannotPrefix, wrapper)
}
