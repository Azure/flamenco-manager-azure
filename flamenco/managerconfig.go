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

// ManagerConfig returns a Manager configuration file matching the given configuration.
func ManagerConfig(config azconfig.AZConfig, netStack aznetwork.NetworkStack) []byte {
	type TemplateContext struct {
		Name                     string
		AcmeDomainName           string
		PrivateIP                string
		WorkerRegistrationSecret string
	}

	ctx := TemplateContext{
		strings.Title(config.VMName),
		netStack.FQDN(),
		netStack.PrivateIP,
		"", // TODO: generate secret and store in Worker config as well.
	}

	_, myFile, _, ok := runtime.Caller(0)
	if !ok {
		logrus.Panic("unable to determine source file location")
	}
	myDir := path.Dir(myFile)
	templatePath := path.Join(path.Dir(myDir), "templates/flamenco-manager.yaml")
	tmpl := template.Must(template.ParseFiles(templatePath))

	buf := bytes.NewBuffer([]byte{})
	err := tmpl.Execute(buf, ctx)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"template":      templatePath,
			logrus.ErrorKey: err,
		}).Fatal("unable to render template")
	}

	return buf.Bytes()
}
