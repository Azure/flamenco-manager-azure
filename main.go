package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	stdlog "log"

	log "github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azbatch"
)

const applicationName = "Azure Go Test"
const applicationVersion = "1.0"

// Components that make up the application

// Signalling channels
var shutdownComplete chan struct{}

var cliArgs struct {
	version bool
	verbose bool
	debug   bool
}

func parseCliArgs() {
	flag.BoolVar(&cliArgs.version, "version", false, "Shows the application version, then exits.")
	flag.BoolVar(&cliArgs.verbose, "verbose", false, "Enable info-level logging.")
	flag.BoolVar(&cliArgs.debug, "debug", false, "Enable debug-level logging.")
	flag.Parse()
}

func configLogging() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the warning severity or above by default.
	level := log.WarnLevel
	if cliArgs.debug {
		level = log.DebugLevel
	} else if cliArgs.verbose {
		level = log.InfoLevel
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

	azbatch.Connect()

	go shutdown(os.Interrupt)
	log.Info("Waiting for shutdown to complete.")
	<-shutdownComplete
}
