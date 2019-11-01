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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Azure/flamenco-manager-azure/azauth"
	"github.com/Azure/flamenco-manager-azure/azbatch"
	"github.com/Azure/flamenco-manager-azure/azconfig"
	"github.com/Azure/flamenco-manager-azure/azresource"
	"github.com/Azure/flamenco-manager-azure/azssh"
	"github.com/Azure/flamenco-manager-azure/azstorage"
	"github.com/Azure/flamenco-manager-azure/azsubscription"
	"github.com/Azure/flamenco-manager-azure/azvm"
	"github.com/Azure/flamenco-manager-azure/flamenco"
	"github.com/Azure/flamenco-manager-azure/textio"
	"github.com/sirupsen/logrus"
)

const applicationName = "Azure Go Test"

var applicationVersion = "1.0"

// Components that make up the application

var cliArgs struct {
	version bool
	quiet   bool
	debug   bool

	subscriptionID string
	location       string
	resourceGroup  string
	storageAccount string
	batchAccount   string
	vmName         string
}

func parseCliArgs() {
	flag.BoolVar(&cliArgs.version, "version", false, "Shows the application version, then exits.")
	flag.BoolVar(&cliArgs.quiet, "quiet", false, "Disable info-level logging (so warning/error only).")
	flag.BoolVar(&cliArgs.debug, "debug", false, "Enable debug-level logging.")

	flag.StringVar(&cliArgs.subscriptionID, "subscription", "", "Subscription ID. If not given, it will be prompted for.")
	flag.StringVar(&cliArgs.location, "location", "", "Physical location of the Azure machines. If not given, it will be prompted for.")
	flag.StringVar(&cliArgs.resourceGroup, "group", "", "Name of the resource group. If not given, it will be prompted for.")
	flag.StringVar(&cliArgs.storageAccount, "sa", "", "Name of the storage account. If not given, it will be prompted for.")
	flag.StringVar(&cliArgs.batchAccount, "ba", "", "Name of the batch account. If not given, it will be prompted for.")
	flag.StringVar(&cliArgs.vmName, "vm", "", "Name of the virtual machine to use. If not given, it will be prompted for.")
	flag.Parse()
}

func configLogging() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the warning severity or above by default.
	level := logrus.InfoLevel
	if cliArgs.debug {
		level = logrus.DebugLevel
	} else if cliArgs.quiet {
		level = logrus.WarnLevel
	}
	logrus.SetLevel(level)
	log.SetOutput(logrus.StandardLogger().Writer())
}

func logStartup() {
	level := logrus.GetLevel()
	defer logrus.SetLevel(level)

	logrus.SetLevel(logrus.InfoLevel)
	logrus.WithFields(logrus.Fields{
		"version": applicationVersion,
	}).Infof("Starting %s", applicationName)
}

func main() {
	startupTime := time.Now()
	parseCliArgs()
	if cliArgs.version {
		fmt.Println(applicationVersion)
		return
	}

	configLogging()
	logStartup()

	ctx, cancelCtx := context.WithCancel(context.Background())

	// Handle Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for signum := range c {
			logrus.WithField("signal", signum).Info("Signal received, shutting down.")
			cancelCtx()
			time.Sleep(1 * time.Second)
			os.Exit(2)
		}
	}()

	config := azconfig.Load()
	sshContext := azssh.LoadSSHContext()

	// Get the Azure credentials into the right file.
	azauth.EnsureCredentialsFile(ctx)

	// Ask for stuff we can't create.
	azsubscription.AskSubscriptionAndSave(ctx, &config, cliArgs.subscriptionID)
	azsubscription.AskLocationAndSave(ctx, &config, cliArgs.location)

	// Ask for the default name for the subsequent prompts.
	if config.DefaultName == "" {
		config.DefaultName = textio.ReadLineWithDefault(ctx, "Default name for subcomponents", config.DefaultName)
		config.Save()
	}

	// Determine what to create and what to assume is there.
	// rg = Resource Group; sa = Storage Account; ba = Batch Account
	
	// Ask for Resource Group name. Keep prompting for a name until a valid name is provided
	for {
		rgName, createRG := azresource.AskResourceGroupName(ctx, config, cliArgs.resourceGroup, config.DefaultName)
		if createRG {
			ok := azresource.EnsureResourceGroup(ctx, &config, rgName)
			if ok {
				break
			}
		} else {
			break
		}	
	}
	azbatch.AskParametersAndSave(ctx, &config, config.DefaultName)

	vmName, vmExists := azvm.ChooseVM(ctx, &config, cliArgs.vmName, config.DefaultName)
	// Create or update Manager VM
	vm, networkStack := azvm.EnsureVM(ctx, config, vmName, vmExists)
	publicIP := *networkStack.PublicIP.IPAddress
	logrus.WithFields(logrus.Fields{
		"vmName":         *vm.Name,
		"publicAddress":  publicIP,
		"fqdn":           networkStack.FQDN(),
		"privateAddress": networkStack.PrivateIP,
		"vnet":           *networkStack.VNet.Name,
	}).Info("found network info")
	azvm.WaitForReady(ctx, config, vmName)

	saName, createSA := azstorage.AskAccountName(ctx, config, cliArgs.storageAccount, config.DefaultName)
	if createSA && !azstorage.CheckAvailability(ctx, config, saName) {
		logrus.WithField("storageAccountName", saName).Fatal("storage account name is not available")
	}
	if createSA {
		azstorage.CreateAndSave(ctx, &config, saName)
	}
	azstorage.GetCredentials(ctx, &config)

	baName, createBA := azbatch.AskAccountName(ctx, config, cliArgs.storageAccount, config.DefaultName)
	if createBA {
		azbatch.CreateAndSave(ctx, &config, baName)
	}

	// Collect dynamically generated files (or bits of files).
	fstab := azstorage.EnsureFileShares(ctx, config)
	tmpl := flamenco.NewTemplateContext(config, networkStack, fstab)
	flamanYAML := tmpl.RenderTemplate("flamenco-manager.yaml")
	flaworkCfg := tmpl.RenderTemplate("flamenco-worker.cfg")
	flaworkStart := tmpl.RenderTemplate("flamenco-worker-startup.sh")

	// Set up the VM via an SSH connection
	ssh := azssh.Connect(sshContext, publicIP)
	ssh.SetupUsers()
	ssh.Close()

	// Reconnect to ensure the admin user is part of the flamenco group.
	ssh = azssh.Connect(sshContext, publicIP)
	ssh.UploadAsFile([]byte(fstab), "fstab-smb")
	ssh.UploadStaticFile("flamenco-manager.service")
	ssh.UploadAsFile(flamanYAML, "default-flamenco-manager.yaml")
	ssh.UploadAsFile(flaworkCfg, "flamenco-worker.cfg")
	ssh.UploadAsFile(flaworkStart, "flamenco-worker-startup.sh")
	ssh.UploadStaticFile(flamenco.InstallScriptName)
	ssh.UploadLocalFile(azauth.CredentialsFile)
	ssh.RunInstallScript()
	ssh.Close()

	azbatch.CreatePool(config, networkStack)

	cancelCtx()

	duration := time.Since(startupTime)
	logrus.WithFields(logrus.Fields{
		"duration": duration,
		"url":      fmt.Sprintf("https://%s/setup", networkStack.FQDN()),
	}).Info("deployment complete")
}
