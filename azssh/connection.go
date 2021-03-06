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

package azssh

import (
	"fmt"
	"strings"
	"time"

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
	if !strings.ContainsRune(address, ':') {
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

// loggingRun runs a command and logs its output. Errors are fatal.
func (c *Connection) loggingRun(logger *logrus.Entry, cmd string, args ...interface{}) {
	session, err := c.client.NewSession()
	if err != nil {
		c.logger.WithError(err).Fatal("error creating SSH session")
	}
	defer session.Close()

	stdoutReader, err := session.StdoutPipe()
	if err != nil {
		logger.WithError(err).Fatal("unable to open stdout pipe")
	}
	stderrReader, err := session.StderrPipe()
	if err != nil {
		logger.WithError(err).Fatal("unable to open stderr pipe")
	}
	stdoutLines, stdoutErr := LineReader(stdoutReader)
	stderrLines, stderrErr := LineReader(stderrReader)

	command := fmt.Sprintf(cmd, args...)
	if err := session.Start(command); err != nil {
		logger.WithError(err).Fatal("unable to start command")
	}

	doneChan := make(chan error)
	go func() {
		doneChan <- session.Wait()
		close(doneChan)
	}()

	func() {
		for {
			select {
			case line := <-stdoutLines:
				logger.WithField("channel", "stdout").Info(line)
			case line := <-stderrLines:
				logger.WithField("channel", "stderr").Info(line)
			case err := <-doneChan:
				if err != nil {
					logger.WithError(err).Fatal("command exited with an error")
				}
				logger.Debug("command completed")
				return
			case <-time.After(5 * time.Minute):
				logger.Fatal("timeout waiting for command output")
			}
		}
	}()

	outErr := stdoutErr()
	errErr := stderrErr()
	if outErr != nil || errErr != nil {
		logger.WithFields(logrus.Fields{
			"stdoutErr": outErr,
			"stderrErr": errErr,
		}).Fatal("error reading stdout/err")
	}
}
