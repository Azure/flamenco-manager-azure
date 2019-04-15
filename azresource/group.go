package azresource

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azauth"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/textio"
)

// EnsureResourceGroup creates a resource group if config.ResourceGroup is "".
// The program is aborted when creation is required but fails.
func EnsureResourceGroup(ctx context.Context, config *azconfig.AZConfig) {
	if config.ResourceGroup != "" {
		logrus.WithField("resourceGroup", config.ResourceGroup).Info("resource group known, not creating new one")
		return
	}

	config.ResourceGroup = textio.ReadLine(ctx, "Desired resource group name")
	if config.ResourceGroup == "" {
		logrus.Fatal("no resource group name given, aborting")
	}

	group, ok := createResourceGroup(ctx, *config)
	if !ok {
		logrus.Fatal("unable to create resource group")
	}

	config.ResourceGroup = *group.Name
	logrus.WithField("resourceGroup", config.ResourceGroup).Info("resource group created")
	config.Save()
}

// createResourceGroup creates a new azure resource group
func createResourceGroup(ctx context.Context, config azconfig.AZConfig) (resources.Group, bool) {
	groupsClient := resources.NewGroupsClient(config.SubscriptionID)
	groupsClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	// groupsClient.RequestInspector = azdebug.LogRequest()
	// groupsClient.ResponseInspector = azdebug.LogResponse()

	logger := logrus.WithFields(logrus.Fields{
		"resourceGroup": config.ResourceGroup,
		"location":      config.Location,
	})
	logger.Info("creating resource group")

	group, err := groupsClient.CreateOrUpdate(ctx, config.ResourceGroup, resources.Group{
		Location: to.StringPtr(config.Location),
	})
	if err != nil {
		logger.WithError(err).Error("unable to create new resource group")
		return resources.Group{}, false
	}
	return group, true
}
