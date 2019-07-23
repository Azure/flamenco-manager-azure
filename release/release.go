package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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

// githubReleaseCreateResponse is received from GitHub,
// see https://developer.github.com/v3/repos/releases/#create-a-release
type githubReleaseCreateResponse struct {
	UploadURL string `json:"upload_url"`
	Name      string `json:"name"`
}

var cliArgs struct {
	version  string
	fileglob string
}

var client http.Client

func main() {
	parseCliArgs()

	bytes := makeRequestPayload()
	os.Stdout.Write(bytes)
	os.Stdout.Write([]byte("\n"))

	client = http.Client{
		Timeout: 1 * time.Minute,
	}

	token := loadPersonalAccessToken()
	resp := doGithubCreateRelease(bytes, token, githubURL)
	if resp.UploadURL == "" {
		logrus.Fatal("upload_url not found in response JSON")
	}

	assets := getUploadableAssets()
	doGitHubUploadAssets(assets, token, resp.UploadURL)

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

func makeRequest(url, authToken string, body []byte) *http.Request {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		logger := logrus.WithField("url", url)
		logger.WithError(err).Fatal("unable to create HTTP request")
	}
	req.Header.Set("Authorization", "token "+authToken)

	return req
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
func doGithubCreateRelease(payload []byte, authToken string, url string) githubReleaseCreateResponse {
	logger := logrus.WithField("url", url)

	req := makeRequest(url, authToken, payload)
	req.Header.Set("Content-Type", "application/json")

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

	response := githubReleaseCreateResponse{}
	unmarshaller := json.NewDecoder(resp.Body)
	if err := unmarshaller.Decode(&response); err != nil {
		logger.WithError(err).Fatal("error reading/parsing HTTP response")
	}

	if niceResponse, err := json.MarshalIndent(response, "", "    "); err != nil {
		logger.WithError(err).Error("unable to nicely format response")
		fmt.Printf("%#v\n", response)
	} else {
		logger.Info("Response from GitHub (well, the fields we're using):")
		os.Stdout.Write(niceResponse)
		os.Stdout.Write([]byte("\n"))
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
func doGitHubUploadAsset(fpath string, authToken string, assetUploadURL string) {
	file, err := ioutil.ReadFile(fpath)
	if err != nil {
		panic(err)
	}

	// Get the file name and use it to build the upload URL. GitHub gives us a URL with
	// "{?name,label}" in there and expects us to replace that.
	fname := path.Base(fpath)
	nameSuffix := "?name=" + url.QueryEscape(fname)
	uploadURL := strings.Replace(assetUploadURL, "{?name,label}", nameSuffix, -1)

	req := makeRequest(uploadURL, authToken, file)
	req.Header.Set("Content-Type", "application/zip")

	logger := logrus.WithField("uploadURL", uploadURL)
	logger.Info("Uploading file")
	resp, err := client.Do(req)
	if err != nil {
		logger.WithError(err).Fatal("error performing HTTP request")
	}

	logger = logger.WithField("statusCode", resp.StatusCode)
	if resp.StatusCode == 201 {
		logger.Info("Asset uploaded successfully")
	} else {
		logger.Error("Unable to upload asset")
	}
}
