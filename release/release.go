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
	appName   = "Flamenco Manager Azure"
	githubURL = "https://api.github.com/repos/Azure/flamenco-manager-azure/releases"
)

type githubRelease struct {
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
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
	token = "token " + token

	resp := doGithubCreateRelease(bytes, token, githubURL)

	if resp["upload_url"] == nil {
		logrus.Fatal("upload_url not found")
	}

	assets := getUploadableAssets()

	doGitHubUploadAssets(assets, token, resp["upload_url"].(string))

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
	}).Info("constructing GitHub release JSON")

	release := githubRelease{
		Name:    cliArgs.version,
		TagName: cliArgs.version,
		Body:    fmt.Sprintf("Version %s of %s", cliArgs.version, appName),
	}

	bytes, err := json.MarshalIndent(release, "", "    ")
	if err != nil {
		logrus.WithError(err).Fatal("unable to marshal to JSON")
	}

	return bytes
}

// loadPersonalAccessToken loads the Github access token from .githubAccessToken.
func loadPersonalAccessToken() string {
	fname, err := filepath.Abs(".githubAccessToken")
	if err != nil {
		logrus.WithError(err).Fatal("unable to construct absolute path")
	}
	logrus.WithField("filename", fname).Info("reading Github access token")

	tokenBytes, err := ioutil.ReadFile(fname)
	if err != nil {
		logrus.WithError(err).Fatal("unable to read personal access token, see https://github.com/settings/tokens")
	}

	return strings.TrimSpace(string(tokenBytes))
}

// doGithubCreateRelease sends a payload to Github, authenticated with the user's token.
// The response is assumed to be JSON, and shown nicely formatted on stdout.
func doGithubCreateRelease(payload []byte, authToken string, url string) bson.M {
	client := http.Client{
		Timeout: 1 * time.Minute,
	}

	logger := logrus.WithField("url", url)
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	// req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.WithError(err).Fatal("unable to create HTTP request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)

	resp, err := client.Do(req)
	if err != nil {
		logger.WithError(err).Fatal("error performing HTTP request")
	}

	logger = logger.WithField("statusCode", resp.StatusCode)

	if resp.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.WithError(err).Fatal("error reading HTTP response")
		}

		logger.WithField("body", string(body)).Fatal("non-JSON response from GitHub")
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
		logger.Info("Response from GitHub:")
		os.Stdout.Write(niceResponse)
	}
	return response
}

// getUploadableAssets returns the list of files in the dist/ directory
func getUploadableAssets() []string {
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
	return paths
}

// doGitHubUploadAssets calls doGitHubUploadAsset for each file in paths
func doGitHubUploadAssets(paths []string, authToken string, url string) {
	for _, fpath := range paths {
		doGitHubUploadAsset(fpath, authToken, url)
	}
}

// doGitHubUploadAsset uploads an asset to the GitHub release
func doGitHubUploadAsset(fpath string, authToken string, url string) {
	file, err := ioutil.ReadFile(fpath)
	if err != nil {
		panic(err)
	}

	// Get the file name and use it to build the upload URL
	fname := path.Base(fpath)
	nameSuffix := "?name=" + fname
	url = strings.Replace(url, "{?name,label}", nameSuffix, -1)

	logger := logrus.WithField("url", url)
	logger.Info("Uploading file")
	req, err := http.NewRequest("POST", url, bytes.NewReader(file))
	if err != nil {
		logger.WithError(err).Fatal("unable to create HTTP request")
	}
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("Authorization", authToken)

	// Construct a Client and perform the upload request
	client := http.Client{
		Timeout: 1 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.WithError(err).Fatal("error performing HTTP request")
	}

	statusCode := resp.StatusCode
	if statusCode == 201 {
		logger.WithField("statusCode", resp.StatusCode).Info("Asset uploaded successfully")
	} else {
		logger.WithField("statusCode", resp.StatusCode).Error("Unable to upload asset")
	}

}
