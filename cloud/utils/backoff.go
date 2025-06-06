/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
