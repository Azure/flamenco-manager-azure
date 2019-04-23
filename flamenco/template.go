package flamenco

import (
	"bytes"
	"path"
	"runtime"
	"strings"
	"text/template"

	"gitlab.com/blender-institute/azure-go-test/aznetwork"

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
)

// TemplateContext contains everything necessary for rendering templates.
type TemplateContext struct {
	Name                     string
	AcmeDomainName           string
	PrivateIP                string
	WorkerRegistrationSecret string
}

// NewTemplateContext constructs a new context for rendering templated config files.
func NewTemplateContext(config azconfig.AZConfig, netStack aznetwork.NetworkStack) TemplateContext {
	ctx := TemplateContext{
		strings.Title(config.VMName),
		netStack.FQDN(),
		netStack.PrivateIP,
		"", // TODO: generate secret and store in Worker config as well.
	}
	return ctx
}

// RenderTemplate renders a templated config file.
func (tc *TemplateContext) RenderTemplate(templateFile string) []byte {
	logger := logrus.WithField("templateFile", templateFile)

	_, myFile, _, ok := runtime.Caller(0)
	if !ok {
		logger.Panic("unable to determine source file location")
	}
	myDir := path.Dir(myFile)
	templatePath := path.Join(path.Dir(myDir), "templates", templateFile)
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
