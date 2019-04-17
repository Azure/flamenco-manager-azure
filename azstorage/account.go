package azstorage

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azauth"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/aznetwork"
	"gitlab.com/blender-institute/azure-go-test/textio"
)

func getAccountClient(config azconfig.AZConfig) storage.AccountsClient {
	accountClient := storage.NewAccountsClient(config.SubscriptionID)
	accountClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	// accountClient.RequestInspector = azdebug.LogRequest()
	// accountClient.ResponseInspector = azdebug.LogResponse()
	return accountClient
}

// AskAccountName asks for a storage account name, potentially overridable by a CLI arg.
func AskAccountName(ctx context.Context, config azconfig.AZConfig, cliAccountName string) (desiredName string, mustCreate bool) {
	if cliAccountName != "" {
		logrus.WithField("storageAccountName", cliAccountName).Debug("creating storage account from CLI")
		return cliAccountName, true
	}

	if config.StorageAccountName != "" {
		logrus.WithField("storageAccountName", config.StorageAccountName).Info("storage account known, not creating new one")
		return config.StorageAccountName, false
	}

	desiredName = textio.ReadLine(ctx, "Desired storage account name")
	if desiredName == "" {
		logrus.Fatal("no storage account name given, aborting")
	}

	return desiredName, true
}

// CreateAndSave creates a storage account and stores it in the config.
func CreateAndSave(ctx context.Context, config *azconfig.AZConfig, accountName string, netStack aznetwork.NetworkStack) {
	account, ok := CreateAccount(ctx, *config, netStack, accountName)
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

	if *result.NameAvailable != true {
		logger.WithFields(logrus.Fields{
			"serverMessage": *result.Message,
			"reason":        result.Reason,
		}).Error("storage account name not available")
		return
	}

	return true
}

// CreateAccount creates a new azure storage account
func CreateAccount(ctx context.Context, config azconfig.AZConfig,
	netStack aznetwork.NetworkStack, accountName string,
) (storage.Account, bool) {
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
