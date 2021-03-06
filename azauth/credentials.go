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

package azauth

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	// CredentialsFile contains the Azure API credentials.
	CredentialsFile = "client_credentials.json"
)

// EnsureCredentialsFile creates the credentials file using the AZ CLI client if it doesn't exist yet.
func EnsureCredentialsFile(ctx context.Context) {
	logger := logrus.WithField("credentialsFile", CredentialsFile)
	if credStat, err := os.Stat(CredentialsFile); err == nil && credStat.Size() > 0 {
		logger.Debug("credentials file exists")
		return
	}

	logger.Info("creating credentials file")

	credFile, err := os.Create(CredentialsFile)
	if err != nil {
		logger.WithError(err).Fatal("unable to create credentials file")
	}
	defer credFile.Close()

	cliArgs := []string{"az", "ad", "sp", "create-for-rbac", "--sdk-auth"}
	logger = logger.WithField("cliArgs", strings.Join(cliArgs, " "))

	cmd := exec.CommandContext(ctx, cliArgs[0], cliArgs[1:]...)

	// Capture stdout
	outpipe, err := cmd.StdoutPipe()
	if err != nil {
		logger.WithError(err).Fatal("unable to create stdout pipe for AZ CLI command")
	}
	go io.Copy(credFile, outpipe)

	// Capture stderr
	errpipe, err := cmd.StderrPipe()
	if err != nil {
		logger.WithError(err).Fatal("unable to create stderr pipe for AZ CLI command")
	}
	var stderrBytes []byte
	go func() { stderrBytes, _ = ioutil.ReadAll(errpipe) }()

	// Run the command and wait for it to complete.
	if err := cmd.Start(); err != nil {
		logger.WithError(err).Fatal("unable to run AZ CLI command")
	}
	if err := cmd.Wait(); err != nil {
		stderr := string(stderrBytes)

		if strings.Contains(stderr, "'az login'") {
			logger.WithError(err).Warn("error running AZ CLI command")
			logrus.Fatal("run 'az login' before starting Flamenco Azure Deploy")
		}
		logger.WithFields(logrus.Fields{
			"stderr":        stderr,
			logrus.ErrorKey: err,
		}).Fatal("error running AZ CLI command")
	}

	// Now the credentials file exists, and our job is done.
}
