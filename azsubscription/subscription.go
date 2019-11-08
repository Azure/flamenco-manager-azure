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

package azsubscription

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"github.com/Azure/flamenco-manager-azure/azauth"
)

func getSubscriptionClient() subscriptions.Client {
	client := subscriptions.NewClient()
	client.Authorizer = azauth.Load(azure.PublicCloud.ResourceManagerEndpoint)
	return client
}

// ListSubscriptions returns a list of subscription IDs.
func ListSubscriptions(ctx context.Context) []subscriptions.Subscription {
	logrus.Info("fetching Azure subscriptions")

	client := getSubscriptionClient()
	iter, err := client.ListComplete(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("unable to list Azure subscriptions")
	}

	result := []subscriptions.Subscription{}
	for iter.NotDone() {
		subsInfo := iter.Value()
		result = append(result, subsInfo)

		if err := iter.NextWithContext(ctx); err != nil {
			logrus.WithError(err).Fatal("unable to iterate Azure subscriptions")
		}
	}

	return result
}
