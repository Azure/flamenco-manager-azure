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
		fileloc = CredentialsFile
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
