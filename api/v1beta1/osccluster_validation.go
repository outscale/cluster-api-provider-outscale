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

package v1beta1

import (
	"errors"
	"net"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	minPort      = 0
	maxPort      = 65536
	minInterval  = 4
	maxInterval  = 601
	minThreshold = 0
	maxThreshold = 11
	minTimeout   = 1
	maxTimeout   = 61
)

// ValidateOscClusterSpec validate each parameters of oscCluster spec
func ValidateOscClusterSpec(spec OscClusterSpec) field.ErrorList {
	var allErrs field.ErrorList

	if spec.Network.LoadBalancer.LoadBalancerName != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.LoadBalancerName, field.NewPath("loadBalancerName"), ValidateLoadBalancerName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	return allErrs
}

// ValidateCidr check that the cidr string is a valide CIDR
func ValidateCidr(cidr string) (string, error) {
	if !strings.Contains(cidr, "/") {
		return cidr, errors.New("invalid Not A CIDR")
	}
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr, err
	}
	return cidr, nil
}

// ValidateIpProtocol check that ipProtocol is valid
func ValidateIpProtocol(protocol string) (string, error) {
	switch {
	case protocol == "tcp" || protocol == "udp" || protocol == "icmp" || protocol == "-1":
		return protocol, nil
	default:
		return protocol, errors.New("Invalid protocol")
	}
}

// ValidateFlow check that flow is valid
func ValidateFlow(flow string) (string, error) {
	switch {
	case flow == "Inbound" || flow == "Outbound":
		return flow, nil
	default:
		return flow, errors.New("Invalid flow")
	}
}

// ValidateDescription check that description is valid
func ValidateDescription(description string) (string, error) {
	isValidateDescription := regexp.MustCompile("^[\x20-\x7E]{0,255}$").MatchString
	if isValidateDescription(description) {
		return description, nil
	} else {
		return description, errors.New("Invalid Description")
	}
}

// ValidatePort check that the  port is a valide port
func ValidatePort(port int32) (int32, error) {
	if port > minPort && port < maxPort {
		return port, nil
	} else {
		return port, errors.New("Invalid Port")
	}
}

// ValidateLoadBalancerType check that the  loadBalancerType is a valid
func ValidateLoadBalancerType(loadBalancerType string) (string, error) {
	if loadBalancerType == "internet-facing" || loadBalancerType == "internal" {
		return loadBalancerType, nil
	} else {
		return loadBalancerType, errors.New("Invalid LoadBalancerType")
	}
}

// ValidateInterval check that the interval is a valide time of second
func ValidateInterval(interval int32) (int32, error) {
	if interval > minInterval && interval < maxInterval {
		return interval, nil
	} else {
		return interval, errors.New("Invalid Interval")
	}
}

// ValidateThreshold check that the threshold is a valide number of ping
func ValidateThreshold(threshold int32) (int32, error) {
	if threshold > minThreshold && threshold < maxThreshold {
		return threshold, nil
	} else {
		return threshold, errors.New("Invalid threshold")
	}
}

// ValidateProtocol check that the protocol string is a valide protocol
func ValidateProtocol(protocol string) (string, error) {
	switch {
	case protocol == "HTTP" || protocol == "TCP":
		return protocol, nil
	case protocol == "SSL" || protocol == "HTTPS":
		return protocol, errors.New("Ssl certificat is required")
	default:
		return protocol, errors.New("Invalid protocol")
	}
}

// ValidateTimeout check that the timeoout is a valide maximum time of second
func ValidateTimeout(timeout int32) (int32, error) {
	if timeout > minTimeout && timeout < maxTimeout {
		return timeout, nil
	} else {
		return timeout, errors.New("Invalid Timeout")
	}
}

// ValidateLoadBalancerName check that the loadBalancerName is a valide name of load balancer
func ValidateLoadBalancerName(loadBalancerName string) (string, error) {
	isValidateLoadBalancerName := regexp.MustCompile(`^[0-9A-Za-z\s\-]{0,32}$`).MatchString
	if isValidateLoadBalancerName(loadBalancerName) {
		return loadBalancerName, nil
	} else {
		return loadBalancerName, errors.New("Invalid Description")
	}
}

func ValidateAndReturnErrorList[T any](value T, fieldPath *field.Path, validateFunc func(T) (T, error)) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := validateFunc(value)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, value, err.Error()))
		return allErrs
	}
	return allErrs
}
