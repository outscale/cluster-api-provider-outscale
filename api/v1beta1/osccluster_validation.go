package v1beta1

import (
	"errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"net"
	"regexp"
	"strings"
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
	if spec.Network.Net.IpRange != "" {
		if errs := ValidateAndReturnErrorList(spec.Network.Net.IpRange, field.NewPath("ipRange"), ValidateCidr); len(errs) > 0 {
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
			if errs := ValidateAndReturnErrorList(subnetSpec.IpSubnetRange, field.NewPath("ipSubnetRange"), ValidateCidr); len(errs) > 0 {
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
					if errs := ValidateAndReturnErrorList(securityGroupRuleSpec.IpProtocol, field.NewPath("ipProtocol"), ValidateIpProtocol); len(errs) > 0 {
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
					if errs := ValidateAndReturnErrorList(securityGroupRuleSpec.IpRange, field.NewPath("ipRange"), ValidateCidr); len(errs) > 0 {
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

// ValidateCidr check that the cidr string is a valide CIDR
func ValidateCidr(cidr string) (string, error) {
	if !strings.Contains(cidr, "/") {
		return cidr, errors.New("Invalid Not A CIDR")
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
