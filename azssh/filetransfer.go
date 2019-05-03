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
	"io/ioutil"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

// UploadStaticFile reads a local file from 'files-static' and sends it to the server via SSH.
// WARNING: the given filename must be a simple name, no spaces, no directory, no need for shell escaping.
func (c *Connection) UploadStaticFile(filename string) {
	c.UploadLocalFile(path.Join("files-static", filename))
}

// UploadLocalFile reads a local file and sends it to the server via SSH.
// WARNING: the given filename must be a simple name, no spaces, no directory, no need for shell escaping.
func (c *Connection) UploadLocalFile(filename string) {
	logger := c.logger.WithField("filename", filename)

	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.WithError(err).Fatal("unable to read file")
	}
	c.UploadAsFile(contents, path.Base(filename))
}

// UploadAsFile sends bytes to the SSH server and stores them in a file.
// WARNING: the given filename must be a simple name, no spaces, no directory, no need for shell escaping.
func (c *Connection) UploadAsFile(content []byte, filename string) {
	logger := c.logger.WithField("filename", filename)

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
