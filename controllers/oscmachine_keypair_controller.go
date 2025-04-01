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

package controllers

import (
	"context"
	"fmt"

	osc "github.com/outscale/osc-sdk-go/v2"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkKeypairFormatParameters check keypair format
func checkKeypairFormatParameters(machineScope *scope.MachineScope) (string, error) {
	var keypairSpec *infrastructurev1beta1.OscKeypair
	nodeSpec := machineScope.GetNode()
	if nodeSpec.KeyPair.Name == "" {
		nodeSpec.SetKeyPairDefaultValue()
		keypairSpec = &nodeSpec.KeyPair
	} else {
		keypairSpec = machineScope.GetKeypair()
	}

	keypairName := keypairSpec.Name
	machineScope.V(2).Info("Check Keypair parameters")
	keypairTagName, err := tag.ValidateTagNameValue(keypairName)
	if err != nil {
		return keypairTagName, err
	}

	return "", nil
}

// checkKeypairSameName check that keypair name is the same in vm and keypair section
func checkKeypairSameName(machineScope *scope.MachineScope) error {
	var keypairSpec *infrastructurev1beta1.OscKeypair
	nodeSpec := machineScope.GetNode()
	if nodeSpec.KeyPair.Name == "" {
		nodeSpec.SetKeyPairDefaultValue()
		keypairSpec = &nodeSpec.KeyPair
	} else {
		keypairSpec = machineScope.GetKeypair()
	}
	keypairName := keypairSpec.Name
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	machineScope.V(2).Info("Check keypair name is the same in vm and keypair section ")
	vmKeypairName := vmSpec.KeypairName
	if keypairName != vmKeypairName {
		return fmt.Errorf("%s is not the same in vm and keypair section", keypairName)
	}
	return nil
}

// getKeyPairResourceId return the keypairName from the resourceMap base on resourceName (tag name + cluster uid)
func getKeyPairResourceId(resourceName string, machineScope *scope.MachineScope) (string, error) {
	keypairRef := machineScope.GetKeypairRef()
	if keypairName, ok := keypairRef.ResourceMap[resourceName]; ok {
		return keypairName, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// reconcileKeypair reconcile the keypair of the machine
func reconcileKeypair(ctx context.Context, machineScope *scope.MachineScope, keypairSvc security.OscKeyPairInterface) (reconcile.Result, error) {

	keypairSpec := machineScope.GetKeypair()
	keypairRef := machineScope.GetKeypairRef()
	keypairName := keypairSpec.Name
	machineScope.V(2).Info("Get Keypair if existing", "keypair", keypairName)
	var keypair *osc.Keypair
	var err error

	if len(keypairRef.ResourceMap) == 0 {
		keypairRef.ResourceMap = make(map[string]string)
	}
	keypairRef.ResourceMap[keypairName] = keypairName

	if keypair, err = keypairSvc.GetKeyPair(keypairName); err != nil {
		machineScope.V(2).Info("Fail to get keypair", "keypair", keypairName)
		return reconcile.Result{}, err
	}
	if keypair == nil {
		machineScope.V(2).Info("Keypair will be created", "keypair", keypairName)
		_, err := keypairSvc.CreateKeyPair(keypairName)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if keypairSpec.ResourceId == "" {
		machineScope.V(4).Info("Keypair Ressource id is empty", "keypair", keypairName)
		keypairRef.ResourceMap[keypairName] = keypairName
	}
	machineScope.V(2).Info("Get Keypair after reconcile keypair", "keypair", keypairName)
	return reconcile.Result{}, nil
}

// reconcileDeleteKeypair reconcile the destruction of the keypair of the machine
func reconcileDeleteKeypair(ctx context.Context, machineScope *scope.MachineScope, keypairSvc security.OscKeyPairInterface) (reconcile.Result, error) {
	keypairSpec := machineScope.GetKeypair()
	keypairName := keypairSpec.Name

	if keypairName == "" {
		machineScope.V(3).Info("Machine has no keypair")
		return reconcile.Result{}, nil
	}
	deleteKeypair := machineScope.GetDeleteKeypair()
	if !deleteKeypair {
		machineScope.V(3).Info("Keeping keypair", "keypair", keypairName)
		return reconcile.Result{}, nil
	}

	keypair, err := keypairSvc.GetKeyPair(keypairName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if keypair == nil {
		machineScope.V(3).Info("Keypair is already deleted", "keypair", keypairName)
		return reconcile.Result{}, err
	}
	machineScope.V(2).Info("Deleting keypair", "keypair", keypairName)
	err = keypairSvc.DeleteKeyPair(keypairName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete keypair: %w", err)
	}

	return reconcile.Result{}, nil
}
