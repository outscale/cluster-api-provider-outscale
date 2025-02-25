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

// ValidateOscClusterSpec validates a OscClusterSpec.
func ValidateOscClusterSpec(spec OscClusterSpec) field.ErrorList {
	var allErrs field.ErrorList

	if spec.Network.LoadBalancer.LoadBalancerName != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.LoadBalancerName, field.NewPath("network", "loadBalancer", "loadbalancername"), ValidateLoadBalancerName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Network.LoadBalancer.LoadBalancerType != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.LoadBalancerType, field.NewPath("network", "loadBalancer", "loadbalancertype"), ValidateLoadBalancerType); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Network.Net.IpRange != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.Net.IpRange, field.NewPath("network", "net", "ipRange"), ValidateCidr); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	return allErrs
}

// ValidateCidr checks that the cidr string is a valid CIDR
func ValidateCidr(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	return err
}

// ValidateIpProtocol checks that ipProtocol is valid
func ValidateIpProtocol(protocol string) error {
	switch protocol {
	case "tcp", "udp", "icmp", "-1":
		return nil
	default:
		return errors.New("invalid protocol")
	}
}

// ValidateFlow checks that flow is valid
func ValidateFlow(flow string) error {
	switch flow {
	case "Inbound", "Outbound":
		return nil
	default:
		return errors.New("invalid flow (allowed: Inbound, Outbound)")
	}
}

// ValidateDescription checks that description is valid
func ValidateDescription(description string) error {
	isValidateDescription := regexp.MustCompile("^[\x20-\x7E]{0,255}$").MatchString
	if isValidateDescription(description) {
		return nil
	} else {
		return errors.New("invalid description")
	}
}

// ValidatePort checks that the  port is a valid port
func ValidatePort(port int32) error {
	if port > minPort && port < maxPort {
		return nil
	} else {
		return errors.New("invalid port")
	}
}

// ValidateLoadBalancerType checks that the  loadBalancerType is a valid
func ValidateLoadBalancerType(loadBalancerType string) error {
	switch loadBalancerType {
	case "internet-facing", "internal":
		return nil
	default:
		return errors.New("invalid loadBalancer type (allowed: internet-facing, internal)")
	}
}

// ValidateInterval checks that the interval is a valid time of second
func ValidateInterval(interval int32) error {
	if interval > minInterval && interval < maxInterval {
		return nil
	} else {
		return errors.New("invalid interval")
	}
}

// ValidateThreshold checks that the threshold is a valid number of ping
func ValidateThreshold(threshold int32) error {
	if threshold > minThreshold && threshold < maxThreshold {
		return nil
	} else {
		return errors.New("invalid threshold")
	}
}

// ValidateProtocol checks that the protocol string is a valid protocol
func ValidateProtocol(protocol string) error {
	switch protocol {
	case "HTTP", "TCP":
		return nil
	case "SSL", "HTTPS":
		return errors.New("SSL certificate is required")
	default:
		return errors.New("invalid protocol")
	}
}

// ValidateTimeout checks that the timeoout is a valid maximum time of second
func ValidateTimeout(timeout int32) error {
	if timeout > minTimeout && timeout < maxTimeout {
		return nil
	} else {
		return errors.New("invalid timeout")
	}
}

var isValidateLoadBalancerName = regexp.MustCompile(`^[0-9A-Za-z\s\-]{0,32}$`).MatchString

// ValidateLoadBalancerName checks that the loadBalancerName is a valid name of load balancer
func ValidateLoadBalancerName(loadBalancerName string) error {
	if isValidateLoadBalancerName(loadBalancerName) {
		return nil
	} else {
		return errors.New("invalid description")
	}
}

func ValidateAndReturnErrorList[T any](value T, fieldPath *field.Path, validateFunc func(T) error) field.ErrorList {
	allErrs := field.ErrorList{}
	err := validateFunc(value)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, value, err.Error()))
		return allErrs
	}
	return allErrs
}
