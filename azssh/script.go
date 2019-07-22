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
	"github.com/sirupsen/logrus"
	"github.com/Azure/flamenco-manager-azure/flamenco"
)

// SetupUsers sets up the users and groups on Flamenco Manager.
func (c *Connection) SetupUsers() {
	c.logger.Info("setting up users")
	c.run("sudo groupadd --force %s", flamenco.UnixGroupName)
	c.run("sudo usermod %s --append --groups %s", flamenco.AdminUsername, flamenco.UnixGroupName)
}

// RunInstallScript sends the install script to the VM and runs it there.
func (c *Connection) RunInstallScript() {
	c.run("chmod +x %s", flamenco.InstallScriptName)

	c.loggingRun(c.logger, "bash %s", flamenco.InstallScriptName)
	c.logger.WithFields(logrus.Fields{
		"scriptName": flamenco.InstallScriptName,
	}).Info("installation script completed")
}
