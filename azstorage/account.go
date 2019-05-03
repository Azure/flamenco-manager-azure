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

package azstorage

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/flamenco-deploy-azure/azauth"
	"gitlab.com/blender-institute/flamenco-deploy-azure/azconfig"
	"gitlab.com/blender-institute/flamenco-deploy-azure/textio"
)

func getAccountClient(config azconfig.AZConfig) storage.AccountsClient {
	accountClient := storage.NewAccountsClient(config.SubscriptionID)
	accountClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	// accountClient.RequestInspector = azdebug.LogRequest()
	// accountClient.ResponseInspector = azdebug.LogResponse()
	return accountClient
}

// AskAccountName asks for a storage account name, potentially overridable by a CLI arg.
func AskAccountName(
	ctx context.Context, config azconfig.AZConfig,
	cliAccountName, defaultAccountName string,
) (desiredName string, mustCreate bool) {
	if cliAccountName != "" {
		logrus.WithField("storageAccountName", cliAccountName).Debug("creating storage account from CLI")
		return cliAccountName, true
	}

	if config.StorageAccountName != "" {
		logrus.WithField("storageAccountName", config.StorageAccountName).Info("storage account known, not creating new one")
		return config.StorageAccountName, false
	}

	desiredName = textio.ReadLineWithDefault(ctx, "Desired storage account name", defaultAccountName)
	if desiredName == "" {
		logrus.Fatal("no storage account name given, aborting")
	}

	return desiredName, true
}

// CreateAndSave creates a storage account and stores it in the config.
func CreateAndSave(ctx context.Context, config *azconfig.AZConfig, accountName string) {
	account, ok := CreateAccount(ctx, *config, accountName)
	if !ok {
		logrus.Fatal("unable to create storage account")
	}

	config.StorageAccountName = *account.Name
	logrus.WithField("storageAccountName", config.StorageAccountName).Info("storage account created")
	config.Save()
}

// CheckAvailability checks whether the desired storage account name is still available.
func CheckAvailability(ctx context.Context, config azconfig.AZConfig, accountName string) (isAvailable bool) {
	accountClient := getAccountClient(config)

	logger := logrus.WithFields(logrus.Fields{
		"storageAccountName": accountName,
		"resourceGroup":      config.ResourceGroup,
		"location":           config.Location,
	})
	logger.Info("checking storage account name availability")

	result, err := accountClient.CheckNameAvailability(
		ctx,
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(accountName),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		})
	if err != nil {
		logger.WithError(err).Error("storage account check-name-availability failed")
		return
	}

	if !*result.NameAvailable {
		logger.WithFields(logrus.Fields{
			"serverMessage": *result.Message,
			"reason":        result.Reason,
		}).Error("storage account name not available")
		return
	}

	return true
}

// CreateAccount creates a new azure storage account
func CreateAccount(ctx context.Context, config azconfig.AZConfig, accountName string) (storage.Account, bool) {
	accountClient := getAccountClient(config)

	logger := logrus.WithFields(logrus.Fields{
		"storageAccountName": accountName,
		"resourceGroup":      config.ResourceGroup,
		"location":           config.Location,
	})

	logger.Info("creating storage account")
	future, err := accountClient.Create(
		ctx,
		config.ResourceGroup,
		accountName,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardLRS},
			Kind:                              storage.Storage,
			Location:                          to.StringPtr(config.Location),
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{},
		})
	if err != nil {
		logrus.WithError(err).Error("failed to start creating storage account")
		return storage.Account{}, false
	}

	err = future.WaitForCompletionRef(ctx, accountClient.Client)
	if err != nil {
		logrus.WithError(err).Error("failed waiting for storage account creation")
		return storage.Account{}, false
	}

	account, err := future.Result(accountClient)
	if err != nil {
		logrus.WithError(err).Error("failed retrieving storage account")
		return storage.Account{}, false
	}

	return account, true
}
