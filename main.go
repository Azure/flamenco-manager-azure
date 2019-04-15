package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	stdlog "log"

	log "github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azbatch"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/azstorage"
)

const applicationName = "Azure Go Test"

var applicationVersion = "1.0"

// Components that make up the application

// Signalling channels
var shutdownComplete chan struct{}

var cliArgs struct {
	version        bool
	quiet          bool
	debug          bool
	showStartupCLI bool
}

func parseCliArgs() {
	flag.BoolVar(&cliArgs.version, "version", false, "Shows the application version, then exits.")
	flag.BoolVar(&cliArgs.quiet, "quiet", false, "Disable info-level logging (so warning/error only).")
	flag.BoolVar(&cliArgs.debug, "debug", false, "Enable debug-level logging.")
	flag.BoolVar(&cliArgs.showStartupCLI, "startupCLI", false, "Just show the startup task CLI, do not start the pool.")
	flag.Parse()
}

func configLogging() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the warning severity or above by default.
	level := log.InfoLevel
	if cliArgs.debug {
		level = log.DebugLevel
	} else if cliArgs.quiet {
		level = log.WarnLevel
	}
	log.SetLevel(level)
	stdlog.SetOutput(log.StandardLogger().Writer())
}

func logStartup() {
	level := log.GetLevel()
	defer log.SetLevel(level)

	log.SetLevel(log.InfoLevel)
	log.WithFields(log.Fields{
		"version": applicationVersion,
	}).Infof("Starting %s", applicationName)
}

func shutdown(signum os.Signal) {
	timeout := make(chan bool)

	go func() {
		log.WithField("signal", signum).Info("Signal received, shutting down.")
		timeout <- false
	}()

	select {
	case <-timeout:
		log.Warning("Shutdown complete, stopping process.")
		close(shutdownComplete)
	case <-time.After(5 * time.Second):
		log.Error("Shutdown forced, stopping process.")
		os.Exit(-2)
	}

}

func main() {
	parseCliArgs()
	if cliArgs.version {
		fmt.Println(applicationVersion)
		return
	}

	configLogging()
	logStartup()

	shutdownComplete = make(chan struct{})

	// Handle Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for signum := range c {
			// Run the shutdown sequence in a goroutine, so that multiple Ctrl+C presses can be handled in parallel.
			go shutdown(signum)
		}
	}()

	config := azconfig.Load()
	if cliArgs.showStartupCLI {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
		defer cancel()

		poolParams := azbatch.PoolParameters()
		withCreds := azstorage.ReplaceAccountDetails(ctx, config, poolParams)
		fmt.Println(*withCreds.StartTask.CommandLine)
		log.Info("shutting down after logging account storage key stuff")
		return
	}

	azbatch.CreatePool(config)

	go shutdown(os.Interrupt)
	log.Info("Waiting for shutdown to complete.")
	<-shutdownComplete
}
