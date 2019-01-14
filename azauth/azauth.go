package azauth

import (
	log "github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Load authorisation details from azure.PublicCloud.XXXManagementEndpoint URLs
func Load(url string) autorest.Authorizer {
	log.Info("loading auth file from AZURE_AUTH_LOCATION")
	authorizer, err := auth.NewAuthorizerFromFileWithResource(url)
	if err != nil {
		log.WithFields(log.Fields{
			log.ErrorKey: err,
			"url":        url,
			"env_var":    "AZURE_AUTH_LOCATION",
		}).Fatal("unable to load authorization file")
	}
	return authorizer
}
