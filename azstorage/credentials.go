package azstorage

import (
	"context"

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
)

// Credentials has everything you need to mount a file share from a storage account.
type Credentials struct {
	Username string // the storage account name
	Password string // the storage account key
}

// GetCredentials obtains the credentials to mount shares from the storage account.
func GetCredentials(ctx context.Context, config azconfig.AZConfig) Credentials {
	accountClient := getAccountClient(config)
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
	return Credentials{
		Username: config.StorageAccountName,
		Password: *firstKey.Value,
	}
}
