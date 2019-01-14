package azbatch

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/batch/2018-12-01.8.0/batch"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"gitlab.com/blender-institute/azure-go-test/azauth"

	log "github.com/sirupsen/logrus"
)

const (
	batchURL       = "https://flamenco.westeurope.batch.azure.com"
	batchParamFile = "azure_batch_pool.json"
)

// Connect to the Azure Batch service.
func Connect() {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
	defer cancel()

	poolParams := poolParameters()
	createPoolIfNotExist(ctx, poolParams)
}

func poolParameters() batch.PoolAddParameter {
	logger := log.WithField("filename", batchParamFile)
	paramFile, err := os.Open(batchParamFile)
	if err != nil {
		logger.WithError(err).Fatal("unable to open Azure Batch pool parameters")
	}
	defer paramFile.Close()

	params := batch.PoolAddParameter{}
	decoder := json.NewDecoder(paramFile)
	if err := decoder.Decode(&params); err != nil {
		logger.WithError(err).Fatal("unable to decode Azure Batch pool parameters")
	}

	if params.ID == nil {
		logger.Fatal("pool parameter 'id' must be set")
	}
	return params
}

func createPoolIfNotExist(ctx context.Context, poolParams batch.PoolAddParameter) {
	logger := log.WithField("pool_id", *poolParams.ID)

	poolClient := batch.NewPoolClient(batchURL)
	poolClient.Authorizer = azauth.Load(azure.PublicCloud.BatchManagementEndpoint)

	// poolClient.RequestInspector = azdebug.LogRequest()
	// poolClient.ResponseInspector = azdebug.LogResponse()

	log.Debug("fetching pools")
	poolExists := false
	resultPage, err := poolClient.List(ctx, "", "", "", nil, nil, nil, nil, nil)
	if err != nil {
		log.WithError(err).Fatal("unable to list existing pools")
	}

	for resultPage.NotDone() {
		for _, foundPool := range resultPage.Values() {
			log.WithField("found_id", *foundPool.ID).Info("found existing Azure Batch pool")
			poolExists = poolExists || (*foundPool.ID == *poolParams.ID)
		}
		err := resultPage.NextWithContext(ctx)
		if err != nil {
			log.WithError(err).Fatal("unable to get next page of pools")
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
