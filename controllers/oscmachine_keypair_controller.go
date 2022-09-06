package controllers

import (
	"context"
	b64 "encoding/base64"
	"fmt"

	"golang.org/x/crypto/ssh"

	osc "github.com/outscale/osc-sdk-go/v2"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// validKeyPairPublicKey return if public key is valid
func validKeyPairPublicKey(publicKey string) bool {
	publicKeyBinary, err := b64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return false
	}

	_, _, _, _, err = ssh.ParseAuthorizedKey(publicKeyBinary)
	if err != nil {
		return false
	}
	return true
}

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

	publicKey := keypairSpec.PublicKey
	if publicKey == "" {
		return keypairTagName, fmt.Errorf("keypair Public Ip is empty")
	} else {
		if !validKeyPairPublicKey(publicKey) {
			return keypairTagName, fmt.Errorf("Invalid keypairType")
		}
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
	keypairRef := machineScope.GetKeypairRef()
	keypairName := keypairSpec.Name
	var keypair *osc.Keypair
	var err error

	if len(keypairRef.ResourceMap) == 0 {
		keypairRef.ResourceMap = make(map[string]string)
	}
	keypairRef.ResourceMap[keypairName] = keypairSpec.ResourceId

	if keypair, err = keypairSvc.GetKeyPair(keypairName); err != nil {
		return reconcile.Result{}, err
	}
	if keypair == nil || keypairSpec.ResourceId == "" {
		_, err := keypairSvc.CreateKeyPair(keypairName)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		if keypairSpec.PublicKey != "" {
			if !validKeyPairPublicKey(keypairSpec.PublicKey) {
				return reconcile.Result{}, fmt.Errorf("keypair public IP is not valid")
			} else {
				if keypairSpec.ResourceId != "" {
					keypairRef.ResourceMap[keypairName] = keypairSpec.ResourceId
				}
			}
		} else {
			return reconcile.Result{}, fmt.Errorf("keypair public IP is empty")
		}
	}
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

	machineScope.Info("Remove keypair")
	err = keypairSvc.DeleteKeyPair(keypairName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("Can not delete keypair for OscCluster %s/%s", machineScope.GetNamespace(), machineScope.GetName())
	}
	return reconcile.Result{}, nil
}
