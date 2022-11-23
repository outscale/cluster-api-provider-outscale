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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkKeypairFormatParameters check keypair format
func checkKeypairFormatParameters(machineScope *scope.MachineScope) (string, error) {
	machineScope.V(2).Info("Check Keypair parameters")
	var keypairSpec *infrastructurev1beta1.OscKeypair
	nodeSpec := machineScope.GetNode()
	if nodeSpec.KeyPair.Name == "" {
		nodeSpec.SetKeyPairDefaultValue()
		keypairSpec = &nodeSpec.KeyPair
	} else {
		keypairSpec = machineScope.GetKeypair()
	}

	keypairName := keypairSpec.Name
	keypairTagName, err := tag.ValidateTagNameValue(keypairName)
	if err != nil {
		return keypairTagName, err
	}

	return "", nil
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
	machineScope.V(2).Info("Create Keypair or add existing one")
	var keypairSpec *infrastructurev1beta1.OscKeypair
	keypairSpec = machineScope.GetKeypair()
	machineScope.V(4).Info("Get Keypair if existing", "keypair", keypairSpec.Name)
	keypairRef := machineScope.GetKeypairRef()
	keypairName := keypairSpec.Name
	var keypair *osc.Keypair
	var err error

	if len(keypairRef.ResourceMap) == 0 {
		keypairRef.ResourceMap = make(map[string]string)
	}
	keypairRef.ResourceMap[keypairName] = keypairName

	if keypair, err = keypairSvc.GetKeyPair(keypairName); err != nil {
		machineScope.V(4).Info("######### fail to get keypair #####", "keypair", keypairSpec.Name)
		return reconcile.Result{}, err
	}
	if keypair == nil {
		machineScope.V(4).Info("######### key pair will be created #####", "keypair", keypairSpec.Name)
		_, err := keypairSvc.CreateKeyPair(keypairName)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if keypairSpec.ResourceId == "" {
		machineScope.V(4).Info("######### key pair Ressource id is empty  #####", "keypair", keypairSpec.Name)
		keypairRef.ResourceMap[keypairName] = keypairName
	}
	machineScope.V(4).Info("######## Get Keypair after reconcile keypair ######", "keypair", keypairSpec.Name)
	return reconcile.Result{}, nil
}

// reconcileDeleteKeypair reconcile the destruction of the keypair of the machine
func reconcileDeleteKeypair(ctx context.Context, machineScope *scope.MachineScope, keypairSvc security.OscKeyPairInterface) (reconcile.Result, error) {
	oscmachine := machineScope.OscMachine
	machineScope.V(2).Info("Delete Key pair")
	keypairSpec := machineScope.GetKeypair()
	keypairName := keypairSpec.Name

	keypair, err := keypairSvc.GetKeyPair(keypairName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if keypair == nil {
		controllerutil.RemoveFinalizer(oscmachine, "")
		return reconcile.Result{}, err
	}
	deleteKeypair := machineScope.GetDeleteKeypair()
	if deleteKeypair {
		machineScope.V(2).Info("Remove keypair")
		err = keypairSvc.DeleteKeyPair(keypairName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("Can not delete keypair for OscCluster %s/%s", machineScope.GetNamespace(), machineScope.GetName())
		}
	} else {
		machineScope.V(2).Info("Keep keypair")
	}

	return reconcile.Result{}, nil
}
