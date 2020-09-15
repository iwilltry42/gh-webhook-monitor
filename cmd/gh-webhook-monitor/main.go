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

	"github.com/iwilltry42/gh-webhook-monitor/pkg/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	GHApp               types.GitHubApp
	Repositories        []string
	WebhookTargetRegexp *regexp.Regexp
	WaitTime            time.Duration
)

// repoRegexp matches several variants of repo addresses that can be passed to this application
var repoRegexp = regexp.MustCompile(`^(?P<protocol>http://|https://|git@)?(?P<github_domain>github\.com)?/?(?P<owner>[A-Za-z0-9-_]+)/(?P<repo>[A-Za-z0-9-_]+)(\.git|/.*)?`)

func configFromEnv() error {

	// Setup GitHub App used for authentication
	GHApp = types.GitHubApp{
		ID:             os.Getenv("GWM_GH_APP_ID"),
		InstallationID: os.Getenv("GWM_GH_APP_INST_ID"),
		PemFile:        os.Getenv("GWM_GH_APP_PEM"),
	}

	// wait time: time to wait between iterations
	wt := strings.TrimSpace(os.Getenv("GWM_WAIT_TIME"))

	if wt == "" {
		WaitTime = types.DEFAULT_WAIT_TIME
	} else {
		var err error
		WaitTime, err = time.ParseDuration(wt)
		if err != nil {
			log.Errorf("Failed to parse wait time '%s' to time.Duration format", wt)
			os.Exit(1)
		}
	}

	// Regexp to match against webhook target URLs
	webhookTargetRegexp := strings.TrimSpace(os.Getenv("GWM_WEBHOOK_TARGET_REGEXP"))
	if webhookTargetRegexp != "" {
		WebhookTargetRegexp = regexp.MustCompile(webhookTargetRegexp)
	}

	// Generate List of repositories
	repoURLList := strings.Split(os.Getenv("GWM_REPOS_INCLUDE"), ",")
	repositoryListConfig := types.RepositoryListConfig{}

	repositoryListConfig.IncludeRepositories = []string{}

	for _, repoURL := range repoURLList {
		repo, ok := validateAndNormalizeRepositoryIdentifier(repoURL)
		if !ok {
			return fmt.Errorf("Cannot handle repository: Invalid repository identifier '%s'", repoURL)
		}

		log.Printf("Handling repo '%s'", repo)

		repositoryListConfig.IncludeRepositories = append(repositoryListConfig.IncludeRepositories, repo)
	}

	return nil
}

func validateAndNormalizeRepositoryIdentifier(identifier string) (string, bool) {
	// trim leading and trailing whitespaces
	identifier = strings.TrimSpace(identifier)

	// match identifier against regexp
	match := repoRegexp.FindStringSubmatch(identifier)
	if len(match) == 0 {
		return identifier, false
	}

	// get matching groups from regexp
	submatches := mapSubexpNames(repoRegexp.SubexpNames(), match)

	// return repo identifier in <owner>/<repo> format
	return fmt.Sprintf("%s/%s", submatches["owner"], submatches["repo"]), true
}

func checkWebhooks() {
	var err error

	for {
		// renew token in case it expired
		if time.Now().After(GHApp.InstallationToken.ExpiresAt) {
			GHApp.InstallationToken.Token, GHApp.InstallationToken.ExpiresAt, err = getGitHubAppInstallationToken(context.Background(), GHApp)
			if err != nil {
				log.Errorln("Failed to get GH App Installation Token")
				log.Fatalln(err)
			}
		}

		// get webhooks from all repositories
		for _, repo := range Repositories {
			resp, err := doRequest(GHApp.InstallationToken.Token, fmt.Sprintf("https://api.github.com/repos/%s/hooks", repo), http.MethodGet)
			if err != nil {
				log.Errorf("Failed to get hooks for repo '%s'", repo)
				repositoryFailedWebhookList.WithLabelValues(repo, "requestError").Inc()
			}

			var hookResponse []types.GHRepositoryHook

			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Errorln("Failed to read response body\n%w", err)
				repositoryFailedWebhookList.WithLabelValues(repo, "readResponseError").Inc()
			}

			resp.Body.Close()

			if err := json.Unmarshal(respBody, &hookResponse); err != nil {
				log.Errorf("Failed to unmarshal hook response for repo '%s'", repo)
				repositoryFailedWebhookList.WithLabelValues(repo, "unmarshalResponseBodyError").Inc()
			}

			for _, hook := range hookResponse {
				if WebhookTargetRegexp != nil && !WebhookTargetRegexp.MatchString(hook.Config.URL) {
					log.Infof("Webhook Target URL '%s' does not match provided Regexp ('%s'), ignoring...", hook.Config.URL, WebhookTargetRegexp)
					continue
				}
				log.Infof("Repo %s - Hook %s -> %s :: %d", repo, hook.URL, hook.Config.URL, hook.LastResponse.Code)
				webhookLastStatusCode.WithLabelValues(repo, hook.URL, hook.Config.URL, fmt.Sprintf("%d", hook.LastResponse.Code)).Inc()
			}
		}

		log.Infof("Waiting for %s...", WaitTime)
		time.Sleep(WaitTime)
	}
}

func main() {
	var err error

	// expose metrics for Prometheus
	http.Handle("/metrics", promhttp.Handler())

	// configure application from environment variables
	if err = configFromEnv(); err != nil {
		log.Errorln("Failed to create configuration")
		log.Fatalln(err)
	}

	// authenticate against GitHub as a GitHub app
	GHApp.InstallationToken.Token, GHApp.InstallationToken.ExpiresAt, err = getGitHubAppInstallationToken(context.Background(), GHApp)
	if err != nil {
		log.Errorln("Failed to get GH App Installation Token")
		log.Fatalln(err)
	}

	go checkWebhooks()

	log.Fatal(http.ListenAndServe(":8080", nil))

}
