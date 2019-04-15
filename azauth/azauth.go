package azauth

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Load authorisation details from azure.PublicCloud.XXXManagementEndpoint URLs
func Load(url string) autorest.Authorizer {
	fileloc := os.Getenv("AZURE_AUTH_LOCATION")
	if fileloc == "" {
		fileloc = "client_credentials.json"
		err := os.Setenv("AZURE_AUTH_LOCATION", fileloc)
		if err != nil {
			logrus.WithError(err).Fatal("unable to set AZURE_AUTH_LOCATION environment variable")
		}
	}

	logger := logrus.WithField("authFile", fileloc)
	logger.Debug("loading credentials file")

	authorizer, err := auth.NewAuthorizerFromFileWithResource(url)
	if err != nil {
		logger.WithFields(logrus.Fields{
			logrus.ErrorKey: err,
			"authURL":       url,
		}).Fatal("unable to load authorization file")
	}
	return authorizer
}
