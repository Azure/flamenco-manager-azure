package aznetwork

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azauth"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
)

// NetworkStack contains all the network info we need.
type NetworkStack struct {
	VNet      network.VirtualNetwork
	PublicIP  network.PublicIPAddress
	PrivateIP string
	Interface network.Interface
}

func getNicClient(config azconfig.AZConfig) network.InterfacesClient {
	nicClient := network.NewInterfacesClient(config.SubscriptionID)
	nicClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	return nicClient
}

func getVnetClient(config azconfig.AZConfig) network.VirtualNetworksClient {
	vnetClient := network.NewVirtualNetworksClient(config.SubscriptionID)
	vnetClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	return vnetClient
}

func getIPClient(config azconfig.AZConfig) network.PublicIPAddressesClient {
	ipClient := network.NewPublicIPAddressesClient(config.SubscriptionID)
	ipClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	return ipClient
}

func getSubnetsClient(config azconfig.AZConfig) network.SubnetsClient {
	subnetsClient := network.NewSubnetsClient(config.SubscriptionID)
	subnetsClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	return subnetsClient
}

// CreateNetworkStack creates a virtual network, a public IP, and a NIC.
func CreateNetworkStack(ctx context.Context, config azconfig.AZConfig, basename string) NetworkStack {
	vnet := createVirtualNetwork(ctx, config, basename+"-vnet")
	publicIP := createPublicIP(ctx, config, basename+"-ip")
	nic := createNIC(ctx, config, vnet, publicIP, basename+"-nic")
	privateIP := findPrivateIP(config, nic)
	return NetworkStack{vnet, publicIP, privateIP, nic}
}

func createVirtualNetwork(ctx context.Context, config azconfig.AZConfig, vnetName string) network.VirtualNetwork {
	vnetClient := getVnetClient(config)

	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
		"vnetName":      vnetName,
	})
	logger.Info("creating virtual network")

	future, err := vnetClient.CreateOrUpdate(
		ctx,
		config.ResourceGroup,
		vnetName,
		network.VirtualNetwork{
			Location: to.StringPtr(config.Location),
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				Subnets: &[]network.Subnet{{
					Name: to.StringPtr("default"),
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr("10.0.0.0/16"),
						ServiceEndpoints: &[]network.ServiceEndpointPropertiesFormat{{
							Service: to.StringPtr("Microsoft.Storage"),
						}},
					},
				}},
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{"10.0.0.0/8"},
				},
			},
		})

	if err != nil {
		logger.WithError(err).Fatal("error creating virtual network")
	}

	err = future.WaitForCompletionRef(ctx, vnetClient.Client)
	if err != nil {
		logger.WithError(err).Fatal("error creating virtual network")
	}

	vnet, err := future.Result(vnetClient)
	if err != nil {
		logger.WithError(err).Fatal("error creating virtual network")
	}

	return vnet
}

func createPublicIP(ctx context.Context, config azconfig.AZConfig, ipName string) network.PublicIPAddress {
	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
		"ipName":        ipName,
	})
	logger.Info("creating public IP")

	ipClient := getIPClient(config)
	future, err := ipClient.CreateOrUpdate(
		ctx,
		config.ResourceGroup,
		ipName,
		network.PublicIPAddress{
			Name:     to.StringPtr(ipName),
			Location: to.StringPtr(config.Location),
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
			},
		},
	)
	if err != nil {
		logger.WithError(err).Fatal("error creating public IP address")
	}

	err = future.WaitForCompletionRef(ctx, ipClient.Client)
	if err != nil {
		logger.WithError(err).Fatal("error creating public IP address")
	}

	ip, err := future.Result(ipClient)
	if err != nil {
		logger.WithError(err).Fatal("error creating public IP address")
	}

	logger.WithField("publicIP", *ip.PublicIPAddressPropertiesFormat.IPAddress).Info("public IP created")
	return ip
}

