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

package azstorage

import (
	"context"

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/flamenco-deploy-azure/azconfig"
)

// GetCredentials obtains the credentials to mount shares from the storage account.
func GetCredentials(ctx context.Context, config *azconfig.AZConfig) {
	accountClient := getAccountClient(*config)
	logger := logrus.WithFields(logrus.Fields{
		"storageAccountName": config.StorageAccountName,
		"resourceGroup":      config.ResourceGroup,
		"location":           config.Location,
	})
	logger.Info("obtaining storage key")

	result, err := accountClient.ListKeys(ctx, config.ResourceGroup, config.StorageAccountName)
	if err != nil {
		logger.WithError(err).Fatal("unable to load storage keys")
	}
	if result.Keys == nil || len(*result.Keys) == 0 {
		logger.Fatal("this storage account has no access keys")
	}

	firstKey := (*result.Keys)[0]
	creds := azconfig.StorageCredentials{
		Username: config.StorageAccountName,
		Password: *firstKey.Value,
	}

	config.StorageCreds = creds
}
