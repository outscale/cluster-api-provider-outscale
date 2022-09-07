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

// ValidateOscClusterSpec validate each parameters of oscCluster spec.
func ValidateOscClusterSpec(spec OscClusterSpec) field.ErrorList {
	var allErrs field.ErrorList
	if spec.Network.Net.IPRange != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.Net.IPRange, field.NewPath("ipRange"), ValidateCidr); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Network.LoadBalancer.LoadBalancerType != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.LoadBalancerType, field.NewPath("loadBalancerType"), ValidateLoadBalancerType); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Network.LoadBalancer.HealthCheck.CheckInterval != 0 {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.HealthCheck.CheckInterval, field.NewPath("checkInterval"), ValidateInterval); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.HealthyThreshold != 0 {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.HealthCheck.HealthyThreshold, field.NewPath("healthyThreshold"), ValidateThreshold); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.UnhealthyThreshold != 0 {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.HealthCheck.UnhealthyThreshold, field.NewPath("unhealthyThreshold"), ValidateThreshold); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.Protocol != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.HealthCheck.Protocol, field.NewPath("protocol"), ValidateProtocol); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.Listener.BackendProtocol != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.Listener.BackendProtocol, field.NewPath("backendProtocol"), ValidateProtocol); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.Listener.LoadBalancerProtocol != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.Listener.LoadBalancerProtocol, field.NewPath("loadBalancerProtocol"), ValidateProtocol); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.Timeout != 0 {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.HealthCheck.Timeout, field.NewPath("timeout"), ValidateTimeout); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.Listener.BackendPort != 0 {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.Listener.BackendPort, field.NewPath("backendPort"), ValidatePort); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.Listener.LoadBalancerPort != 0 {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.Listener.LoadBalancerPort, field.NewPath("loadBalancerPort"), ValidatePort); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.Port != 0 {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.HealthCheck.Port, field.NewPath("port"), ValidatePort); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.LoadBalancerName != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.LoadBalancer.LoadBalancerName, field.NewPath("loadBalancerName"), ValidateLoadBalancerName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if len(spec.Network.Subnets) != 0 {
		subnetsSpec := spec.Network.Subnets
		for _, subnetSpec := range subnetsSpec {
			if errs := ValidateAndReturnErrorList(subnetSpec.IPSubnetRange, field.NewPath("ipSubnetRange"), ValidateCidr); len(errs) > 0 {
				allErrs = append(allErrs, errs...)
			}
		}
	}

	if len(spec.Network.SecurityGroups) != 0 {
		securityGroupsSpec := spec.Network.SecurityGroups
		for _, securityGroupSpec := range securityGroupsSpec {
			if errs := ValidateAndReturnErrorList(securityGroupSpec.Description, field.NewPath("description"), ValidateDescription); len(errs) > 0 {
				allErrs = append(allErrs, errs...)
			}
			if len(securityGroupSpec.SecurityGroupRules) != 0 {
				securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
				for _, securityGroupRuleSpec := range securityGroupRulesSpec {
					if errs := ValidateAndReturnErrorList(securityGroupRuleSpec.IPProtocol, field.NewPath("ipProtocol"), ValidateIPProtocol); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
					if errs := ValidateAndReturnErrorList(securityGroupRuleSpec.Flow, field.NewPath("flow"), ValidateFlow); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
					if errs := ValidateAndReturnErrorList(securityGroupRuleSpec.FromPortRange, field.NewPath("fromPortRange"), ValidatePort); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
					if errs := ValidateAndReturnErrorList(securityGroupRuleSpec.ToPortRange, field.NewPath("toPortRange"), ValidatePort); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
					if errs := ValidateAndReturnErrorList(securityGroupRuleSpec.IPRange, field.NewPath("ipRange"), ValidateCidr); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
				}
			}
		}
	}
	if len(spec.Network.RouteTables) != 0 {
		routeTablesSpec := spec.Network.RouteTables
		for _, routeTableSpec := range routeTablesSpec {
			if len(routeTableSpec.Routes) != 0 {
				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					if errs := ValidateAndReturnErrorList(routeSpec.Destination, field.NewPath("destination"), ValidateCidr); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
				}
			}
		}
	}
	return allErrs
}

// ValidateCidr check that the cidr string is a valide CIDR.
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

// ValidateIPProtocol check that ipProtocol is valid.
func ValidateIPProtocol(protocol string) (string, error) {
	switch {
	case protocol == "tcp" || protocol == "udp" || protocol == "icmp" || protocol == "-1":
		return protocol, nil
	default:
		return protocol, errors.New("invalid protocol")
	}
}

// ValidateFlow check that flow is valid.
func ValidateFlow(flow string) (string, error) {
	switch {
	case flow == "Inbound" || flow == "Outbound":
		return flow, nil
	default:
		return flow, errors.New("invalid flow")
	}
}

// ValidateDescription check that description is valid.
func ValidateDescription(description string) (string, error) {
	isValidateDescription := regexp.MustCompile("^[\x20-\x7E]{0,255}$").MatchString
	if isValidateDescription(description) {
		return description, nil
	}
	return description, errors.New("invalid Description")
}

// ValidatePort check that the  port is a valide port.
func ValidatePort(port int32) (int32, error) {
	if port > minPort && port < maxPort {
		return port, nil
	}
	return port, errors.New("invalid Port")
}

// ValidateLoadBalancerType check that the  loadBalancerType is a valid.
func ValidateLoadBalancerType(loadBalancerType string) (string, error) {
	if loadBalancerType == "internet-facing" || loadBalancerType == "internal" {
		return loadBalancerType, nil
	}
	return loadBalancerType, errors.New("invalid LoadBalancerType")
}

// ValidateInterval check that the interval is a valide time of second.
func ValidateInterval(interval int32) (int32, error) {
	if interval > minInterval && interval < maxInterval {
		return interval, nil
	}
	return interval, errors.New("invalid Interval")
}

// ValidateThreshold check that the threshold is a valide number of ping.
func ValidateThreshold(threshold int32) (int32, error) {
	if threshold > minThreshold && threshold < maxThreshold {
		return threshold, nil
	}
	return threshold, errors.New("invalid threshold")
}

// ValidateProtocol check that the protocol string is a valide protocol.
func ValidateProtocol(protocol string) (string, error) {
	switch {
	case protocol == "HTTP" || protocol == "TCP":
		return protocol, nil
	case protocol == "SSL" || protocol == "HTTPS":
		return protocol, errors.New("ssl certificat is required")
	default:
		return protocol, errors.New("invalid protocol")
	}
}

// ValidateTimeout check that the timeoout is a valide maximum time of second.
func ValidateTimeout(timeout int32) (int32, error) {
	if timeout > minTimeout && timeout < maxTimeout {
		return timeout, nil
	}
	return timeout, errors.New("invalid Timeout")
}

// ValidateLoadBalancerName check that the loadBalancerName is a valide name of load balancer.
func ValidateLoadBalancerName(loadBalancerName string) (string, error) {
	isValidateLoadBalancerName := regexp.MustCompile(`^[0-9A-Za-z\s\-]{0,32}$`).MatchString
	if isValidateLoadBalancerName(loadBalancerName) {
		return loadBalancerName, nil
	}
	return loadBalancerName, errors.New("invalid Description")
}

// ValidateAndReturnErrorList is a generic function to validate and return error.
func ValidateAndReturnErrorList[T any](value T, fieldPath *field.Path, validateFunc func(T) (T, error)) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := validateFunc(value)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, value, err.Error()))
		return allErrs
	}
	return allErrs
}
