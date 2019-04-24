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

package flamenco

import (
	"bytes"
	"path"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/flamenco-deploy-azure/azconfig"
	"gitlab.com/blender-institute/flamenco-deploy-azure/aznetwork"
)

// TemplateContext contains everything necessary for rendering templates.
type TemplateContext struct {
	Name                     string
	AcmeDomainName           string
	PrivateIP                string
	WorkerRegistrationSecret string
	FSTabForStorage          string
	UnixGroupName            string
}

// NewTemplateContext constructs a new context for rendering templated config files.
func NewTemplateContext(
	config azconfig.AZConfig,
	netStack aznetwork.NetworkStack,
	fstab string,
) TemplateContext {
	ctx := TemplateContext{
		Name:                     strings.Title(config.VMName),
		AcmeDomainName:           netStack.FQDN(),
		PrivateIP:                netStack.PrivateIP,
		WorkerRegistrationSecret: config.WorkerRegistrationSecret,
		FSTabForStorage:          fstab,
		UnixGroupName:            UnixGroupName,
	}
	return ctx
}

// RenderTemplate renders a templated config file.
func (tc *TemplateContext) RenderTemplate(templateFile string) []byte {
	logger := logrus.WithField("templateFile", templateFile)
	templatePath := path.Join("files-templated", templateFile)
	tmpl := template.Must(template.ParseFiles(templatePath))

	buf := bytes.NewBuffer([]byte{})
	err := tmpl.Execute(buf, tc)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"templatePath":  templatePath,
			logrus.ErrorKey: err,
		}).Fatal("unable to render template")
	}

	return buf.Bytes()
}
