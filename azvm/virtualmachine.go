package azvm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azauth"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/azdebug"
	"gitlab.com/blender-institute/azure-go-test/textio"
)

func getVMClient(config azconfig.AZConfig) compute.VirtualMachinesClient {
	vmClient := compute.NewVirtualMachinesClient(config.SubscriptionID)
	vmClient.Authorizer = azauth.Load(azure.PublicCloud.ServiceManagementEndpoint)
	vmClient.RequestInspector = azdebug.LogRequest()
	vmClient.ResponseInspector = azdebug.LogResponse()
	return vmClient
}

// ListVMs fetches a list of available virtual machine names.
func ListVMs(ctx context.Context, config azconfig.AZConfig) []string {
	vmClient := getVMClient(config)
	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
	})
	logger.Debug("fetching VM list")

	vmNames := []string{}
	vmListPage, err := vmClient.List(ctx, config.ResourceGroup)
	if err != nil {
		logger.WithError(err).Fatal("unable to fetch list of existing VMs")
	}
	for vmListPage.NotDone() {
		for _, vmInfo := range vmListPage.Values() {
			locationMatches := config.Location == *vmInfo.Location
			logger.WithFields(logrus.Fields{
				"id":              *vmInfo.ID,
				"name":            *vmInfo.Name,
				"location":        *vmInfo.Location,
				"locationMatches": locationMatches,
			}).Debug("found VM")
			if !locationMatches {
				continue
			}
			vmNames = append(vmNames, *vmInfo.Name)
		}

		if err := vmListPage.NextWithContext(ctx); err != nil {
			logger.WithError(err).Fatal("unable to fetch next page of VMs")
		}
	}
	return vmNames
}

// ChooseVM lets the user pick a virtual machine.
// if vmName is not empty, that name is used instead, and this function just determines whether that VM already exists.
func ChooseVM(ctx context.Context, config azconfig.AZConfig, vmName string) (chosenVMName string, isExisting bool) {
	vmNames := ListVMs(ctx, config)
	vmChoices := textio.StrMap(vmNames)

	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
	})
	logger.WithField("numVMs", len(vmNames)).Info("retrieved list of existing VMs")

	// If a name was already given, we don't need to prompt any more.
	if vmName != "" {
		return vmName, vmChoices[vmName]
	}

	if len(vmNames) > 0 {
		vmName, isExisting = textio.Choose(ctx, vmNames, "Desired VM name, can be new or an existing name")
	} else {
		vmName = textio.ReadLine(ctx, "Desired name for new VM")
	}
	if vmName == "" {
		logger.Fatal("no name given, aborting")
	}

	return vmName, isExisting
}

// EnsureVM either returns the VM info (isExisting=true) or creates a new VM (isExisting=false)
func EnsureVM(ctx context.Context, config azconfig.AZConfig, vmName string, isExisting bool) (compute.VirtualMachine, NetworkStack) {
	vmClient := getVMClient(config)

	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
		"vmName":        vmName,
	})
	if !isExisting {
		logger.Info("creating new VM")
		return createVM(ctx, config, vmName)
	}

	logger.Info("retrieving existing VM")
	vm, err := vmClient.Get(ctx, config.ResourceGroup, vmName, compute.InstanceView)
	if err != nil {
		logger.WithError(err).Fatal("unable to retrieve VM info")
	}

	stack := findVMNetworkStack(ctx, config, vm)
	return vm, stack
}

func loadSSHKey() string {
	// TODO: make this configurable/promptable and/or support ssh-agent
	sshPublicKeyPath := os.ExpandEnv("$HOME/.ssh/id_rsa.pub")

	logger := logrus.WithField("sshPublicKeyPath", sshPublicKeyPath)
	sshBytes, err := ioutil.ReadFile(sshPublicKeyPath)
	if err != nil {
		logger.WithError(err).Fatal("failed to read SSH key data")
	}
	return string(sshBytes)
}

func createVM(ctx context.Context, config azconfig.AZConfig, vmName string) (compute.VirtualMachine, NetworkStack) {
	sshKeyData := loadSSHKey()
	adminPassword := RandStringBytes(32)

	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
		"vmName":        vmName,
	})

	netstack := CreateNetworkStack(ctx, config, vmName)

	logger.Info("creating virtual machine")
	vmClient := getVMClient(config)
	future, err := vmClient.CreateOrUpdate(
		ctx,
		config.ResourceGroup,
		vmName,
		compute.VirtualMachine{
			Location: to.StringPtr(config.Location),
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: compute.VirtualMachineSizeTypesStandardDS1V2,
				},
				StorageProfile: &compute.StorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr(publisher),
						Offer:     to.StringPtr(offer),
						Sku:       to.StringPtr(sku),
						Version:   to.StringPtr("latest"),
					},
				},
				OsProfile: &compute.OSProfile{
					ComputerName:  to.StringPtr(vmName),
					AdminUsername: to.StringPtr(adminUsername),
					AdminPassword: to.StringPtr(adminPassword),
					LinuxConfiguration: &compute.LinuxConfiguration{
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{{
								Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", adminUsername)),
								KeyData: to.StringPtr(sshKeyData),
							}},
						},
					},
				},
				NetworkProfile: &compute.NetworkProfile{
					NetworkInterfaces: &[]compute.NetworkInterfaceReference{{
						ID: netstack.Interface.ID,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					}},
				},
			},
		},
	)
	if err != nil {
		logger.WithError(err).Fatal("error creating VM")
	}

	err = future.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		logger.WithError(err).Fatal("error creating VM")
	}

	vm, err := future.Result(vmClient)
	if err != nil {
		logger.WithError(err).Fatal("error creating VM")
	}

	return vm, netstack
}

func findVMNetworkStack(ctx context.Context, config azconfig.AZConfig, vm compute.VirtualMachine) NetworkStack {
	if vm.NetworkProfile == nil || vm.NetworkProfile.NetworkInterfaces == nil || len(*vm.NetworkProfile.NetworkInterfaces) == 0 {
		logrus.Fatal("this VM has no network interface")
	}

	nicRef := (*vm.NetworkProfile.NetworkInterfaces)[0]
	return GetNetworkStack(ctx, config, *nicRef.ID)
}
