/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package v1beta1

import (
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func AppendValidation(erl field.ErrorList, errs ...*field.Error) field.ErrorList {
	for _, err := range errs {
		if err != nil {
			erl = append(erl, err)
		}
	}
	return erl
}

func MergeValidation(errs ...*field.Error) field.ErrorList {
	erl := make(field.ErrorList, 0, len(errs))
	return AppendValidation(erl, errs...)
}

func ValidateEmpty(p *field.Path, value, condition string) *field.Error {
	if value != "" {
		return field.Forbidden(p, condition)
	}
	return nil
}

func ValidateEmptySlice[E any](p *field.Path, value []E, condition string) *field.Error {
	if len(value) > 0 {
		return field.Forbidden(p, condition)
	}
	return nil
}

func ValidateRequired(p *field.Path, value, condition string) *field.Error {
	if value == "" {
		return field.Required(p, condition)
	}
	return nil
}

func ValidateRequiredSlice[E any](p *field.Path, value []E, condition string) *field.Error {
	if len(value) == 0 {
		return field.Required(p, condition)
	}
	return nil
}

func Or(errs ...*field.Error) *field.Error {
	if len(errs) == 0 {
		return nil
	}
	for _, err := range errs {
		if err == nil {
			return nil
		}
	}
	return errs[0]
}

func Optional(err *field.Error) *field.Error {
	if err == nil || err.Type == field.ErrorTypeRequired {
		return nil
	}
	return err
}

var isValidSubregion = regexp.MustCompile(`(cloudgouv-)?(eu|us|ap)-(north|east|south|west|northeast|northwest|southeast|southwest)-[1-2][a-c]`).MatchString

// ValidateSubregionName checks that subregionName is a valid az format
func ValidateSubregion(p *field.Path, value string) *field.Error {
	if value == "" {
		return nil
	}
	switch {
	case isValidSubregion(value):
		return nil
	default:
		return field.Invalid(p, value, "invalid subregion")
	}
}
