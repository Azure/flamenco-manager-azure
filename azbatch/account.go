package azbatch

import (
	"context"

	batchARM "github.com/Azure/azure-sdk-for-go/services/batch/mgmt/2017-09-01/batch"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azauth"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/textio"
)

// EnsureAccount creates a batch account if config.BatchAccountName is "".
// The program is aborted when creation is required but fails.
func EnsureAccount(ctx context.Context, config *azconfig.AZConfig) {
	if config.BatchAccountName != "" {
		logrus.WithField("batchAccountName", config.BatchAccountName).Info("batch account known, not creating new one")
		return
	}

	config.BatchAccountName = textio.ReadLine(ctx, "Desired batch account name")
	if config.BatchAccountName == "" {
		logrus.Fatal("no batch account name given, aborting")
	}

	account, ok := CreateAccount(ctx, *config)
	if !ok {
		logrus.Fatal("unable to create batch account")
	}

	config.BatchAccountName = *account.Name
	logrus.WithField("batchAccountName", config.BatchAccountName).Info("batch account created")
	config.Save()
}

// CreateAccount creates a new azure batch account
func CreateAccount(ctx context.Context, config azconfig.AZConfig) (batchARM.Account, bool) {
	accountClient := batchARM.NewAccountClient(config.SubscriptionID)
	accountClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	// accountClient.RequestInspector = azdebug.LogRequest()
	// accountClient.ResponseInspector = azdebug.LogResponse()

	logger := logrus.WithFields(logrus.Fields{
		"batchAccountName": config.BatchAccountName,
		"resourceGroup":    config.ResourceGroup,
		"location":         config.Location,
	})
	logger.Info("creating batch account")
	res, err := accountClient.Create(ctx, config.ResourceGroup, config.BatchAccountName, batchARM.AccountCreateParameters{
		Location: to.StringPtr(config.Location),
	})
	if err != nil {
		logger.WithError(err).Error("failed starting batch account creation")
		return batchARM.Account{}, false
	}

	err = res.WaitForCompletionRef(ctx, accountClient.Client)
	if err != nil {
		logger.WithError(err).Error("failed waiting for batch account creation")
		return batchARM.Account{}, false
	}

	account, err := res.Result(accountClient)
	if err != nil {
		logger.WithError(err).Error("failed retrieving batch account")
		return batchARM.Account{}, false
	}

	return account, true
}
