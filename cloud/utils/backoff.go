/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package utils

import (
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

var throttlingErrors = []int{429, 503}

// getEnv return env variable
func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// RetryIf tries on TCP errors (httpResp is nil) or on specific HTTP codes.
func RetryIf(httpResp *http.Response) bool {
	if httpResp == nil {
		return true
	}
	return slices.Contains(throttlingErrors, httpResp.StatusCode)
}

// EnvBackoff is the environment for backoff function.
func EnvBackoff() wait.Backoff {
	// BACKOFF_DURATION integer in second The initial duration.
	duration, err := strconv.Atoi(getEnv("BACKOFF_DURATION", "1"))
	if err != nil {
		duration = 1
	}

	// BACKOFF_FACTOR float Duration is multiplied by factor each iteration
	factor, err := strconv.ParseFloat(getEnv("BACKOFF_FACTOR", "1.5"), 32)
	if err != nil {
		factor = 1.5
	}

	// BACKOFF_STEPS integer : The remaining number of iterations in which
	// the duration parameter may change
	steps, err := strconv.Atoi(getEnv("BACKOFF_STEPS", "10"))
	if err != nil {
		steps = 10
	}
	return wait.Backoff{
		Duration: time.Duration(duration) * time.Second,
		Factor:   factor,
		Steps:    steps,
	}
}
