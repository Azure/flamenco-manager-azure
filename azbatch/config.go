/* (c) 2019, Blender Foundation
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package azbatch

import (
	"context"
	"fmt"

	"github.com/Azure/flamenco-manager-azure/flamenco"

	"github.com/Azure/azure-sdk-for-go/services/batch/2018-12-01.8.0/batch"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"github.com/Azure/flamenco-manager-azure/azconfig"
	"github.com/Azure/flamenco-manager-azure/aznetwork"
	"github.com/Azure/flamenco-manager-azure/azstorage"
	"github.com/Azure/flamenco-manager-azure/textio"
)

// AskParametersAndSave asks the user for the batch pool parameters and saves them in the config.
func AskParametersAndSave(ctx context.Context, config *azconfig.AZConfig, defaultPoolName string) {
	if config.Batch != nil && config.Batch.PoolID != "" && config.Batch.VMSize != "" {
		logrus.WithFields(logrus.Fields{
			"poolID":                 config.Batch.PoolID,
			"vmSize":                 config.Batch.VMSize,
			"targetDedicatedNodes":   config.Batch.TargetDedicatedNodes,
			"targetLowPriorityNodes": config.Batch.TargetLowPriorityNodes,
		}).Info("batch pool config loaded")
		return
	}

	poolID := textio.ReadLineWithDefault(ctx, "Desired batch pool ID", defaultPoolName)
	if poolID == "" {
		logrus.Fatal("no batch pool ID given, aborting")
	}

	fmt.Println()
	fmt.Println("For sizes, see https://docs.microsoft.com/azure/batch/batch-pool-vm-sizes")
	vmSize := textio.ReadLine(ctx, "Desired batch node VM size [Standard_F16s]")
	if vmSize == "" {
		vmSize = "Standard_F16s"
	}

	targetDedicatedNodes := textio.ReadNonNegativeInt(ctx, "Target dedicated node count [0]", true)
	targetLowPriorityNodes := textio.ReadNonNegativeInt(ctx, "Target low-priority node count [0]", true)

	config.Batch = &azconfig.AZBatchConfig{
		PoolID:                 poolID,
		VMSize:                 vmSize,
		TargetDedicatedNodes:   int32(targetDedicatedNodes),
		TargetLowPriorityNodes: int32(targetLowPriorityNodes),
	}
	config.Save()
}

// PoolParameters returns the batch pool parameters.
func PoolParameters(config azconfig.AZConfig, netStack aznetwork.NetworkStack) batch.PoolAddParameter {
	mountOpts := azstorage.GetMountOptions(config, "flamenco-resources")
	startCmd := fmt.Sprintf("bash -exc 'sudo mkdir -p /mnt/flamenco-resources; "+
		"sudo groupadd --force %s; "+
		"grep \" /mnt/flamenco-resources \" -q /proc/mounts || sudo mount -t cifs //%s.file.core.windows.net/flamenco-resources /mnt/flamenco-resources -o %s; "+
		"bash -ex /mnt/flamenco-resources/flamenco-worker-startup.sh'",
		flamenco.UnixGroupName,
		config.StorageCreds.Username, mountOpts,
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