func createNIC(ctx context.Context, config azconfig.AZConfig,
	vnet network.VirtualNetwork, publicIP network.PublicIPAddress,
	nicName string,
) network.Interface {
	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
		"nicName":       nicName,
		"vnet":          *vnet.Name,
	})

	if vnet.Subnets == nil || len(*vnet.Subnets) == 0 {
		logger.Fatal("virtual network has no subnet")
	}
	subnet := (*vnet.Subnets)[0]
	logger = logger.WithField("subnet", *subnet.Name)

	logger.Info("creating network interface card")
	nicParams := network.Interface{
		Name:     to.StringPtr(nicName),
		Location: to.StringPtr(config.Location),
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					Name: to.StringPtr("ipConfig1"),
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						Subnet:                    &subnet,
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress:           &publicIP,
					},
				},
			},
		},
	}

	nicClient := getNicClient(config)
	future, err := nicClient.CreateOrUpdate(ctx, config.ResourceGroup, nicName, nicParams)
	if err != nil {
		logger.WithError(err).Fatal("error creating network interface card")
	}

	err = future.WaitForCompletionRef(ctx, nicClient.Client)
	if err != nil {
		logger.WithError(err).Fatal("error creating network interface card")
	}

	nic, err := future.Result(nicClient)
	if err != nil {
		logger.WithError(err).Fatal("error creating network interface card")
	}

	return nic
}

// GetNetworkStack obtains virtual network components from a NIC.
func GetNetworkStack(ctx context.Context, config azconfig.AZConfig, nicID string) NetworkStack {
	nic := findNIC(ctx, config, nicID)
	publicIP := findPublicIP(ctx, config, nic)
	privateIP := findPrivateIP(config, nic)
	vnet := findVNet(ctx, config, nic)

	return NetworkStack{vnet, publicIP, privateIP, nic}
}

func findNIC(ctx context.Context, config azconfig.AZConfig, nicID string) network.Interface {
	// From the NIC ID, get its name; somehow we only get the ID from the VM, but we can only get the nic by its name.
	parts := strings.Split(nicID, "/")
	nicName := parts[len(parts)-1]

	nicClient := getNicClient(config)
	nic, err := nicClient.Get(ctx, config.ResourceGroup, nicName, "")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"nicID":         nicID,
			logrus.ErrorKey: err,
		}).Fatal("unable to get NIC")
	}

	return nic
}

func findPrivateIP(config azconfig.AZConfig, nic network.Interface) string {
	for _, ipConfig := range *nic.IPConfigurations {
		if ipConfig.PrivateIPAddress == nil || *ipConfig.PrivateIPAddress == "" {
			continue
		}

		return *ipConfig.PrivateIPAddress
	}

	logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
		"nicID":         *nic.ID,
	}).Fatal("this NIC has no private IP address")
	return ""
}

func findPublicIP(ctx context.Context, config azconfig.AZConfig, nic network.Interface) network.PublicIPAddress {
	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
		"nicID":         *nic.ID,
	})
	logger.Debug("finding public IP address")

	var publicIPID string
	for _, ipConfig := range *nic.IPConfigurations {
		if ipConfig.PublicIPAddress == nil {
			continue
		}

		publicIPID = *ipConfig.PublicIPAddress.ID
		break
	}
	if publicIPID == "" {
		logger.Fatal("unable to find public IP address")
	}

	ipClient := getIPClient(config)
	ipIDParts := strings.Split(publicIPID, "/")
	ipName := ipIDParts[len(ipIDParts)-1]
	publicIP, err := ipClient.Get(ctx, config.ResourceGroup, ipName, "")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"nicID":         *nic.ID,
			"publicIPID":    publicIPID,
			logrus.ErrorKey: err,
		}).Fatal("unable to retrieve public IP")
	}

	return publicIP
}

func findVNet(ctx context.Context, config azconfig.AZConfig, nic network.Interface) network.VirtualNetwork {
	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
		"nicID":         *nic.ID,
	})

	if nic.IPConfigurations == nil || len(*nic.IPConfigurations) == 0 {
		logger.Fatal("NIC has no IP configurations")
	}

	// Splitting the ID string without verifying it has the format we expect is a hack,
	// but it's unclear how we can correctly obtain the name of the virtual network.
	ipConfig := (*nic.IPConfigurations)[0]
	logger = logger.WithField("subnet", *ipConfig.Subnet.ID)
	logger.Debug("found subnet")

	subnetParts := strings.Split(*ipConfig.Subnet.ID, "/")
	vnetName := subnetParts[len(subnetParts)-3]
	logger = logger.WithField("vnet", vnetName)

	vnetClient := getVnetClient(config)
	vnet, err := vnetClient.Get(ctx, config.ResourceGroup, vnetName, "")
	if err != nil {
		logger.WithError(err).Fatal("unable to get virtual network")
	}

	return vnet
}
