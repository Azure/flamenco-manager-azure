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

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azbatch"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/azresource"
	"gitlab.com/blender-institute/azure-go-test/azssh"
	"gitlab.com/blender-institute/azure-go-test/azstorage"
	"gitlab.com/blender-institute/azure-go-test/azvm"
)

const applicationName = "Azure Go Test"

var applicationVersion = "1.0"

// Components that make up the application

var cliArgs struct {
	version        bool
	quiet          bool
	debug          bool
	showStartupCLI bool

	resourceGroup  string
	storageAccount string
	batchAccount   string
	vmName         string
}

func parseCliArgs() {
	flag.BoolVar(&cliArgs.version, "version", false, "Shows the application version, then exits.")
	flag.BoolVar(&cliArgs.quiet, "quiet", false, "Disable info-level logging (so warning/error only).")
	flag.BoolVar(&cliArgs.debug, "debug", false, "Enable debug-level logging.")
	flag.BoolVar(&cliArgs.showStartupCLI, "startupCLI", false, "Just show the startup task CLI, do not start the pool.")
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
	if cliArgs.showStartupCLI {
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(1*time.Minute))
		defer cancel()

		poolParams := azbatch.PoolParameters()
		withCreds := azstorage.ReplaceAccountDetails(ctx, config, poolParams)
		fmt.Println(*withCreds.StartTask.CommandLine)
		logrus.Info("shutting down after logging account storage key stuff")
		return
	}

	sshContext := azssh.LoadSSHContext()

	// Determine what to create and what to assume is there.
	// sa = Storage Account; ba = Batch Account
	rgName, createRG := azresource.AskResourceGroupName(ctx, config, cliArgs.resourceGroup)
	saName, createSA := azstorage.AskAccountName(ctx, config, cliArgs.storageAccount)
	if createSA && !azstorage.CheckAvailability(ctx, config, saName) {
		logrus.WithField("storageAccountName", saName).Fatal("storage account name is not available")
	}
	baName, createBA := azbatch.AskAccountName(ctx, config, cliArgs.storageAccount)
	vmName, vmExists := azvm.ChooseVM(ctx, &config, cliArgs.vmName)

	// Create & update stuff.
	if createRG {
		azresource.EnsureResourceGroup(ctx, &config, rgName)
	}
	vm, networkStack := azvm.EnsureVM(ctx, config, vmName, vmExists)
	publicIP := *networkStack.PublicIP.IPAddress
	logrus.WithFields(logrus.Fields{
		"vmName":         *vm.Name,
		"publicAddress":  publicIP,
		"fqdn":           *networkStack.PublicIP.DNSSettings.Fqdn,
		"privateAddress": networkStack.PrivateIP,
		"vnet":           *networkStack.VNet.Name,
	}).Info("found network info")
	azvm.WaitForReady(ctx, config, vmName)

	if createSA {
		azstorage.CreateAndSave(ctx, &config, saName)
	}
	if createBA {
		azbatch.CreateAndSave(ctx, &config, baName)
	}

	fstab := azstorage.EnsureFileShares(ctx, config)

	// Set up the VM via an SSH connection
	ssh := azssh.Connect(sshContext, publicIP)
	ssh.SetupUsers()
	ssh.Close()

	// Reconnect to ensure the admin user is part of the flamenco group.
	ssh = azssh.Connect(sshContext, publicIP)
	ssh.UploadAsFile([]byte(fstab), "fstab-smb")
	ssh.RunInstallScript()
	ssh.Close()

	// azbatch.CreatePool(config)

	cancelCtx()
}
