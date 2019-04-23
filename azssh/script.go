package azssh

import (
	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/flamenco"
)

// SetupUsers sets up the users and groups on Flamenco Manager.
func (c *Connection) SetupUsers() {
	c.logger.Info("setting up users")
	c.run("sudo groupadd --force %s", flamenco.UnixGroupName)
	c.run("sudo usermod %s --append --groups %s", flamenco.AdminUsername, flamenco.UnixGroupName)
}

// RunInstallScript sends the install script to the VM and runs it there.
func (c *Connection) RunInstallScript() {
	c.UploadLocalFile(flamenco.InstallScriptName)
	c.run("chmod +x %s", flamenco.InstallScriptName)

	c.loggingRun(c.logger, "bash %s", flamenco.InstallScriptName)
	c.logger.WithFields(logrus.Fields{
		"scriptName": flamenco.InstallScriptName,
	}).Info("installation script completed")
}
