package flamenco

import (
	"bytes"
	"path"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/aznetwork"
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
