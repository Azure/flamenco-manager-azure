package azstorage

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azauth"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/textio"
)

// EnsureAccount creates a storage account if config.StorageAccountName is "".
// The program is aborted when creation is required but fails.
func EnsureAccount(ctx context.Context, config *azconfig.AZConfig) {
	if config.StorageAccountName != "" {
		logrus.WithField("storageAccountName", config.StorageAccountName).Info("storage account known, not creating new one")
		return
	}

	config.StorageAccountName = textio.ReadLine(ctx, "Desired storage account name")
	if config.StorageAccountName == "" {
		logrus.Fatal("no storage account name given, aborting")
	}

	account, ok := CreateAccount(ctx, *config)
	if !ok {
		logrus.Fatal("unable to create storage account")
	}

	config.StorageAccountName = *account.Name
	logrus.WithField("storageAccountName", config.StorageAccountName).Info("storage account created")
	config.Save()
}

// CreateAccount creates a new azure storage account
func CreateAccount(ctx context.Context, config azconfig.AZConfig) (storage.Account, bool) {
	accountClient := storage.NewAccountsClient(config.SubscriptionID)
	accountClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	// accountClient.RequestInspector = azdebug.LogRequest()
	// accountClient.ResponseInspector = azdebug.LogResponse()

	logger := logrus.WithFields(logrus.Fields{
		"storageAccountName": config.StorageAccountName,
		"resourceGroup":      config.ResourceGroup,
		"location":           config.Location,
	})
	logger.Info("checking storage account name availability")

	result, err := accountClient.CheckNameAvailability(
		ctx,
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(config.StorageAccountName),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
		})
	if err != nil {
		logger.WithError(err).Error("storage account check-name-availability failed")
		return storage.Account{}, false
	}

	if *result.NameAvailable != true {
		logger.WithFields(logrus.Fields{
			"serverMessage": *result.Message,
			"reason":        result.Reason,
		}).Error("storage account name not available")
		return storage.Account{}, false
	}

	logger.Info("creating storage account")

	future, err := accountClient.Create(
		ctx,
		config.ResourceGroup,
		config.StorageAccountName,
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
