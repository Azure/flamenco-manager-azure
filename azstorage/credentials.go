package azstorage

import (
	"context"

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/flamenco-deploy-azure/azconfig"
)

// GetCredentials obtains the credentials to mount shares from the storage account.
func GetCredentials(ctx context.Context, config *azconfig.AZConfig) {
	accountClient := getAccountClient(*config)
	logger := logrus.WithFields(logrus.Fields{
		"storageAccountName": config.StorageAccountName,
		"resourceGroup":      config.ResourceGroup,
		"location":           config.Location,
	})
	logger.Info("obtaining storage key")

	result, err := accountClient.ListKeys(ctx, config.ResourceGroup, config.StorageAccountName)
	if err != nil {
		logger.WithError(err).Fatal("unable to load storage keys")
	}
	if result.Keys == nil || len(*result.Keys) == 0 {
		logger.Fatal("this storage account has no access keys")
	}

	firstKey := (*result.Keys)[0]
	creds := azconfig.StorageCredentials{
		Username: config.StorageAccountName,
		Password: *firstKey.Value,
	}

	config.StorageCreds = creds
}
