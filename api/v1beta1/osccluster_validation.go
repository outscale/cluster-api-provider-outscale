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
	"fmt"
	"net/netip"
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	minPort      = 1
	maxPort      = 65535
	minInterval  = 5
	maxInterval  = 600
	minThreshold = 1
	maxThreshold = 10
	minTimeout   = 2
	maxTimeout   = 60
)

// ValidateOscClusterSpec validates a OscClusterSpec.
func ValidateOscClusterSpec(spec OscClusterSpec) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, ValidateNet(spec.Network.Net)...)
	allErrs = append(allErrs, ValidateSubnets(spec.Network.Subnets, spec.Network.Net)...)
	allErrs = append(allErrs, ValidateNatServices(spec.Network.NatServices, spec.Network.Subnets, spec.Network.Net)...)
	allErrs = append(allErrs, ValidateSecurityGroups(spec.Network.SecurityGroups, spec.Network.Net)...)
	allErrs = append(allErrs, ValidateLoadbalancer(spec.Network.LoadBalancer)...)
	allErrs = append(allErrs, ValidateAllowFromIPs(spec.Network.AllowFromIPRanges)...)
	return allErrs
}

func ValidateNet(spec OscNet) field.ErrorList {
	switch {
	case spec == OscNet{}:
		return nil
	case spec.UseExisting:
		return MergeValidation(
			ValidateRequired(field.NewPath("network", "net", "resourceId"), spec.ResourceId, "must be set when reusing a network"),
			ValidateEmpty(field.NewPath("network", "net", "ipRange"), spec.IpRange, "must not be set when reusing a network"),
		)
	default:
		return MergeValidation(
			ValidateRequired(field.NewPath("network", "net", "ipRange"), spec.IpRange, "must be set when not reusing a network"),
			ValidateCidr(field.NewPath("network", "net", "ipRange"), spec.IpRange),
		)
	}
}

func ValidateSubnets(specs []OscSubnet, net OscNet) field.ErrorList {
	var erl field.ErrorList
	for _, spec := range specs {
		switch {
		case net.UseExisting:
			erl = AppendValidation(erl,
				ValidateRequired(field.NewPath("network", "subnets", "resourceId"), spec.ResourceId, "must be set when reusing a network"),
				ValidateRequiredSlice(field.NewPath("network", "subnets", "roles"), spec.Roles, "must be set when reusing a network"),
				ValidateEmpty(field.NewPath("network", "subnets", "ipSubnetRange"), spec.IpSubnetRange, "must not be set when reusing a network"),
				ValidateEmpty(field.NewPath("network", "subnets", "subregionName"), spec.SubregionName, "must not be set when reusing a network"),
			)
		default:
			erl = AppendValidation(erl,
				ValidateRequired(field.NewPath("network", "subnets", "ipSubnetRange"), spec.IpSubnetRange, "must be set when not reusing a network"),
				ValidateSubregion(field.NewPath("network", "subnets", "subregionName"), spec.SubregionName),
			)
		}
	}
	erl = AppendValidation(erl, ValidateSubnetCidr(specs, net)...)
	return erl
}

func ValidateNatServices(specs []OscNatService, subnets []OscSubnet, net OscNet) field.ErrorList {
	var erl field.ErrorList
	if net.UseExisting {
		return AppendValidation(erl,
			ValidateEmptySlice(field.NewPath("network", "natServices"), specs, "no nat services must be defined when reusing a network"),
		)
	}
	for _, spec := range specs {
		erl = AppendValidation(erl,
			ValidateSubregion(field.NewPath("network", "natServices", "subregionName"), spec.SubregionName),
		)
	}
	return erl
}

func ValidateSecurityGroups(specs []OscSecurityGroup, net OscNet) field.ErrorList {
	var erl field.ErrorList
	for _, spec := range specs {
		switch {
		case net.UseExisting:
			erl = AppendValidation(erl,
				ValidateRequired(field.NewPath("network", "securityGroups", "resourceId"), spec.ResourceId, "must be set when reusing a network"),
				ValidateRequiredSlice(field.NewPath("network", "securityGroups", "roles"), spec.Roles, "must be set when reusing a network"),
				ValidateEmptySlice(field.NewPath("network", "securityGroups", "securityGroupRules"), spec.SecurityGroupRules, "must not be set when reusing a network"),
			)
		default:
			erl = AppendValidation(erl,
				Or(
					ValidateRequired(field.NewPath("network", "securityGroups", "name"), spec.Name, "name or roles must be set"),
					ValidateRequiredSlice(field.NewPath("network", "securityGroups", "roles"), spec.Roles, "name or roles must be set"),
				),
			)
			erl = AppendValidation(erl, ValidateSecurityGroupRules(spec.SecurityGroupRules)...)
		}
	}
	return erl
}

