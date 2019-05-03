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

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/flamenco-deploy-azure/azauth"
	"gitlab.com/blender-institute/flamenco-deploy-azure/azconfig"
	"gitlab.com/blender-institute/flamenco-deploy-azure/textio"
)

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

	desiredName = textio.ReadLineWithDefault(ctx, "Desired resource group", defaultAccountName)
	if desiredName == "" {
		logrus.Fatal("no resource group given, aborting")
	}

	return desiredName, true
}

// EnsureResourceGroup creates a resource group and saves it to the config.
// The program is aborted when creation is required but fails.
func EnsureResourceGroup(ctx context.Context, config *azconfig.AZConfig, groupName string) {
	config.ResourceGroup = groupName
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
