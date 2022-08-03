package v1beta1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOscClusterTemplate_ValidateCreate(t *testing.T) {
	clusterTestCases := []struct {
		name string
		clusterSpec OscClusterSpec
		expValidateCreateErr error
	}{
		{
			name: "create with bad ip range prefix net",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						IpRange: "10.0.0.0/36",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: ipRange: Invalid value: \"10.0.0.0/36\": invalid CIDR address: 10.0.0.0/36"),
		},
		{
			name: "create with bad ip range net",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						IpRange: "10.0.0.256/16",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: ipRange: Invalid value: \"10.0.0.256/16\": invalid CIDR address: 10.0.0.256/16"),
		},
		{
			name: "create with bad ip range prefix subnet",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Subnets: []*OscSubnet{
						{
							Name:          "test-webhook",
							IpSubnetRange: "10.0.0.0/36",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: ipSubnetRange: Invalid value: \"10.0.0.0/36\": invalid CIDR address: 10.0.0.0/36"),
		},
		{
			name: "create route with bad ip range prefix destination",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					RouteTables: []*OscRouteTable{
						{
							Routes: []OscRoute{
								{
									Destination: "10.0.0.0/36",
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: destination: Invalid value: \"10.0.0.0/36\": invalid CIDR address: 10.0.0.0/36"),
		},
		{
			name: "create route with bad ip range destination",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					RouteTables: []*OscRouteTable{
						{
							Routes: []OscRoute{
								{
									Destination: "10.0.0.256/16",
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: destination: Invalid value: \"10.0.0.256/16\": invalid CIDR address: 10.0.0.256/16"),
		},
		{
			name: "create with bad ip range subnet",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Subnets: []*OscSubnet{
						{
							Name:          "test-webhook",
							IpSubnetRange: "10.0.0.256/16",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: ipSubnetRange: Invalid value: \"10.0.0.256/16\": invalid CIDR address: 10.0.0.256/16"),
		},
		{
			name: "create with bad ipProtocol securityGroupRule",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "sctp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: ipProtocol: Invalid value: \"sctp\": Invalid protocol"),
		},
		{
			name: "create with bad flow securityGroupRule",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "NoBound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: flow: Invalid value: \"NoBound\": Invalid flow"),
		},
		{
			name: "create with bad description securityGroup",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook λ",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: description: Invalid value: \"test webhook λ\": Invalid Description"),
		},
		{
			name: "create with bad fromPortRange securityGroup",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 65537,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: fromPortRange: Invalid value: 65537: Invalid Port"),
		},
		{
			name: "create with bad toPortRange securityGroupRule",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   65537,
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: toPortRange: Invalid value: 65537: Invalid Port"),
		},
		{
			name: "create with bad ip range prefix securityGroupRule",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.0.0/36",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: ipRange: Invalid value: \"10.0.0.0/36\": invalid CIDR address: 10.0.0.0/36"),
		},
		{
			name: "create with bad ip range securityGroupRule",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.0.256/16",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: ipRange: Invalid value: \"10.0.0.256/16\": invalid CIDR address: 10.0.0.256/16"),
		},
		{
			name: "create with bad loadBalancerType",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						LoadBalancerType: "internet",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: loadBalancerType: Invalid value: \"internet\": Invalid LoadBalancerType"),
		},
		{
			name: "create with bad loadBalancerName",
			clusterSpec: OscClusterSpec {
				Network: OscNetwork {
					LoadBalancer: OscLoadBalancer {
						LoadBalancerName: "test-webhook@test",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: loadBalancerName: Invalid value: \"test-webhook@test\": Invalid Description"),
		},
		{
			name: "create with bad healthCheck interval",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						HealthCheck: OscLoadBalancerHealthCheck{
							CheckInterval: 602,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: checkInterval: Invalid value: 602: Invalid Interval"),
		},
		{
			name: "create with bad healthy threshold",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						HealthCheck: OscLoadBalancerHealthCheck{
							HealthyThreshold: 12,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: healthyThreshold: Invalid value: 12: Invalid threshold"),
		},
		{
			name: "create with bad unhealthy threshold",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						HealthCheck: OscLoadBalancerHealthCheck{
							UnhealthyThreshold: 12,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: unhealthyThreshold: Invalid value: 12: Invalid threshold"),
		},
		{
			name: "create with bad healthcheck protocol",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						HealthCheck: OscLoadBalancerHealthCheck{
							Protocol: "SCTP",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: protocol: Invalid value: \"SCTP\": Invalid protocol"),
		},
		{
			name: "create with bad backend protocol loadBalancer",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						Listener: OscLoadBalancerListener{
							BackendProtocol: "SCTP",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: backendProtocol: Invalid value: \"SCTP\": Invalid protocol"),
		},
		{
			name: "create with bad protocol loadBalancer",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						Listener: OscLoadBalancerListener{
							LoadBalancerProtocol: "SCTP",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: loadBalancerProtocol: Invalid value: \"SCTP\": Invalid protocol"),
		},
		{
			name: "create with bad timeout",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						HealthCheck: OscLoadBalancerHealthCheck{
							Timeout: 62,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: timeout: Invalid value: 62: Invalid Timeout"),
		},
		{
			name: "create with bad backend port",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						Listener: OscLoadBalancerListener{
							BackendPort: 65537,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: backendPort: Invalid value: 65537: Invalid Port"),
		},
		{
			name: "create with bad loadBalancer port",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						Listener: OscLoadBalancerListener{
                                                        LoadBalancerPort: 65537,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: loadBalancerPort: Invalid value: 65537: Invalid Port"),
		},
		{
			name: "create with bad healthcheck loadBalancer port",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						HealthCheck: OscLoadBalancerHealthCheck{
							Port: 65537,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: port: Invalid value: 65537: Invalid Port"),
		},
	}
	for _, ctc := range clusterTestCases {
		t.Run(ctc.name, func(t *testing.T) {
			oscInfraClusterTemplate := createOscInfraClusterTemplate(ctc.clusterSpec, "webhook-test", "default")
			err := oscInfraClusterTemplate.ValidateCreate()
			if err != nil {
				assert.Equal(t, ctc.expValidateCreateErr.Error(), err.Error(), "ValidateCreate() should return the same error")
			} else {
				assert.Nil(t, ctc.expValidateCreateErr)
			}
		})
	}
}

func TestOscClusterTemplate_ValidateUpdate(t *testing.T) {
	clusterTestCases := []struct {
		name                 string
		oldClusterSpec       OscClusterSpec
		newClusterSpec       OscClusterSpec
		expValidateUpdateErr error
	}{
		{
			name: "Update only oscClusterTemplate name",
			oldClusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						Name:    "test-webhook",
						IpRange: "10.0.0.0/24",
					},
					Subnets: []*OscSubnet{
						{
							Name:          "test-webhook",
							IpSubnetRange: "10.0.0.32/28",
						},
					},
					RouteTables: []*OscRouteTable{
						{
							Name: "test-webhook",
							Routes: []OscRoute{
								{
									Name:        "test-webhook",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.0.32/28",
									FromPortRange: 10250,
									ToPortRange:   10250,
								},
							},
						},
					},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "test-webhook",
						LoadBalancerType: "internet-facing",
					},
				},
			},
			newClusterSpec: OscClusterSpec{

				Network: OscNetwork{
					Net: OscNet{
						Name:    "test-webhook",
						IpRange: "10.0.0.0/24",
					},
					Subnets: []*OscSubnet{
						{
							Name:          "test-webhook",
							IpSubnetRange: "10.0.0.32/28",
						},
					},
					RouteTables: []*OscRouteTable{
						{
							Name: "test-webhook",
							Routes: []OscRoute{
								{
									Name:        "test-webhook",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.0.32/28",
									FromPortRange: 10250,
									ToPortRange:   10250,
								},
							},
						},
					},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "test-webhook",
						LoadBalancerType: "internet-facing",
					},
				},
			},
			expValidateUpdateErr: nil,
		},
	{
			name: "Update only net ipRange oscClusterTemplate",
			oldClusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						Name:    "test-webhook",
						IpRange: "10.0.0.0/24",
					},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "test-webhook",
						LoadBalancerType: "internet-facing",
					},
				},
			},
			newClusterSpec: OscClusterSpec{

				Network: OscNetwork{
					Net: OscNet{
						Name:    "test-webhook",
						IpRange: "10.0.0.1/24",
					},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "test-webhook",
						LoadBalancerType: "internet-facing",
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscClusterTemmplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: OscClusterTemplate.spec.template.spec: Invalid value: v1beta1.OscClusterTemplate{TypeMeta:v1.TypeMeta{Kind:\"\", APIVersion:\"\"}, ObjectMeta:v1.ObjectMeta{Name:\"webhook-test\", GenerateName:\"\", Namespace:\"default\", SelfLink:\"\", UID:\"\", ResourceVersion:\"\", Generation:0, CreationTimestamp:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), DeletionTimestamp:<nil>, DeletionGracePeriodSeconds:(*int64)(nil), Labels:map[string]string(nil), Annotations:map[string]string(nil), OwnerReferences:[]v1.OwnerReference(nil), Finalizers:[]string(nil), ClusterName:\"\", ManagedFields:[]v1.ManagedFieldsEntry(nil)}, Spec:v1beta1.OscClusterTemplateSpec{Template:v1beta1.OscClusterTemplateResource{ObjectMeta:v1beta1.ObjectMeta{Labels:map[string]string(nil), Annotations:map[string]string(nil)}, Spec:v1beta1.OscClusterSpec{Network:v1beta1.OscNetwork{LoadBalancer:v1beta1.OscLoadBalancer{LoadBalancerName:\"test-webhook\", LoadBalancerType:\"internet-facing\", SubnetName:\"\", SecurityGroupName:\"\", Listener:v1beta1.OscLoadBalancerListener{BackendPort:0, BackendProtocol:\"\", LoadBalancerPort:0, LoadBalancerProtocol:\"\"}, HealthCheck:v1beta1.OscLoadBalancerHealthCheck{CheckInterval:0, HealthyThreshold:0, Port:0, Protocol:\"\", Timeout:0, UnhealthyThreshold:0}}, Net:v1beta1.OscNet{Name:\"test-webhook\", IpRange:\"10.0.0.1/24\", ResourceId:\"\"}, Subnets:[]*v1beta1.OscSubnet(nil), InternetService:v1beta1.OscInternetService{Name:\"\", ResourceId:\"\"}, NatService:v1beta1.OscNatService{Name:\"\", PublicIpName:\"\", SubnetName:\"\", ResourceId:\"\"}, RouteTables:[]*v1beta1.OscRouteTable(nil), SecurityGroups:[]*v1beta1.OscSecurityGroup(nil), PublicIps:[]*v1beta1.OscPublicIp(nil)}, ControlPlaneEndpoint:v1beta1.APIEndpoint{Host:\"\", Port:0}}}}}: OscClusterTemplate spec.template.spec field is immutable."),
		},

	}
	for _, ctc := range clusterTestCases {
		t.Run(ctc.name, func(t *testing.T) {
			oscOldInfraClusterTemplate := createOscInfraClusterTemplate(ctc.oldClusterSpec, "old-webhook-test", "default")
			oscInfraClusterTemplate := createOscInfraClusterTemplate(ctc.newClusterSpec, "webhook-test", "default")
			err := oscInfraClusterTemplate.ValidateUpdate(oscOldInfraClusterTemplate)
			if err != nil {
				assert.Equal(t, ctc.expValidateUpdateErr.Error(), err.Error(), "ValidateUpdate should return the same error")
			} else {
				assert.Nil(t, ctc.expValidateUpdateErr)
			}
		})
	}
}

func createOscInfraClusterTemplate(infraClusterSpec OscClusterSpec, name string, namespace string) *OscClusterTemplate {
	oscInfraClusterTemplate := &OscClusterTemplate{
		Spec: OscClusterTemplateSpec{
			Template: OscClusterTemplateResource{
				Spec: infraClusterSpec,
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return oscInfraClusterTemplate
}
