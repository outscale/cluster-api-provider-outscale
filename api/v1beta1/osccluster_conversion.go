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
	infrav1beta2 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta2"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscCluster) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OscCluster)
	if err := Convert_v1beta1_OscCluster_To_v1beta2_OscCluster(src, dst, nil); err != nil {
		return err
	}
	restored := &infrav1beta2.OscCluster{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}

	dst.Spec.Network.Bastion = restored.Spec.Network.Bastion
	dst.Spec.Network.Bastion.Enable = restored.Spec.Network.Bastion.Enable
	if dst.Spec.Network.Bastion.Enable == true {
		dst.Spec.Network.Bastion.ClusterName = restored.Spec.Network.Bastion.ClusterName
		dst.Spec.Network.Bastion.DeviceName = restored.Spec.Network.Bastion.DeviceName
		dst.Spec.Network.Bastion.ImageId = restored.Spec.Network.Bastion.ImageId
		dst.Spec.Network.Bastion.ImageName = restored.Spec.Network.Bastion.ImageName
		dst.Spec.Network.Bastion.KeypairName = restored.Spec.Network.Bastion.KeypairName
		dst.Spec.Network.Bastion.Name = restored.Spec.Network.Bastion.Name
		dst.Spec.Network.Bastion.PrivateIps = restored.Spec.Network.Bastion.PrivateIps
		dst.Spec.Network.Bastion.PublicIpName = restored.Spec.Network.Bastion.PublicIpName
		dst.Spec.Network.Bastion.ResourceId = restored.Spec.Network.Bastion.ResourceId
		dst.Spec.Network.Bastion.RootDisk = restored.Spec.Network.Bastion.RootDisk
		dst.Spec.Network.Bastion.SecurityGroupNames = restored.Spec.Network.Bastion.SecurityGroupNames
		dst.Spec.Network.Bastion.SubnetName = restored.Spec.Network.Bastion.SubnetName
		dst.Spec.Network.Bastion.VmType = restored.Spec.Network.Bastion.VmType
	}
	if restored.Spec.Network.SubregionName != "" {
		dst.Spec.Network.SubregionName = restored.Spec.Network.SubregionName
	}
	dst.ObjectMeta = src.ObjectMeta
	dst.Status.Network.LinkRouteTableRef = restored.Status.Network.LinkRouteTableRef
	for _, restoredRouteTable := range restored.Spec.Network.RouteTables {
		for _, dstRouteTable := range dst.Spec.Network.RouteTables {
			dstRouteTable.Subnets = restoredRouteTable.Subnets
		}
	}

	for i, dstRouteTable := range dst.Spec.Network.RouteTables {
		dstRouteTable.Subnets = append(dstRouteTable.Subnets, src.Spec.Network.RouteTables[i].SubnetName)
	}
	return nil

}

func (dst *OscCluster) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OscCluster)
	dst.ObjectMeta = src.ObjectMeta
	if err := Convert_v1beta2_OscCluster_To_v1beta1_OscCluster(src, dst, nil); err != nil {
		return err
	}

	if err := utilconversion.MarshalData(src, dst); err != nil {
		return err
	}
	return nil
}

func (dst *OscClusterList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OscClusterList)
	return Convert_v1beta2_OscClusterList_To_v1beta1_OscClusterList(src, dst, nil)
}

func Convert_v1beta1_OscResourceMapReference_To_v1beta2_OscResourceReference(in *OscResourceMapReference, out *infrav1beta2.OscResourceReference, s apiconversion.Scope) error {
	out = (*infrav1beta2.OscResourceReference)(in.DeepCopy())
	return nil
}

func Convert_v1beta2_OscResourceReference_To_v1beta1_OscResourceMapReference(in *infrav1beta2.OscResourceReference, out *OscResourceMapReference, s apiconversion.Scope) error {
	out = (*OscResourceMapReference)(in.DeepCopy())
	return nil
}

func Convert_v1beta2_OscNetworkResource_To_v1beta1_OscNetworkResource(in *infrav1beta2.OscNetworkResource, out *OscNetworkResource, s apiconversion.Scope) error {
	if err := autoConvert_v1beta2_OscNetworkResource_To_v1beta1_OscNetworkResource(in, out, s); err != nil {
		return err
	}
	for key, value := range in.LinkRouteTableRef {
		if out.LinkRouteTableRef.ResourceMap == nil {
			out.LinkRouteTableRef.ResourceMap = make(map[string]string)
		}
		out.LinkRouteTableRef.ResourceMap[key] = value[0]
	}

	return nil
}

func Convert_v1beta1_OscNetworkResource_To_v1beta2_OscNetworkResource(in *OscNetworkResource, out *infrav1beta2.OscNetworkResource, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_OscNetworkResource_To_v1beta2_OscNetworkResource(in, out, s); err != nil {
		return err
	}
	for key, value := range in.LinkRouteTableRef.ResourceMap {
		if len(out.LinkRouteTableRef) == 0 {
			out.LinkRouteTableRef = make(map[string][]string)
		}
		out.LinkRouteTableRef[key] = []string{value}
	}

	out.InternetServiceRef = infrav1beta2.OscResourceReference(in.InternetServiceRef)
	out.NatServiceRef = infrav1beta2.OscResourceReference(in.NatServiceRef)
	out.NetRef = infrav1beta2.OscResourceReference(in.NetRef)
	out.SubnetRef = infrav1beta2.OscResourceReference(in.SubnetRef)
	out.SecurityGroupsRef = infrav1beta2.OscResourceReference(in.SecurityGroupsRef)
	out.RouteTablesRef = infrav1beta2.OscResourceReference(in.RouteTablesRef)
	out.SecurityGroupRuleRef = infrav1beta2.OscResourceReference(in.SecurityGroupRuleRef)
	out.RouteRef = infrav1beta2.OscResourceReference(in.RouteRef)
	return nil
}

func Convert_v1beta2_OscNetwork_To_v1beta1_OscNetwork(in *infrav1beta2.OscNetwork, out *OscNetwork, s apiconversion.Scope) error {
	if err := autoConvert_v1beta2_OscNetwork_To_v1beta1_OscNetwork(in, out, s); err != nil {
		return err
	}
	for _, outRouteTable := range out.RouteTables {
		for _, inRouteTable := range in.RouteTables {
			outRouteTable.SubnetName = inRouteTable.Subnets[0]
		}
	}
	return nil
}
func Convert_v1beta2_OscClusterStatus_To_v1beta1_OscClusterStatus(in *infrav1beta2.OscClusterStatus, out *OscClusterStatus, s apiconversion.Scope) error {
	if err := autoConvert_v1beta2_OscClusterStatus_To_v1beta1_OscClusterStatus(in, out, s); err != nil {
		return err
	}

	return nil
}

func Convert_v1beta2_OscRouteTable_To_v1beta1_OscRouteTable(in *infrav1beta2.OscRouteTable, out *OscRouteTable, s apiconversion.Scope) error {
	if err := autoConvert_v1beta2_OscRouteTable_To_v1beta1_OscRouteTable(in, out, s); err != nil {
		return err
	}
	out.SubnetName = in.Subnets[0]
	return nil
}

func Convert_v1beta1_OscRouteTable_To_v1beta2_OscRouteTable(in *OscRouteTable, out *infrav1beta2.OscRouteTable, s apiconversion.Scope) error {
	if err := autoConvert_v1beta1_OscRouteTable_To_v1beta2_OscRouteTable(in, out, s); err != nil {
		return err
	}
	out.Subnets[0] = in.SubnetName
	return nil
}
