package utils

import (
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// ErrInvalidName indicates the name did not pass funciton name validation.
type ErrInvalidFunctionName error

// ErrInvalidEnvVarName indicates the name did not pass env var name validation.
type ErrInvalidEnvVarName error

// ValidateFunctionName validatest that the input name is a valid function name, ie. valid DNS-1123 label.
// It must consist of lower case alphanumeric characters or '-' and start and end with an alphanumeric character
// (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')
func ValidateFunctionName(name string) error {

	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		// In case of invalid name the error is this:
		//	"a DNS-1123 label must consist of lower case alphanumeric characters or '-',
		//   and must start and end with an alphanumeric character (e.g. 'my-name',
		//   or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')"
		// Let's reuse it for our purposes, ie. replace "DNS-1123 label" substring with "function name"
		return ErrInvalidFunctionName(errors.New(strings.Replace(strings.Join(errs, ""), "a DNS-1123 label", "Function name", 1)))
	}

	return nil
}

// ValidateEnvVarName validatest that the input name is a valid Kubernetes Environmet Variable name.
// It must  must consist of alphabetic characters, digits, '_', '-', or '.', and must not start with a digit
// (e.g. 'my.env-name',  or 'MY_ENV.NAME',  or 'MyEnvName1', regex used for validation is '[-._a-zA-Z][-._a-zA-Z0-9]*'))
func ValidateEnvVarName(name string) error {
	if errs := validation.IsEnvVarName(name); len(errs) > 0 {
		return ErrInvalidEnvVarName(errors.New(strings.Join(errs, "")))
	}

	return nil
}
