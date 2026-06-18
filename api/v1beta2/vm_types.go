package v1beta2

import "github.com/outscale/osc-sdk-go/v3/pkg/osc"

// +kubebuilder:validation:AtMostOneOf:=ID,Name
// +kubebuilder:validation:AtLeastOneOf:=ID,Name
type OscImage struct {
	// The image id.
	ID string `json:"id,omitempty"`
	// The image name.
	Name string `json:"name,omitempty"`
	// The image account owner ID.
	AccountId string `json:"accountId,omitempty"`
	// Use an "Outscale Opensource" image
	OutscaleOpenSource bool `json:"outscaleOpenSource,omitempty"`
}

type OscVolume struct {
	// The volume device (/dev/xvdX)
	Device string `json:"device"`
	// The volume iops (io1 volumes only)
	Iops int `json:"iops,omitempty"`
	// The volume size in gibibytes (GiB)
	Size int `json:"size,omitempty"`
	// The volume type (io1, gp2 or standard)
	// +kubebuilder:validation:Enum:=io1,gp2,standard
	Type osc.VolumeType `json:"volumeType,omitempty"`
	// The id of a snapshot to use as a volume source.
	FromSnapshot string `json:"fromSnapshot,omitempty"`
}

// +kubebuilder:validation:Enum:=leastNodes;random
type SubregionMode string

const (
	SubregionModeLeastNodes SubregionMode = "leastNodes"
	SubregionModeRandom     SubregionMode = "random"
)

type OscFGPU struct {
	// The fGPU model to add to the VM (e.g. nvidia-h100).
	// The fGPU will be released when the node VM is deleted.
	Model string `json:"model,omitempty"`
}

type OscPlacement struct {
	// Try to put VMs with the same repulseServer value on different physical servers. For workers, set by default to the MachineDeployment name unless RepulseCluster is set.
	// Define to an empty string if you want to disable.
	// +optional
	RepulseServer *string `json:"repulseServer,omitempty"`
	// Try to put VMs with the same attractServer value on the same physical server.
	// +optional
	AttractServer string `json:"attractServer,omitempty"`
	// serverStrict makes repulseServer/attractServer mandatory. CreateVm will fail if VM placement is not possible.
	// +optional
	ServerStrict bool `json:"serverStrict,omitempty"`
	// Try to put VMs with the same repulseCluster value on different clusters. Not set by default.
	// +optional
	RepulseCluster string `json:"repulseCluster,omitempty"`
	// Try to put VMs with the same attractCluster value on the same cluster.
	// +optional
	AttractCluster string `json:"attractCluster,omitempty"`
	// clusterStrict makes repulseCluster/attractCluster mandatory. CreateVm will fail if VM placement is not possible.
	// +optional
	ClusterStrict bool `json:"clusterStrict,omitempty"`
}

type OscVm struct {
	Image OscImage `json:"image,omitempty"`
	// keypairName is the name of the keypair to use.
	// +kubebuilder:validation:Required
	KeypairName string `json:"keypairName,omitempty"`
	// vmType defines the type of vm (tinav7.c4r8p1 by default)
	// +optional
	VmType string `json:"vmType,omitempty"`
	// rootVolume defines the root volume.
	RootVolume OscVolume `json:"rootVolume,omitempty"`
	// publicIp defines if a public IP needs be configured.
	// +optional
	PublicIp bool `json:"publicIp,omitempty"`
	// publicIpPool defines the name of the pool from which public IPs will be picked.
	// +optional
	PublicIpPool string `json:"publicIpPool,omitempty"`
	// fGPU configuration for this VM.
	// +optional
	FGPU *OscFGPU `json:"fGPU,omitempty"`
	// subregionMode defines the way nodes will be allocated in subregions (leastNodes or random; by default, leastNodes).
	// +optional
	SubregionMode SubregionMode `json:"subregionMode,omitempty"`
	// subregions lists the subregions where the machines needs to be placed. If empty, the subregions defined at cluster level will be used.
	// +optional
	Subregions []string `json:"subregions,omitempty"`
	// role defines the node role (bastion, controlplane or worker, worker by default).
	// +optional
	Role OscRole `json:"role,omitempty"`
	// tags to add to the VM.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
	// placement defines VM placement constraints.
	// +optional
	Placement OscPlacement `json:"placement,omitempty"`
	// additionalVolumes defines additional volumes to be linked to this VM.
	// +optional
	AdditionalVolumes []OscVolume `json:"additionalVolumes,omitempty"`
}

func (vm *OscVm) GetRole() OscRole {
	if vm.Role != "" {
		return vm.Role
	}
	return RoleWorker
}
