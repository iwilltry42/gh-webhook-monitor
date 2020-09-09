package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// ghRepositoryHookLastResponse represents the last_response part of a single webhook item in the GitHub repository webhook API response
type ghRepositoryHookLastResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ghRepositoryHook represents a single list item of the GitHub repository webhook API response
type ghRepositoryHook struct {
	Type         string                       `json:"type"`
	ID           int                          `json:"id"`
	Name         string                       `json:"name"`
	Active       bool                         `json:"active"`
	Events       []string                     `json:"events"`
	Config       map[string]interface{}       `json:"config"`
	UpdatedAt    time.Time                    `json:"updated_at"`
	CreatedAt    time.Time                    `json:"created_at"`
	URL          string                       `json:"url"`
	TestURL      string                       `json:"test_url"`
	PingURL      string                       `json:"ping_url"`
	LastResponse ghRepositoryHookLastResponse `json:"last_response"`
}

type GitHubAppInstallationToken struct {
	Token     string
	ExpiresAt time.Time
}

// GitHubApp holds all config options that we need to authenticate as a GitHub App installation
type GitHubApp struct {
	ID                string
	InstallationID    string
	PemFile           string
	InstallationToken GitHubAppInstallationToken
}

var GHApp GitHubApp

var Repositories []string

// repoRegexp matches several variants of repo addresses that can be passed to this application
var repoRegexp = regexp.MustCompile(`^(?P<protocol>http://|https://|git@)?(?P<github_domain>github\.com)?/?(?P<owner>[A-Za-z0-9-_]+)/(?P<repo>[A-Za-z0-9-_]+)(\.git|/.*)?`)

func main() {
	GHApp = GitHubApp{
		ID:             os.Getenv("GWM_GH_APP_ID"),
		InstallationID: os.Getenv("GWM_GH_APP_INST_ID"),
		PemFile:        os.Getenv("GWM_GH_APP_PEM"),
	}
	repoURLList := strings.Split(os.Getenv("GWM_REPOS"), ",")

	Repositories = []string{}

	for _, repoURL := range repoURLList {
		match := repoRegexp.FindStringSubmatch(repoURL)
		if len(match) == 0 {
			log.Fatalf("Failed to match repo regexp on repo %s", repoURL)
		}

		submatches := mapSubexpNames(repoRegexp.SubexpNames(), match)

		repo := fmt.Sprintf("%s/%s", submatches["owner"], submatches["repo"])

		log.Printf("Handling repo '%s'", repo)

		Repositories = append(Repositories, repo)
	}

	var err error

	GHApp.InstallationToken.Token, GHApp.InstallationToken.ExpiresAt, err = getGitHubAppInstallationToken(context.Background(), GHApp)
	if err != nil {
		log.Errorln("Failed to get GH App Installation Token")
		log.Fatalln(err)
	}

	if time.Now().After(GHApp.InstallationToken.ExpiresAt) {
		GHApp.InstallationToken.Token, GHApp.InstallationToken.ExpiresAt, err = getGitHubAppInstallationToken(context.Background(), GHApp)
		if err != nil {
			log.Errorln("Failed to get GH App Installation Token")
			log.Fatalln(err)
		}
	}

	for _, repo := range Repositories {
		resp, err := doRequest(GHApp.InstallationToken.Token, fmt.Sprintf("https://api.github.com/repos/%s/hooks", repo), http.MethodGet)
		if err != nil {
			log.Errorf("Failed to get hooks for repo '%s'", repo)
			log.Fatalln(err)
		}
		defer resp.Body.Close()

		var hookResponse []ghRepositoryHook

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorln("Failed to read response body")
			log.Fatalln(err)
		}
		resp.Body.Close()

		if err := json.Unmarshal(respBody, &hookResponse); err != nil {
			log.Errorf("Failed to unmarshal hook response for repo '%s'", repo)
			log.Fatalln(err)
		}

		for _, hook := range hookResponse {
			log.Infof("-> %s -> %s -> %d", repo, hook.URL, hook.LastResponse.Code)
		}
	}
}