func ValidateSecurityGroupRules(specs []OscSecurityGroupRule) field.ErrorList {
	var erl field.ErrorList
	for _, spec := range specs {
		erl = AppendValidation(erl,
			ValidateFlow(field.NewPath("network", "securityGroups", "securityGroupRules", "flow"), spec.Flow),
			ValidateIpProtocol(field.NewPath("network", "securityGroups", "securityGroupRules", "ipProtocol"), spec.IpProtocol),
			Or(
				ValidateRequired(field.NewPath("network", "securityGroups", "securityGroupRules", "ipRange"), spec.IpRange, "ipRange or ipRanges must be set"),
				ValidateRequiredSlice(field.NewPath("network", "securityGroups", "securityGroupRules", "ipRanges"), spec.IpRanges, "ipRange or ipRanges must be set"),
			),
			ValidateRange(field.NewPath("network", "securityGroups", "securityGroupRules", "fromPortRange"), spec.FromPortRange, minPort, maxPort),
			ValidateRange(field.NewPath("network", "securityGroups", "securityGroupRules", "toPortRange"), spec.ToPortRange, minPort, maxPort),
			ValidatePortRange(field.NewPath("network", "securityGroups", "securityGroupRules", "toPortRange"), spec.FromPortRange, spec.ToPortRange, "toPortRange must be >= fromPortRange"),
		)
		if spec.IpRange != "" {
			erl = AppendValidation(erl,
				ValidateCidr(field.NewPath("network", "securityGroups", "securityGroupRules", "ipRange"), spec.IpRange),
				ValidateEmptySlice(field.NewPath("network", "securityGroups", "securityGroupRules", "ipRanges"), spec.IpRanges, "ipRanges must not be set if ipRange is set"),
			)
		}
		if len(spec.IpRanges) > 0 {
			for _, ipRange := range spec.IpRanges {
				erl = AppendValidation(erl,
					ValidateCidr(field.NewPath("network", "securityGroups", "securityGroupRules", "ipRanges"), ipRange),
				)
			}
		}
	}
	return erl
}

func ValidateLoadbalancer(spec OscLoadBalancer) field.ErrorList {
	var erl field.ErrorList
	erl = AppendValidation(erl,
		ValidateLoadBalancerName(field.NewPath("network", "loadBalancer", "loadbalancername"), spec.LoadBalancerName),
		Optional(ValidateLoadBalancerType(field.NewPath("network", "loadBalancer", "loadbalancertype"), spec.LoadBalancerType)),

		Optional(ValidateRange(field.NewPath("network", "loadBalancer", "listener", "loadbalancerport"), spec.Listener.LoadBalancerPort, minPort, maxPort)),
		Optional(ValidateProtocol(field.NewPath("network", "loadBalancer", "listener", "loadbalancerprotocol"), spec.Listener.LoadBalancerProtocol)),
		Optional(ValidateRange(field.NewPath("network", "loadBalancer", "listener", "backendport"), spec.Listener.BackendPort, minPort, maxPort)),
		Optional(ValidateProtocol(field.NewPath("network", "loadBalancer", "listener", "backendprotocol"), spec.Listener.BackendProtocol)),

		Optional(ValidateRange(field.NewPath("network", "loadBalancer", "healthCheck", "checkinterval"), spec.HealthCheck.CheckInterval, minInterval, maxInterval)),
		Optional(ValidateRange(field.NewPath("network", "loadBalancer", "healthCheck", "port"), spec.HealthCheck.Port, minPort, maxPort)),
		Optional(ValidateProtocol(field.NewPath("network", "loadBalancer", "healthCheck", "protocol"), spec.HealthCheck.Protocol)),
		Optional(ValidateRange(field.NewPath("network", "loadBalancer", "healthCheck", "timeout"), spec.HealthCheck.Timeout, minTimeout, maxTimeout)),
		Optional(ValidateRange(field.NewPath("network", "loadBalancer", "healthCheck", "healthythreshold"), spec.HealthCheck.HealthyThreshold, minThreshold, maxThreshold)),
		Optional(ValidateRange(field.NewPath("network", "loadBalancer", "healthCheck", "unhealthythreshold"), spec.HealthCheck.UnhealthyThreshold, minThreshold, maxThreshold)),
	)
	return erl
}

