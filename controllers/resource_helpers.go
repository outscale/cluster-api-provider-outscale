/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import "errors"

const (
	defaultResource = "default"
)

var (
	ErrNoResourceFound    = errors.New("not found")
	ErrMissingResource    = errors.New("missing resource")
	ErrNoChangeToResource = errors.New("resource has not changed")
)

func getResource(name string, m map[string]string) string {
	if m == nil {
		return ""
	}
	return m[name]
}
