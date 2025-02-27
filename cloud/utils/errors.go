package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/outscale/osc-sdk-go/v2"
)

type OAPIError struct {
	errors []osc.Errors
}

func (err OAPIError) Error() string {
	if len(err.errors) == 0 {
		return "unknown error"
	}
	oe := err.errors[0]
	str := oe.GetCode() + "/" + oe.GetType()
	details := oe.GetDetails()
	if details != "" {
		str += " (" + details + ")"
	}
	return str
}

func ExtractOAPIError(err error, httpRes *http.Response) error {
	var genericError osc.GenericOpenAPIError
	if errors.As(err, &genericError) {
		errorsResponse, ok := genericError.Model().(osc.ErrorResponse)
		if ok && len(*errorsResponse.Errors) > 0 {
			return OAPIError{errors: *errorsResponse.Errors}
		}
	}
	if httpRes != nil {
		return fmt.Errorf("http error %w", err)
	}
	return err
}
