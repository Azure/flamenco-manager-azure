package azstorage

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/batch/2018-12-01.8.0/batch"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"gitlab.com/blender-institute/azure-go-test/azauth"
	"gitlab.com/blender-institute/azure-go-test/azconfig"

	log "github.com/sirupsen/logrus"
)

func client(config azconfig.AZConfig) storage.AccountsClient {
	accountsClient := storage.NewAccountsClient(config.SubscriptionID)
	return accountsClient
}

// ReplaceAccountDetails performs replacement of STORAGE_ACCOUNT and STORAGE_KEY variables.
// The replacement is only done in the startup task command line.
func ReplaceAccountDetails(ctx context.Context, config azconfig.AZConfig, poolParams batch.PoolAddParameter) batch.PoolAddParameter {
	var startupCLI string

	accountName := config.StorageAccountName
	logger := log.WithFields(log.Fields{
		"storageAccount": accountName,
		"resourceGroup":  config.ResourceGroup,
	})
	logger.Info("retrieving storage key")

	storageClient := client(config)
	storageClient.Authorizer = azauth.Load(azure.PublicCloud.ServiceManagementEndpoint)

	result, err := storageClient.ListKeys(ctx, config.ResourceGroup, accountName)
	if err != nil {
		logger.WithError(err).Fatal("unable to obtain storage account keys")
	}
	if result.Keys == nil {
		logger.Fatal("nil storage account keys received")
	}
	switch keyCount := len(*result.Keys); keyCount {
	case 0:
		logger.Fatal("zero storage account keys received")
	case 1:
		logger.Debug("found one storage account key")
	default:
		logger.WithField("keyCount", keyCount).Warning("multiple storage keys found, using the first")
	}

	storageKey := (*result.Keys)[0]
	logger.WithFields(log.Fields{
		"name":        *storageKey.KeyName,
		"permissions": storageKey.Permissions,
	}).Debug("found storage account key")
	if storageKey.Value == nil || *storageKey.Value == "" {
		logger.WithField("keyName", *storageKey.KeyName).Fatal("storage key has no value")
	}

	startupCLI = *poolParams.StartTask.CommandLine
	startupCLI = strings.Replace(startupCLI, "{STORAGE_ACCOUNT}", accountName, -1)
	startupCLI = strings.Replace(startupCLI, "{STORAGE_KEY}", *storageKey.Value, -1)

	poolParams.StartTask.CommandLine = &startupCLI

	return poolParams
}
