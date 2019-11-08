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
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/Azure/flamenco-manager-azure/azconfig"
	"github.com/Azure/flamenco-manager-azure/textio"
)

// AskSubscriptionAndSave asks the user for the subscription ID and saves it in the config.
func AskSubscriptionAndSave(ctx context.Context, config *azconfig.AZConfig, subscriptionID string) {
	if subscriptionID != "" {
		logrus.WithField("subscriptionID", subscriptionID).Info("taking subscription ID from CLI arguments")
		config.SubscriptionID = subscriptionID
		config.Save()
		return
	}

	if config.SubscriptionID != "" {
		logrus.WithField("subscriptionID", config.SubscriptionID).Info("taking subscription ID from config file")
		return
	}

	available := ListSubscriptions(ctx)
	switch len(available) {
	case 0:
		logrus.Fatal("Your account does not have any subscription, visit the Azure website and create one.")
	case 1:
		config.SubscriptionID = *available[0].SubscriptionID
		logrus.WithField("subscriptionID", config.SubscriptionID).Info("using your Azure subscription")
	default:
		logrus.WithField("subscriptionCount", len(available)).Info("multiple Azure subscriptions found")

		fmt.Println("Available subscriptions:")
		for idx, subs := range available {
			fmt.Printf("    %2d: %s\n", idx+1, *subs.DisplayName)
		}
		choice := textio.ReadNonNegativeInt(ctx, "Azure Subscription number", false)
		if choice < 1 || choice > len(available) {
			logrus.WithField("index", choice).Fatal("that subscription is not available")
		}
		config.SubscriptionID = *available[choice-1].SubscriptionID
		logrus.WithField("subscriptionID", config.SubscriptionID).Info("using Azure subscription")
	}
	config.Save()
}

// AskLocationAndSave asks the user for the Azure Location and saves it in the config.
func AskLocationAndSave(ctx context.Context, config *azconfig.AZConfig, location string) {
	if location != "" {
		logrus.WithField("location", location).Info("taking Azure Location from CLI arguments")
		config.Location = location
		config.Save()
		return
	}

	if config.Location != "" {
		logrus.WithField("location", config.Location).Info("taking Azure Location from config file")
		return
	}

	available := ListLocations(ctx, config.SubscriptionID)
	switch len(available) {
	case 0:
		logrus.Fatal("Your account does not have any locations available.")
	case 1:
		config.Location = *available[0].Name
		logrus.WithField("location", config.Location).Info("using the only available location")
	default:
		logrus.WithField("locationCount", len(available)).Info("multiple Azure locations available")

		fmt.Println("Available locations:")
		for idx, subs := range available {
			fmt.Printf("    %2d: %s\n", idx+1, *subs.DisplayName)
		}
		choice := textio.ReadNonNegativeInt(ctx, "Azure Location number", false)
		if choice < 1 || choice > len(available) {
			logrus.WithField("index", choice).Fatal("that location is not available")
		}
		config.Location = *available[choice-1].Name
		logrus.WithField("location", config.Location).Info("using Azure Location")
	}
	config.Save()
}
