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
	machineScope.Info("Check Keypair parameters")

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
	machineScope.Info("Create Keypair or add existing one")
	var keypairSpec *infrastructurev1beta1.OscKeypair
	keypairSpec = machineScope.GetKeypair()
	machineScope.Info("Get Keypair if existing", "keypair", keypairSpec.Name)

	keypairRef := machineScope.GetKeypairRef()
	keypairName := keypairSpec.Name
	var keypair *osc.Keypair
	var err error

	if len(keypairRef.ResourceMap) == 0 {
		keypairRef.ResourceMap = make(map[string]string)
	}
	keypairRef.ResourceMap[keypairName] = keypairName

	if keypair, err = keypairSvc.GetKeyPair(keypairName); err != nil {
		machineScope.Info("######### fail to get keypair #####", "keypair", keypairSpec.Name)
		return reconcile.Result{}, err
	}
	if keypair == nil {
		machineScope.Info("######### key pair will be created #####", "keypair", keypairSpec.Name)
		_, err := keypairSvc.CreateKeyPair(keypairName)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if keypairSpec.ResourceId == "" {
		machineScope.Info("######### key pair Ressource id is empty  #####", "keypair", keypairSpec.Name)
		keypairRef.ResourceMap[keypairName] = keypairName
	}
	machineScope.Info("###### Get Keypair after reconcile keypair ######", "keypair", keypairSpec.Name)
	return reconcile.Result{}, nil
}

// reconcileDeleteKeypair reconcile the destruction of the keypair of the machine
func reconcileDeleteKeypair(ctx context.Context, machineScope *scope.MachineScope, keypairSvc security.OscKeyPairInterface) (reconcile.Result, error) {
	oscmachine := machineScope.OscMachine
	machineScope.Info("Delete Key pair")
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
		machineScope.Info("Remove keypair")
		err = keypairSvc.DeleteKeyPair(keypairName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("Can not delete keypair for OscCluster %s/%s", machineScope.GetNamespace(), machineScope.GetName())
		}
	} else {
		machineScope.Info("Keep keypair")
	}
	return reconcile.Result{}, nil
}
