package azconfig

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const (
	configFile = "azure_config.yaml"
)

// AZConfig is loaded from azure_config.yaml
type AZConfig struct {
	// File this config was read from, so it can be saved after modification.
	filename string

	// ID of the Azure subscription. It is the "id" field shown by `az account list`
	SubscriptionID string `json:"subscriptionID" yaml:"subscriptionID"`
	// Physical location of the resource group, such as 'westeurope' or 'eastus'.
	Location string `json:"location" yaml:"location"`

	// Name of the resource group that will contain the Flamenco infrastructure.
	ResourceGroup string `json:"resourceGroup,omitempty" yaml:"resourceGroup,omitempty"`
	// Name of the Azure Batch account that will contain the Flamenco Worker VM pool.
	BatchAccountName string `json:"batchAccountName,omitempty" yaml:"batchAccountName,omitempty"`
	// Name of the Azure Storage account that will contain the Flamenco files.
	StorageAccountName string `json:"storageAccountName,omitempty" yaml:"storageAccountName,omitempty"`
}

// Load returns the config file, or hard-exits the process if it cannot be loaded.
func Load() AZConfig {
	logger := logrus.WithField("filename", configFile)
	paramFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		logger.WithError(err).Fatal("unable to open Azure Batch pool parameters")
	}

	abspath, err := filepath.Abs(configFile)
	if err != nil {
		logger.WithError(err).Fatal("unable to construct absolute path")
	}

	params := AZConfig{}
	params.filename = abspath
	if err := yaml.Unmarshal(paramFile, &params); err != nil {
		logger.WithError(err).Fatal("unable to decode Azure Batch pool parameters")
	}

	if params.SubscriptionID == "" {
		logger.Fatal("property 'subscriptionID' must be set")
	}
	if params.Location == "" {
		logger.Fatal("property 'location' must be set")
	}
	return params
}

// StorageAccountID computes the storage account ID given the other properties.
func (azc AZConfig) StorageAccountID() string {
	return fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s",
		azc.SubscriptionID,
		azc.ResourceGroup,
		azc.StorageAccountName,
	)
}

// Save stores the config as YAML.
func (azc AZConfig) Save() {
	logger := logrus.WithField("filename", azc.filename)
	if azc.filename == "" {
		logger.Fatal("unable to save config file, filename unknown")
	}
	logger.Debug("saving configuration")

	fileContents, err := yaml.Marshal(azc)
	if err != nil {
		logger.WithError(err).Fatal("unable to construct configuration file")
	}

	tmpname := azc.filename + "~"
	if err := ioutil.WriteFile(tmpname, fileContents, 0666); err != nil {
		logger.WithFields(logrus.Fields{
			logrus.ErrorKey: err,
			"writingTo":     tmpname,
		}).Fatal("unable to save configuration file")
	}

	if err := os.Remove(azc.filename); err != nil {
		logger.WithError(err).Fatal("unable to delete old config file")
	}
	if err := os.Rename(tmpname, azc.filename); err != nil {
		logrus.WithFields(logrus.Fields{
			logrus.ErrorKey: err,
			"renameFrom":    tmpname,
			"renameTo":      azc.filename,
		}).Fatal("unable to rename configuration file")
	}
}
