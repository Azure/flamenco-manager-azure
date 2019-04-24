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
