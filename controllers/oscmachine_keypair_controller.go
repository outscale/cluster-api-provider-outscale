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

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	log := ctrl.LoggerFrom(ctx)
	keypairSpec := machineScope.GetKeypair()
	keypairRef := machineScope.GetKeypairRef()
	keypairName := keypairSpec.Name
	var keypair *osc.Keypair
	var err error

	if len(keypairRef.ResourceMap) == 0 {
		keypairRef.ResourceMap = make(map[string]string)
	}
	keypairRef.ResourceMap[keypairName] = keypairName

	log.V(4).Info("Checking keypair", "keypair", keypairName)
	if keypair, err = keypairSvc.GetKeyPair(ctx, keypairName); err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get keypair: %w", err)
	}
	if keypair == nil {
		log.V(2).Info("Creating keypair", "keypair", keypairName)
		_, err := keypairSvc.CreateKeyPair(ctx, keypairName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create keypair: %w", err)
		}
	} else if keypairSpec.ResourceId == "" {
		log.V(4).Info("Setting Keypair Ressource id", "keypair", keypairName)
		keypairRef.ResourceMap[keypairName] = keypairName
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteKeypair reconcile the destruction of the keypair of the machine
func reconcileDeleteKeypair(ctx context.Context, machineScope *scope.MachineScope, keypairSvc security.OscKeyPairInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	oscmachine := machineScope.OscMachine
	keypairSpec := machineScope.GetKeypair()
	keypairName := keypairSpec.Name

	keypair, err := keypairSvc.GetKeyPair(ctx, keypairName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get keypair: %w", err)
	}
	if keypair == nil {
		log.V(3).Info("Keypair is already deleted", "keypair", keypairName)
		controllerutil.RemoveFinalizer(oscmachine, "")
		return reconcile.Result{}, nil
	}
	deleteKeypair := machineScope.GetDeleteKeypair()
	if deleteKeypair {
		log.V(2).Info("Deleting keypair", "keypair", keypairName)
		err = keypairSvc.DeleteKeyPair(ctx, keypairName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete keypair: %w", err)
		}
	} else {
		log.V(3).Info("Keeping keypair", "keypair", keypairName)
	}

	return reconcile.Result{}, nil
}
