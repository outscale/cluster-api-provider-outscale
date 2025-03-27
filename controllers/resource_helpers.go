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
