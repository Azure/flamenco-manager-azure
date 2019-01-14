package azdebug

import (
	"net/http"
	"net/http/httputil"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest"
)

// LogRequest sends all Azure requests to Logrus
func LogRequest() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				log.WithError(err).Error("error in Azure request")
			}
			dump, _ := httputil.DumpRequestOut(r, true)
			log.Debug(string(dump))
			return r, err
		})
	}
}

// LogResponse sends all Azure responses to Logrus
func LogResponse() autorest.RespondDecorator {
	return func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			err := p.Respond(r)
			if err != nil {
				log.WithError(err).Error("error in Azure response")
			}
			dump, _ := httputil.DumpResponse(r, true)
			log.Debug(string(dump))
			return err
		})
	}
}
