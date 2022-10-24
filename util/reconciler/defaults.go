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

package reconciler

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"strconv"
	"time"
)

const (
	DefaultLoopTimeout    = 90 * time.Minute
	DefaultMappingTimeout = 60 * time.Second
)

var ThrottlingErrors = []int{429, 503}

func DefaultedLoopTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultLoopTimeout
	}

	return timeout
}

// getEnv return env variable
func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// KeepRetryWithError retry based on httpResCode and httpResType.
func KeepRetryWithError(requestStr string, httpCode int, throttlingErrors []int) bool {
	for _, v := range throttlingErrors {
		if httpCode == v {
			fmt.Printf("Retry even if got (%v) error type with code error (%v) on request (%s)", httpCode, throttlingErrors, requestStr)
			return true
		}
	}
	return false
}

// EnvBackoff is the environment for backoff function.
func EnvBackoff() wait.Backoff {
	// BACKOFF_DURATION integer in second The initial duration.
	duration, err := strconv.Atoi(getEnv("BACKOFF_DURATION", "1"))
	if err != nil {
		duration = 1
	}

	// BACKOFF_FACTOR float Duration is multiplied by factor each iteration
	factor, err := strconv.ParseFloat(getEnv("BACKOFF_FACTOR", "2.0"), 32)
	if err != nil {
		factor = 1.8
	}

	// BACKOFF_STEPS integer : The remaining number of iterations in which
	// the duration parameter may change
	steps, err := strconv.Atoi(getEnv("BACKOFF_STEPS", "20"))
	if err != nil {
		steps = 13
	}
	fmt.Printf("Debug Returning backoff with params: duration(%v), factor(%v), steps(%v) \n", duration, factor, steps)
	return wait.Backoff{
		Duration: time.Duration(duration) * time.Second,
		Factor:   factor,
		Steps:    steps,
	}
}
