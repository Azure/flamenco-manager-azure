package azbatch

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/batch/2018-12-01.8.0/batch"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"gitlab.com/blender-institute/azure-go-test/azauth"

	log "github.com/sirupsen/logrus"
)

const (
	batchURL               = "https://flamenco.westeurope.batch.azure.com"
	batchResourceGroupName = "cloud_01"
	batchAccountName       = "flamenco"
	batchPoolName          = "flamenco-workers"
)

// Connect to the Azure Batch service.
func Connect() {

	// log.Debug("creating batch account client")
	// accountClient := batch.NewAccountClient(batchURL)
	// accountClient.Authorizer = authorizer
	// accountClient.AddToUserAgent("je-moeder")
	// accountClient.RequestInspector = azdebug.LogRequest()
	// accountClient.ResponseInspector = azdebug.LogResponse()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
	defer cancel()

	createPoolIfNotExist(ctx, "je-moeder-47")
}

func createPoolIfNotExist(ctx context.Context, poolID string) {
	poolClient := batch.NewPoolClient(batchURL)
	poolClient.Authorizer = azauth.Load(azure.PublicCloud.BatchManagementEndpoint)

	// poolClient.RequestInspector = azdebug.LogRequest()
	// poolClient.ResponseInspector = azdebug.LogResponse()

	filter := fmt.Sprintf("id eq '%s'", poolID)
	log.WithField("filter", filter).Debug("listing all pools")

	poolExists := false

	resultPage, err := poolClient.List(ctx, filter, "", "", nil, nil, nil, nil, nil)
	if err != nil {
		log.WithError(err).Fatal("unable to list existing pools")
	}
	for resultPage.NotDone() {
		for _, poolParams := range resultPage.Values() {
			log.WithField("ID", *poolParams.ID).Info("found Azure Batch pool")
			poolExists = poolExists || (*poolParams.ID == poolID)
		}
		err := resultPage.NextWithContext(ctx)
		if err != nil {
			log.WithError(err).Fatal("unable to get next page of pools")
		}
	}
	log.Info("done listing pools")

	if poolExists {
		log.WithField("pool_id", poolID).Debug("Azure Batch pool exists")
		return
	}

	createPool(ctx, poolID, poolClient)
}

func createPool(ctx context.Context, poolID string, poolClient batch.PoolClient) {
	logger := log.WithField("pool_id", poolID)

	toCreate := batch.PoolAddParameter{
		ID: &poolID,
		VirtualMachineConfiguration: &batch.VirtualMachineConfiguration{
			ImageReference: &batch.ImageReference{
				Publisher: to.StringPtr("Canonical"),
				Sku:       to.StringPtr("18.04-LTS"),
				Offer:     to.StringPtr("UbuntuServer"),
				Version:   to.StringPtr("latest"),
			},
			NodeAgentSKUID: to.StringPtr("batch.node.ubuntu 18.04"),
		},
		MaxTasksPerNode:      to.Int32Ptr(1),
		TargetDedicatedNodes: to.Int32Ptr(1),

		// Create a startup task to run a script on each pool machine
		StartTask: &batch.StartTask{
			ResourceFiles: &[]batch.ResourceFile{
				{
					HTTPURL:  to.StringPtr("https://raw.githubusercontent.com/lawrencegripper/azure-sdk-for-go-samples/1441a1dc4a6f7e47c4f6d8b537cf77ce4f7c452c/batch/examplestartup.sh"),
					FilePath: to.StringPtr("echohello.sh"),
					FileMode: to.StringPtr("777"),
				},
			},
			CommandLine:    to.StringPtr("bash -f echohello.sh"),
			WaitForSuccess: to.BoolPtr(true),
			UserIdentity: &batch.UserIdentity{
				AutoUser: &batch.AutoUserSpecification{
					ElevationLevel: batch.Admin,
					Scope:          batch.Task,
				},
			},
		},
		VMSize: to.StringPtr("standard_a1"),
	}

	_, err := poolClient.Add(ctx, toCreate, nil, nil, nil, nil)
	if err != nil {
		logger.WithError(err).Fatal("unable to add Azure Batch pool")
	}
	logger.Info("created Azure Batch pool")
}
