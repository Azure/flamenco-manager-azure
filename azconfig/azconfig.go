package azconfig

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
)

const (
	configFile = "azure_config.json"
)

// AZConfig is loaded from azure_config.json
type AZConfig struct {
	// ID of the Azure subscription. It is the "id" field shown by `az account list`
	SubscriptionID string `json:"subscriptionID"`
	// Name of the resource group that will contain the Flamenco infrastructure.
	ResourceGroup string `json:"resourceGroup"`
	// Name of the Azure Batch account that will contain the Flamenco Worker VM pool.
	BatchAccountName string `json:"batchAccountName"`
	// Name of the Azure Storage account that will contain the Flamenco files.
	StorageAccountName string `json:"storageAccountName"`
	// Physical location of the resource group, such as 'westeurope' or 'eastus'.
	Location string `json:"location"`
}

// Load returns the config file, or hard-exits the process if it cannot be loaded.
func Load() AZConfig {
	logger := log.WithField("filename", configFile)
	paramFile, err := os.Open(configFile)
	if err != nil {
		logger.WithError(err).Fatal("unable to open Azure Batch pool parameters")
	}
	defer paramFile.Close()

	params := AZConfig{}
	decoder := json.NewDecoder(paramFile)
	if err := decoder.Decode(&params); err != nil {
		logger.WithError(err).Fatal("unable to decode Azure Batch pool parameters")
	}

	if params.ResourceGroup == "" {
		logger.Fatal("property 'resourceGroup' must be set")
	}
	if params.BatchAccountName == "" {
		logger.Fatal("property 'batchAccountName' must be set")
	}
	if params.StorageAccountName == "" {
		logger.Fatal("property 'storageAccountName' must be set")
	}
	if params.Location == "" {
		logger.Fatal("property 'location' must be set")
	}
	return params
}
