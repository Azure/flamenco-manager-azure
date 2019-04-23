package azconfig

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const (
	configFile = "azure_config.yaml"
)

// AZBatchConfig has all the batch parameters.
type AZBatchConfig struct {
	PoolID string `yaml:"poolID"` // name of the batch pool
	VMSize string `yaml:"vmSize"` // machine size, like "Standard_F16s"

	TargetDedicatedNodes   int32 `yaml:"targetDedicatedNodes"`
	TargetLowPriorityNodes int32 `yaml:"targetLowPriorityNodes"`
}

// AZConfig is loaded from azure_config.yaml
type AZConfig struct {
	// File this config was read from, so it can be saved after modification.
	filename string

	// ID of the Azure subscription. It is the "id" field shown by `az account list`
	SubscriptionID string ` yaml:"subscriptionID"`
	// Physical location of the resource group, such as 'westeurope' or 'eastus'.
	Location string ` yaml:"location"`

	// Name of the resource group that will contain the Flamenco infrastructure.
	ResourceGroup string `yaml:"resourceGroup,omitempty"`
	// Name of the Azure Batch account that will contain the Flamenco Worker VM pool.
	BatchAccountName string `yaml:"batchAccountName,omitempty"`
	// Name of the Azure Storage account that will contain the Flamenco files.
	StorageAccountName string `yaml:"storageAccountName,omitempty"`
	// Name of the Virtual Machine that's going to run Flamenco Manager.
	VMName string `yaml:"virtualMachine,omitempty"`
	// Worker registration secret; shouldn't change, as we don't overwrite the Manager config if it already exists on the VM.
	WorkerRegistrationSecret string `yaml:"workerRegistrationSecret,omitempty"`

	// this is set by main.go after creating the storage account.
	StorageCreds StorageCredentials `yaml:"-"`

	Batch *AZBatchConfig `yaml:"batch,omitempty"`
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

	if params.WorkerRegistrationSecret == "" {
		logger.Info("generating random worker secret")
		params.WorkerRegistrationSecret = randomWorkerSecret()
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

// DomainName returns the expected public domain name of the Public IP.
func (azc AZConfig) DomainName() string {
	if azc.VMName == "" {
		logrus.Panic("virtual machine name is empty, unable to construct domain name")
	}
	if azc.Location == "" {
		logrus.Panic("location is empty, unable to construct domain name")
	}
	return fmt.Sprintf("%s.%s.cloudapp.azure.com",
		azc.VMName,
		azc.Location,
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

func randomWorkerSecret() string {
	randomBytes := make([]byte, 64)
	if _, err := rand.Read(randomBytes); err != nil {
		logrus.WithError(err).Fatal("error reading random bytes")
	}
	secret := strings.Trim(base64.URLEncoding.EncodeToString(randomBytes), "=")
	return secret
}
