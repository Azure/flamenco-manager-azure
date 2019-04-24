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
	"fmt"
	"time"

	"gitlab.com/blender-institute/flamenco-deploy-azure/aznetwork"

	"gitlab.com/blender-institute/flamenco-deploy-azure/azconfig"

	"github.com/Azure/azure-sdk-for-go/services/batch/2018-12-01.8.0/batch"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"gitlab.com/blender-institute/flamenco-deploy-azure/azauth"

	"github.com/sirupsen/logrus"
)

func getPoolClient(batchURL string) batch.PoolClient {
	poolClient := batch.NewPoolClient(batchURL)
	poolClient.Authorizer = azauth.Load(azure.PublicCloud.BatchManagementEndpoint)
	// poolClient.RequestInspector = azdebug.LogRequest()
	// poolClient.ResponseInspector = azdebug.LogResponse()
	return poolClient
}

// CreatePool starts a pool of Flamenco Workers.
func CreatePool(config azconfig.AZConfig, netStack aznetwork.NetworkStack) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
	defer cancel()

	poolParams := PoolParameters(config, netStack)
	batchURL := constructBatchURL(config)
	createPoolIfNotExist(ctx, batchURL, poolParams)
}

func constructBatchURL(config azconfig.AZConfig) string {
	return fmt.Sprintf("https://%s.%s.batch.azure.com", config.BatchAccountName, config.Location)
}

func createPoolIfNotExist(ctx context.Context, batchURL string, poolParams batch.PoolAddParameter) {
	logger := logrus.WithField("pool_id", *poolParams.ID)
	logrus.Info("fetching batch pools")
	poolClient := getPoolClient(batchURL)

	poolExists := false
	resultPage, err := poolClient.List(ctx, "", "", "", nil, nil, nil, nil, nil)
	if err != nil {
		logrus.WithError(err).Fatal("unable to list existing pools")
	}

	for resultPage.NotDone() {
		for _, foundPool := range resultPage.Values() {
			logrus.WithField("found_id", *foundPool.ID).Info("found existing Azure Batch pool")
			poolExists = poolExists || (*foundPool.ID == *poolParams.ID)
		}
		err := resultPage.NextWithContext(ctx)
		if err != nil {
			logrus.WithError(err).Fatal("unable to get next page of pools")
		}
	}
	logger.WithField("pool_exists", poolExists).Debug("done listing pools")

	if poolExists {
		logger.Debug("Azure Batch pool exists")
		return
	}

	_, err = poolClient.Add(ctx, poolParams, nil, nil, nil, &date.TimeRFC1123{Time: time.Now()})
	if err != nil {
		logger.WithError(err).Fatal("unable to add Azure Batch pool")
	}
	logger.Info("created Azure Batch pool")
}
