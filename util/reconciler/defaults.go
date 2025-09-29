/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"time"
)

const (
	DefaultLoopTimeout    = 90 * time.Minute
	DefaultMappingTimeout = 60 * time.Second
)

func DefaultedLoopTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultLoopTimeout
	}

	return timeout
}
