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

	azresource.EnsureResourceGroup(ctx, &config, cliArgs.resourceGroup)

	vmName, vmExists := azvm.ChooseVM(ctx, config, cliArgs.vmName)
	vm, networkStack := azvm.EnsureVM(ctx, config, vmName, vmExists)
	address := *networkStack.PublicIP.IPAddress
	logrus.WithFields(logrus.Fields{
		"vmName":  *vm.Name,
		"address": address,
		"vnet":    *networkStack.VNet.Name,
	}).Info("found network info")

	// The storage account needs to be limited to the VM's VLAN.

	azstorage.EnsureAccount(ctx, &config, cliArgs.storageAccount)
	azbatch.EnsureAccount(ctx, &config, cliArgs.batchAccount)

	// azbatch.CreatePool(config)

	cancelCtx()
}
