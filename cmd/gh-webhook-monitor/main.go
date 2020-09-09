package main

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type ghRepositoryHookLastResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

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

var repoRegexp = regexp.MustCompile(`^(?P<protocol>http://|https://|git@)?(?P<github_domain>github\.com)?/?(?P<owner>[A-Za-z0-9-_]+)/(?P<repo>[A-Za-z0-9-_]+)(\.git|/.*)?`)

func main() {
	ghAppID := os.Getenv("GWM_GH_APP_ID")
	ghAppInstID := os.Getenv("GWM_GH_APP_INST_ID")
	ghAppPem := os.Getenv("GWM_GH_APP_PEM")
	repoURLList := strings.Split(os.Getenv("GWM_REPOS"), ",")

	repos := []string{}

	for _, repoURL := range repoURLList {
		match := repoRegexp.FindStringSubmatch(repoURL)
		if len(match) == 0 {
			log.Fatalf("Failed to match repo regexp on repo %s", repoURL)
		}

		submatches := mapSubexpNames(repoRegexp.SubexpNames(), match)

		repo := fmt.Sprintf("%s/%s", submatches["owner"], submatches["repo"])

		log.Printf("Handling repo '%s'", repo)

		repos = append(repos, repo)
	}

	var token string
	var tokenExpirationTime time.Time
	var err error

	token, tokenExpirationTime, err = getGitHubAppInstallationToken(context.Background(), ghAppID, ghAppPem, ghAppInstID)
	if err != nil {
		log.Errorln("Failed to get GH App Installation Token")
		log.Fatalln(err)
	}

	if time.Now().After(tokenExpirationTime) {
		token, _, err = getGitHubAppInstallationToken(context.Background(), ghAppID, ghAppPem, ghAppInstID) // TODO: use tokenExpirationTime
		if err != nil {
			log.Errorln("Failed to get GH App Installation Token")
			log.Fatalln(err)
		}
	}

	for _, repo := range repos {
		resp, err := doRequest(token, fmt.Sprintf("https://api.github.com/repos/%s/hooks", repo), http.MethodGet)
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
