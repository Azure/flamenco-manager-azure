package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
)

const (
	appName     = "Flamenco Azure Deploy"
	downloadURL = "https://flamenco.io/download/azure/" // MUST end in slash
	gitlabURL   = "https://gitlab.com/api/v4/projects/blender-institute%2Fflamenco-deploy-azure/releases"
)

type gitlabRelease struct {
	Name        string `json:"name"`
	TagName     string `json:"tag_name"`
	Description string `json:"description"`
	Assets      struct {
		Links []gitlabLink `json:"links"`
	} `json:"assets"`
}

type gitlabLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

var cliArgs struct {
	version  string
	fileglob string
}

func main() {
	parseCliArgs()

	bytes := makeRequestPayload()
	os.Stdout.Write(bytes)

	token := loadPersonalAccessToken()

	doGitlabRequest(bytes, token)

	os.Stdout.Write([]byte("\n"))
}

func parseCliArgs() {
	flag.StringVar(&cliArgs.version, "version", "", "Version to release")
	flag.StringVar(&cliArgs.fileglob, "fileglob", "", "Glob of files to include")
	flag.Parse()
}

// makeRequestPayload constructs the JSON-encoded request to create a new release.
func makeRequestPayload() []byte {
	logrus.WithFields(logrus.Fields{
		"version":  cliArgs.version,
		"fileglob": cliArgs.fileglob,
	}).Info("constructing GitLab release JSON")

	release := gitlabRelease{
		Name:        cliArgs.version,
		TagName:     cliArgs.version,
		Description: fmt.Sprintf("Version %s of %s", cliArgs.version, appName),
	}

	paths, err := filepath.Glob(cliArgs.fileglob)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logrus.ErrorKey: err,
			"fileglob":      cliArgs.fileglob,
		}).Fatal("unable to glob")
	}
	if len(paths) == 0 {
		logrus.WithField("fileglob", cliArgs.fileglob).Fatal("no files matched")
	}
	for _, fpath := range paths {
		fname := path.Base(fpath)
		logrus.WithField("filename", fname).Info("listing")
		link := gitlabLink{
			Name: fname,
			URL:  downloadURL + fname,
		}
		release.Assets.Links = append(release.Assets.Links, link)
	}

	bytes, err := json.MarshalIndent(release, "", "    ")
	if err != nil {
		logrus.WithError(err).Fatal("unable to marshal to JSON")
	}

	return bytes
}

// loadPersonalAccessToken loads the Gitlab access token from .gitlabAccessToken.
func loadPersonalAccessToken() string {
	fname, err := filepath.Abs(".gitlabAccessToken")
	if err != nil {
		logrus.WithError(err).Fatal("unable to construct absolute path")
	}
	logrus.WithField("filename", fname).Info("reading Gitlab access token")

	tokenBytes, err := ioutil.ReadFile(fname)
	if err != nil {
		logrus.WithError(err).Fatal("unable to read personal access token, see https://gitlab.com/profile/personal_access_tokens")
	}

	return strings.TrimSpace(string(tokenBytes))
}

// doGitlabRequest sends a payload to Gitlab, authenticated with the user's token.
// The response is assumed to be JSON, and shown nicely formatted on stdout.
func doGitlabRequest(payload []byte, authToken string) {
	client := http.Client{
		Timeout: 1 * time.Minute,
	}

	logger := logrus.WithField("url", gitlabURL)
	req, err := http.NewRequest("POST", gitlabURL, bytes.NewReader(payload))
	if err != nil {
		logger.WithError(err).Fatal("unable to create HTTP request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Private-Token", authToken)

	resp, err := client.Do(req)
	if err != nil {
		logger.WithError(err).Fatal("error performing HTTP request")
	}

	logger = logger.WithField("statusCode", resp.StatusCode)
	if resp.Header.Get("Content-Type") != "application/json" {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.WithError(err).Fatal("error reading HTTP response")
		}

		logger.WithField("body", string(body)).Fatal("non-JSON response from Gitlab")
	}

	response := bson.M{}
	unmarshaller := json.NewDecoder(resp.Body)
	if err := unmarshaller.Decode(&response); err != nil {
		logger.WithError(err).Fatal("error reading/parsing HTTP response")
	}

	if niceResponse, err := json.MarshalIndent(response, "", "    "); err != nil {
		logger.WithError(err).Error("unable to nicely format response")
		fmt.Printf("%#v\n", response)
	} else {
		logger.Info("Response from Gitlab:")
		os.Stdout.Write(niceResponse)
	}
}
