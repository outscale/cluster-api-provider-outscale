package v1beta1

import (
        "k8s.io/apimachinery/pkg/util/validation/field"
        "errors"
        "strings"
	"net"
	"regexp"
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

func ValidateOscClusterSpec(spec OscClusterSpec) field.ErrorList {
	var allErrs field.ErrorList
	if spec.Network.Net.IpRange != "" {
		if errs := ValidateNetworkCidr(spec.Network.Net.IpRange, field.NewPath("ipRange")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Network.LoadBalancer.LoadBalancerType != "" {
		if errs := ValidateNetworkLoadBalancerType(spec.Network.LoadBalancer.LoadBalancerType, field.NewPath("loadBalancerType")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Network.LoadBalancer.HealthCheck.CheckInterval != 0 {
		if errs := ValidateLoadBalancerInterval(spec.Network.LoadBalancer.HealthCheck.CheckInterval, field.NewPath("checkInterval")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.HealthyThreshold != 0 {
		if errs := ValidateLoadBalancerThreshold(spec.Network.LoadBalancer.HealthCheck.HealthyThreshold, field.NewPath("healthyThreshold")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.UnhealthyThreshold != 0 {
		if errs := ValidateLoadBalancerThreshold(spec.Network.LoadBalancer.HealthCheck.UnhealthyThreshold, field.NewPath("unhealthyThreshold")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.Protocol != "" {
		if errs := ValidateLoadBalancerProtocol(spec.Network.LoadBalancer.HealthCheck.Protocol, field.NewPath("protocol")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.Listener.BackendProtocol != "" {
		if errs := ValidateLoadBalancerProtocol(spec.Network.LoadBalancer.Listener.BackendProtocol, field.NewPath("backendProtocol")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.Listener.LoadBalancerProtocol != "" {
		if errs := ValidateLoadBalancerProtocol(spec.Network.LoadBalancer.Listener.LoadBalancerProtocol, field.NewPath("loadBalancerProtocol")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.Timeout != 0 {
		if errs := ValidateLoadBalancerTimeout(spec.Network.LoadBalancer.HealthCheck.Timeout, field.NewPath("timeout")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.Listener.BackendPort != 0{
		if errs := ValidateNetworkPort(spec.Network.LoadBalancer.Listener.BackendPort, field.NewPath("backendPort")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	
	if spec.Network.LoadBalancer.Listener.LoadBalancerPort != 0 {
		if errs := ValidateNetworkPort(spec.Network.LoadBalancer.Listener.LoadBalancerPort, field.NewPath("loadBalancerPort")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if spec.Network.LoadBalancer.HealthCheck.Port != 0 {
		if errs := ValidateNetworkPort(spec.Network.LoadBalancer.HealthCheck.Port, field.NewPath("port")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	
	if spec.Network.LoadBalancer.LoadBalancerName != "" {
		if errs := ValidateNetworkLoadBalancerName(spec.Network.LoadBalancer.LoadBalancerName, field.NewPath("loadBalancerName")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

        if len(spec.Network.Subnets) != 0 {
                subnetsSpec := spec.Network.Subnets
                for _, subnetSpec := range subnetsSpec {
                        if errs := ValidateNetworkCidr(subnetSpec.IpSubnetRange, field.NewPath("ipSubnetRange")); len(errs) > 0 {
                                allErrs = append(allErrs, errs...)
                        }
                }
        }
		
	if len(spec.Network.SecurityGroups) != 0 {
		securityGroupsSpec := spec.Network.SecurityGroups
		for _, securityGroupSpec := range securityGroupsSpec {
			if errs := ValidateSecurityGroupDescription(securityGroupSpec.Description, field.NewPath("description")); len(errs) > 0 {
				allErrs = append(allErrs, errs...)
			}
			if len(securityGroupSpec.SecurityGroupRules) != 0 {
				securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
				for _, securityGroupRuleSpec := range securityGroupRulesSpec {	
					if errs := ValidateSecurityGroupIpProtocol(securityGroupRuleSpec.IpProtocol, field.NewPath("ipProtocol")); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
					if errs := ValidateSecurityGroupFlow(securityGroupRuleSpec.Flow, field.NewPath("flow")); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
					if errs := ValidateNetworkPort(securityGroupRuleSpec.FromPortRange, field.NewPath("fromPortRange")); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
					if errs := ValidateNetworkPort(securityGroupRuleSpec.ToPortRange, field.NewPath("toPortRange")); len(errs) > 0 {
						allErrs = append(allErrs, errs...)
					}
					if errs := ValidateNetworkCidr(securityGroupRuleSpec.IpRange, field.NewPath("ipRange")); len(errs) > 0 {
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
					if errs := ValidateNetworkCidr(routeSpec.Destination, field.NewPath("destination")); len(errs) > 0 {
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

func ValidateNetworkCidr(networkCidr string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateCidr(networkCidr)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, networkCidr, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateNetworkPort(networkPort int32, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := ValidatePort(networkPort)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, networkPort, err.Error()))
		return allErrs
	}
	return allErrs
}


func ValidateSecurityGroupIpProtocol(securityGroupIpProtocol string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateIpProtocol(securityGroupIpProtocol)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, securityGroupIpProtocol, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateSecurityGroupFlow(securityGroupFlow string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := ValidateFlow(securityGroupFlow)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, securityGroupFlow, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateSecurityGroupDescription(securityGroupDescription string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := ValidateDescription(securityGroupDescription)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, securityGroupDescription, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateNetworkLoadBalancerType(networkLoadBalancerType string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateLoadBalancerType(networkLoadBalancerType)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, networkLoadBalancerType, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateLoadBalancerInterval(loadBalancerInterval int32, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateInterval(loadBalancerInterval)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, loadBalancerInterval, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateLoadBalancerProtocol(loadBalancerProtocol string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateProtocol(loadBalancerProtocol)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, loadBalancerProtocol, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateLoadBalancerThreshold(loadBalancerThreshold int32, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := ValidateThreshold(loadBalancerThreshold)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, loadBalancerThreshold, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateLoadBalancerTimeout(loadBalancerTimeout int32, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := ValidateTimeout(loadBalancerTimeout)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, loadBalancerTimeout, err.Error()))
		return allErrs
	}
	return allErrs
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
func ValidateLoadBalancerName(loadBalancerName string)  (string, error){
        isValidateLoadBalancerName := regexp.MustCompile(`^[0-9A-Za-z\s\-]{0,32}$`).MatchString
        if isValidateLoadBalancerName(loadBalancerName) {
                return loadBalancerName, nil
        } else {
                return loadBalancerName, errors.New("Invalid Description")
        }
}

func ValidateNetworkLoadBalancerName(networkLoadBalancerName string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := ValidateLoadBalancerName(networkLoadBalancerName)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, networkLoadBalancerName, err.Error()))
		return allErrs
	}
	return allErrs
}


