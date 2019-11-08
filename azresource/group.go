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

package azresource

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/flamenco-manager-azure/azauth"
	"github.com/Azure/flamenco-manager-azure/azconfig"
	"github.com/Azure/flamenco-manager-azure/textio"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
)

//ListResourceGroups returns the Azure Resource Groups available to this subscription.
func ListResourceGroups(ctx context.Context, config azconfig.AZConfig) (groups []resources.Group) {
	groupsClient := resources.NewGroupsClient(config.SubscriptionID)
	for iter, err := groupsClient.ListComplete(ctx, "", nil); iter.NotDone(); {
		if err != nil {
			logrus.Fatal("got error: %s", err)
		}
		groups = append(groups, iter.Value())
	}
	return
}

// AskResourceGroupName asks for a resource group, potentially overridable by a CLI arg.
func AskResourceGroupName(
	ctx context.Context, config azconfig.AZConfig,
	cliAccountName, defaultAccountName string,
) (desiredName string, mustCreate bool) {
	if cliAccountName != "" {
		logrus.WithField("resourceGroup", cliAccountName).Debug("creating resource group from CLI")
		return cliAccountName, true
	}

	if config.ResourceGroup != "" {
		logrus.WithField("resourceGroup", config.ResourceGroup).Info("resource group known, not creating new one")
		return config.ResourceGroup, false
	}

	available := ListResourceGroups(ctx, config)
	switch len(available) {
	case 0:
		desiredName = textio.ReadLineWithDefault(ctx, "Desired resource group", defaultAccountName)
		if desiredName == "" {
			logrus.Fatal("no resource group given, aborting")
		}
	case 1:
		desiredName = *available[0].Name
		logrus.WithField("resource group", config.ResourceGroup).Info("using the only available resource groups")
	default:
		logrus.WithField("locationCount", len(available)).Info("multiple Azure resource groups available")

		fmt.Println("Available resource groups:")
		for idx, subs := range available {
			fmt.Printf("    %2d: %s\n", idx+1, *subs.Name)
		}
		choice := textio.ReadNonNegativeInt(ctx, "Azure resource group number", false)
		if choice < 1 || choice > len(available) {
			logrus.WithField("index", choice).Fatal("that resource groups is not available")
		}
		logrus.WithField("resource group", config.Location).Info("using Azure resource groups")
		desiredName = *available[choice-1].Name
	}

	return desiredName, true
}

// EnsureResourceGroup creates a resource group and saves it to the config.
// The program is aborted when creation is required but fails.
func EnsureResourceGroup(ctx context.Context, config *azconfig.AZConfig, groupName string) bool{
	config.ResourceGroup = groupName
	group, ok := createResourceGroup(ctx, *config)
	if !ok {
		logrus.Info("unable to create resource group, please specify a different name")
		// Reset the value of ResourceGroup, so that if AskResourceGroupName is called again, the context will be "clean".
		// See how EnsureResourceGroup is used in main.go
		config.ResourceGroup = ""
		return false
	}

	config.ResourceGroup = *group.Name
	logrus.WithField("resourceGroup", config.ResourceGroup).Info("resource group created")
	config.Save()
	return true
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
