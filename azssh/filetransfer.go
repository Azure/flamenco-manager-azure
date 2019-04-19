package azssh

import (
	"io/ioutil"
	"strings"

	"github.com/sirupsen/logrus"
)

// UploadLocalFile reads a local file and sends it to the server via SSH.
// WARNING: the given filename must be a simple name, no spaces, no directory, no need for shell escaping.
func (c *Connection) UploadLocalFile(filename string) {
	logger := c.logger.WithField("fileName", filename)
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.WithError(err).Fatal("unable to read file")
	}
	c.UploadAsFile(contents, filename)
}

// UploadAsFile sends bytes to the SSH server and stores them in a file.
// WARNING: the given filename must be a simple name, no spaces, no directory, no need for shell escaping.
func (c *Connection) UploadAsFile(content []byte, filename string) {
	logger := c.logger.WithField("fileName", filename)

	session, err := c.client.NewSession()
	if err != nil {
		logger.WithError(err).Fatal("error creating SSH session")
	}
	defer session.Close()

	logger.Info("sending file")
	pipe, err := session.StdinPipe()
	if err != nil {
		logger.WithError(err).Fatal("unable to create pipe")
	}
	go func() {
		pipe.Write(content)
		pipe.Close()
	}()

	combinedOut, err := session.CombinedOutput("cat > " + filename)
	if err != nil {
		stringOut := strings.TrimSpace(string(combinedOut))
		logger.WithFields(logrus.Fields{
			"output":        stringOut,
			logrus.ErrorKey: err,
		}).Fatal("error running command")
	}
}