func ValidateAllowFromIPs(ips []string) field.ErrorList {
	var erl field.ErrorList
	for _, ip := range ips {
		erl = AppendValidation(erl,
			ValidateCidr(field.NewPath("allowFromIPRanges"), ip),
		)
	}
	return erl
}

// ValidateCidr checks that the cidr string is a valid CIDR
func ValidateCidr(p *field.Path, cidr string) *field.Error {
	if cidr == "" {
		return field.Required(p, "a CIDR is required")
	}
	_, err := netip.ParsePrefix(cidr)
	if err != nil {
		return field.Invalid(p, cidr, "invalid CIDR address")
	}
	return nil
}

// ValidateCidr checks that the cidr string is a valid CIDR
func ValidateSubnetCidr(specs []OscSubnet, net OscNet) field.ErrorList {
	p := field.NewPath("network", "subnets", "ipSubnetRange")
	var erl field.ErrorList
	subnets := make([]netip.Prefix, 0, len(specs))
	for _, spec := range specs {
		if spec.IpSubnetRange == "" {
			continue
		}
		subn, err := netip.ParsePrefix(spec.IpSubnetRange)
		if err != nil {
			erl = append(erl, field.Invalid(p, spec.IpSubnetRange, "invalid CIDR address"))
		} else {
			subnets = append(subnets, subn)
		}
	}
	n, err := netip.ParsePrefix(net.IpRange)
	if err != nil {
		return erl
	}
	for i, suba := range subnets {
		if !suba.Overlaps(n) {
			erl = append(erl, field.Invalid(p, suba.String(), "subnet must be contained in net"))
		}
		for j := i + 1; j < len(subnets); j++ {
			if suba.Overlaps(subnets[j]) {
				erl = append(erl, field.Invalid(p, suba.String(), "subnet overlaps "+subnets[j].String()))
			}
		}
	}
	return erl
}

// ValidateIpProtocol checks that ipProtocol is valid
func ValidateIpProtocol(p *field.Path, protocol string) *field.Error {
	if protocol == "" {
		return field.Required(p, "protocol is required")
	}
	switch protocol {
	case "tcp", "udp", "icmp", "-1":
		return nil
	default:
		return field.Invalid(p, protocol, "only tcp, udp, icmp or -1 are allowed")
	}
}

// ValidateFlow checks that flow is valid
func ValidateFlow(p *field.Path, flow string) *field.Error {
	if flow == "" {
		return field.Required(p, "flow is required")
	}
	switch flow {
	case "Inbound", "Outbound":
		return nil
	default:
		return field.Invalid(p, flow, "only Inbound or Outbound are allowed")
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

func ValidateRange[N int | int32](p *field.Path, val, min, max N) *field.Error {
	if val == 0 {
		return field.Required(p, "required")
	}
	if val >= min && val <= max {
		return nil
	}
	return field.Invalid(p, val, fmt.Sprintf("must be between %d and %d", min, max))
}

// ValidatePortRange checks that to >= from.
func ValidatePortRange(p *field.Path, from, to int32, msg string) *field.Error {
	if to >= from {
		return nil
	} else {
		return field.Invalid(p, to, msg)
	}
}

var isValidateLoadBalancerName = regexp.MustCompile(`^[0-9A-Za-z\s\-]{0,32}$`).MatchString

// ValidateLoadBalancerName checks that the loadBalancerName is a valid name of load balancer
func ValidateLoadBalancerName(p *field.Path, loadBalancerName string) *field.Error {
	if loadBalancerName == "" {
		return field.Required(p, "loadBalancer name is required")
	}
	if isValidateLoadBalancerName(loadBalancerName) {
		return nil
	} else {
		return field.Invalid(p, loadBalancerName, "invalid loadBalancer name")
	}
}

// ValidateLoadBalancerType checks that the  loadBalancerType is a valid
func ValidateLoadBalancerType(p *field.Path, loadBalancerType string) *field.Error {
	switch loadBalancerType {
	case "internet-facing", "internal", "":
		return nil
	default:
		return field.Invalid(p, loadBalancerType, "only internet-facing or internal are allowed")
	}
}

// ValidateProtocol checks that the protocol string is a valid protocol
func ValidateProtocol(p *field.Path, protocol string) *field.Error {
	if protocol == "" {
		return field.Required(p, "protocol is required")
	}
	switch protocol {
	case "HTTP", "TCP":
		return nil
	case "SSL", "HTTPS":
		return field.Invalid(p, protocol, "SSL certificate is required")
	default:
		return field.Invalid(p, protocol, "only HTTP and TCP are supported")
	}
}
