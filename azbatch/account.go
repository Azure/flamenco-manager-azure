/* (c) 2019, Blender Foundation
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package azbatch

import (
	"context"

	batchARM "github.com/Azure/azure-sdk-for-go/services/batch/mgmt/2017-09-01/batch"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"github.com/Azure/flamenco-manager-azure/azauth"
	"github.com/Azure/flamenco-manager-azure/azconfig"
	"github.com/Azure/flamenco-manager-azure/textio"
)

func getBatchAccountClient(config azconfig.AZConfig) batchARM.AccountClient {
	accountClient := batchARM.NewAccountClient(config.SubscriptionID)
	accountClient.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	// accountClient.RequestInspector = azdebug.LogRequest()
	// accountClient.ResponseInspector = azdebug.LogResponse()
	return accountClient
}

// AskAccountName asks for a batch account name, potentially overridable by a CLI arg.
func AskAccountName(
	ctx context.Context, config azconfig.AZConfig,
	cliAccountName, defaultAccountName string,
) (desiredName string, mustCreate bool) {
	if cliAccountName != "" {
		logrus.WithField("batchAccountName", cliAccountName).Debug("creating batch account from CLI")
		return cliAccountName, true
	}

	if config.BatchAccountName != "" {
		logrus.WithField("batchAccountName", config.BatchAccountName).Info("batch account known, not creating new one")
		return config.BatchAccountName, false
	}

	desiredName = textio.ReadLineWithDefault(ctx, "Desired batch account name", defaultAccountName)
	if desiredName == "" {
		logrus.Fatal("no batch account name given, aborting")
	}

	return desiredName, true
}

// CreateAndSave creates a batch account and saves it to the config.
func CreateAndSave(ctx context.Context, config *azconfig.AZConfig, accountName string) {
	account, ok := CreateAccount(ctx, *config, accountName)
	if !ok {
		logrus.Fatal("unable to create batch account")
	}

	config.BatchAccountName = *account.Name
	logrus.WithField("batchAccountName", config.BatchAccountName).Info("batch account created")
	config.Save()
}

// CreateAccount creates a new azure batch account
func CreateAccount(ctx context.Context, config azconfig.AZConfig, accountName string) (batchARM.Account, bool) {
	accountClient := getBatchAccountClient(config)

	logger := logrus.WithFields(logrus.Fields{
		"batchAccountName": accountName,
		"resourceGroup":    config.ResourceGroup,
		"location":         config.Location,
	})
	logger.Info("creating batch account")

	params := batchARM.AccountCreateParameters{
		Location: to.StringPtr(config.Location),
		AccountCreateProperties: &batchARM.AccountCreateProperties{},
	}
	res, err := accountClient.Create(ctx, config.ResourceGroup, accountName, params)
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
