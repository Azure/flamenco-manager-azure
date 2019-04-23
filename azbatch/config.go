package azbatch

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/batch/2018-12-01.8.0/batch"
	"github.com/Azure/go-autorest/autorest/to"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/aznetwork"
)

// PoolParameters returns the batch pool parameters.
func PoolParameters(config azconfig.AZConfig, netStack aznetwork.NetworkStack) batch.PoolAddParameter {

	startCmd := fmt.Sprintf("bash -exc 'sudo mkdir -p /mnt/flamenco-resources; "+
		"sudo mount -t cifs //%s.file.core.windows.net/flamenco-resources /mnt/flamenco-resources "+
		"-o vers=3.0,username=%s,password=%s,dir_mode=0777,file_mode=0666,sec=ntlmssp,mfsymlinks; "+
		"bash -ex /mnt/flamenco-resources/flamenco-worker-startup.sh'",
		config.StorageCreds.Username, config.StorageCreds.Username, config.StorageCreds.Password,
	)

	return batch.PoolAddParameter{
		ID: to.StringPtr(config.Batch.PoolID),

		VMSize:                 to.StringPtr(config.Batch.VMSize),
		MaxTasksPerNode:        to.Int32Ptr(1),
		TargetDedicatedNodes:   to.Int32Ptr(config.Batch.TargetDedicatedNodes),
		TargetLowPriorityNodes: to.Int32Ptr(config.Batch.TargetLowPriorityNodes),

		VirtualMachineConfiguration: &batch.VirtualMachineConfiguration{
			ImageReference: &batch.ImageReference{
				Publisher: to.StringPtr("Canonical"),
				Sku:       to.StringPtr("18.04-LTS"),
				Offer:     to.StringPtr("UbuntuServer"),
				Version:   to.StringPtr("latest"),
			},
			NodeAgentSKUID: to.StringPtr("batch.node.ubuntu 18.04"),
		},

		NetworkConfiguration: &batch.NetworkConfiguration{
			SubnetID: to.StringPtr(netStack.SubnetID()),
		},

		StartTask: &batch.StartTask{
			CommandLine:    to.StringPtr(startCmd),
			WaitForSuccess: to.BoolPtr(true),
			UserIdentity: &batch.UserIdentity{
				AutoUser: &batch.AutoUserSpecification{
					ElevationLevel: "Admin",
					Scope:          "Pool",
				},
			},
		},
	}

}
