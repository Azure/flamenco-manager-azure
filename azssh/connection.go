package azssh

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"golang.org/x/crypto/ssh"
)

// Connection models an SSH connection
type Connection struct {
	sshContext Context
	client     *ssh.Client
	logger     *logrus.Entry
}

// Connect connects to a machine via SSH.
func Connect(sshContext Context, address string) Connection {
	if strings.IndexRune(address, ':') == -1 {
		address = address + ":22"
	}

	client, err := ssh.Dial("tcp", address, sshContext.sshConfig)
	logger := logrus.WithField("remoteAddress", address)
	if err != nil {
		logrus.WithError(err).Fatal("SSH connection failed")
	}

	return Connection{
		sshContext,
		client,
		logger,
	}
}

// Close closes the SSH connection.
func (c *Connection) Close() {
	if err := c.client.Close(); err != nil {
		c.logger.WithError(err).Error("error closing SSH connection")
	}
}

// Run a command, return the output. Errors are fatal.
func (c *Connection) run(cmd string, args ...interface{}) string {
	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	session, err := c.client.NewSession()
	if err != nil {
		c.logger.WithError(err).Fatal("error creating SSH session")
	}
	defer session.Close()

	command := fmt.Sprintf(cmd, args...)
	logger := c.logger.WithField("command", command)
	logger.Info("running command via SSH")

	combinedOut, err := session.CombinedOutput(command)
	stringOut := strings.TrimSpace(string(combinedOut))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"output":        stringOut,
			logrus.ErrorKey: err,
		}).Fatal("error running command")
	}

	return stringOut
}
