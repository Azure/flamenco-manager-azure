package azstorage

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-file-go/azfile"
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/flamenco"
)

const (
	defaultQuotaInGB int32 = 5 * 1024 // SMB share quota, in gigabytes

	statusShareAlreadyExists = "ShareAlreadyExists"
)

var (
	defaultSMBShares = []string{
		"flamenco-resources",
		"flamenco-input",
		"flamenco-output",
	}
)

// EnsureFileShares sets up the SMB shares. Returns fstab lines to mount them.
func EnsureFileShares(ctx context.Context, config azconfig.AZConfig) string {
	storageCreds := GetCredentials(ctx, config)
	logrus.WithFields(logrus.Fields{
		"username": storageCreds.Username,
		"password": storageCreds.Password,
	}).Info("obtained storage credentials")

	fstab := []string{}
	shareURL := getShareURL(config, storageCreds)
	for _, shareName := range defaultSMBShares {
		createFileShare(ctx, shareURL, shareName)

		fstabLine := fmt.Sprintf(
			"//%s.file.core.windows.net/%s /mnt/%s cifs vers=3.0,username=%s,password=%s,dir_mode=0770,file_mode=0660,gid=%s,forcegid,sec=ntlmssp,mfsymlinks 0 0",
			config.StorageAccountName,
			shareName, shareName,
			storageCreds.Username, storageCreds.Password,
			flamenco.UnixGroupName,
		)

		fstab = append(fstab, fstabLine)
	}
	return strings.Join(fstab, "\n") + "\n"
}

func getShareURL(config azconfig.AZConfig, storageCreds Credentials) azfile.ServiceURL {
	logger := logrus.WithFields(logrus.Fields{
		"storageAccount": config.StorageAccountName,
	})

	// Use your Storage account's name and key to create a credential object; this is used to access your account.
	credential, err := azfile.NewSharedKeyCredential(storageCreds.Username, storageCreds.Password)
	if err != nil {
		logger.WithError(err).Fatal("unable to construct credentials for Azure Files")
	}

	// Create a request pipeline that is used to process HTTP(S) requests and responses. It requires
	// your account credentials. In more advanced scenarios, you can configure telemetry, retry policies,
	// logging, and other options. Also, you can configure multiple request pipelines for different scenarios.
	pipeline := azfile.NewPipeline(credential, azfile.PipelineOptions{})

	// From the Azure portal, get your Storage account file service URL endpoint.
	// The URL typically looks like this:
	topURL, err := url.Parse(fmt.Sprintf("https://%s.file.core.windows.net", config.StorageAccountName))
	if err != nil {
		logger.WithError(err).Fatal("unable to construct URL")
	}

	// Create an ServiceURL object that wraps the service URL and a request pipeline.
	return azfile.NewServiceURL(*topURL, pipeline)
}

// createFileShare creates an SMB file share.
func createFileShare(ctx context.Context, serviceURL azfile.ServiceURL, shareName string) {
	shareName = strings.ToLower(shareName)
	logger := logrus.WithFields(logrus.Fields{
		"shareName": shareName,
	})

	logger.Info("ensuring SMB share exists")
	shareURL := serviceURL.NewShareURL(shareName)

	_, err := shareURL.Create(ctx, azfile.Metadata{}, defaultQuotaInGB)
	if err != nil {
		storageErr, ok := err.(azfile.StorageError)
		if !ok {
			logger.WithError(err).Fatalf("unable to create SMB share")
		}
		if storageErr.ServiceCode() == statusShareAlreadyExists {
			logger.Debug("SMB share already exists")
			return
		}
		logger.WithFields(logrus.Fields{
			"serviceCode":   storageErr.ServiceCode(),
			logrus.ErrorKey: storageErr,
		}).Fatal("unable to create SMB share")
	}
	logger.Info("SMB share created")
}
