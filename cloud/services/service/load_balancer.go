package service

import(
    "regexp"
)

func ValidateLoadBalancerName(loadBalancerName string) (bool) {
   isValidate := regexp.MustCompile(`^(?!-)[a-zA-Z0-9\-\/]{0,32}(?<!-)$`).MatchString
   return isValidate(loadBalancerName)
}
