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
	WebhookTargetRegexp *regexp.Regexp
)

// repoRegexp matches several variants of repo addresses that can be passed to this application
var repoRegexp = regexp.MustCompile(`^(?P<protocol>http://|https://|git@)?(?P<github_domain>github\.com)?/?(?P<owner>[A-Za-z0-9-_]+)/(?P<repo>[A-Za-z0-9-_]+)(\.git|/.*)?`)

func configFromEnv() (*types.GitHubApp, *types.RepositoryListConfig, time.Duration, error) {

	// Setup GitHub App used for authentication
	ghApp := types.GitHubApp{
		ID:             os.Getenv("GWM_GH_APP_ID"),
		InstallationID: os.Getenv("GWM_GH_APP_INST_ID"),
		PemFile:        os.Getenv("GWM_GH_APP_PEM"),
	}

	// wait time: time to wait between iterations
	wt := strings.TrimSpace(os.Getenv("GWM_WAIT_TIME"))

	var waitTime time.Duration
	if wt == "" {
		waitTime = types.DEFAULT_WAIT_TIME
	} else {
		var err error
		waitTime, err = time.ParseDuration(wt)
		if err != nil {
			log.Errorf("Failed to parse wait time '%s' to time.Duration format", wt)
			os.Exit(1)
		}

		if os.Getenv("GWM_DEBUG") != "" {
			log.SetLevel(log.DebugLevel)
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
			return nil, nil, 0, fmt.Errorf("Cannot handle repository: Invalid repository identifier '%s'", repoURL)
		}

		log.Printf("Handling repo '%s'", repo)

		repositoryListConfig.IncludeRepositories = append(repositoryListConfig.IncludeRepositories, repo)
	}

	return &ghApp, &repositoryListConfig, waitTime, nil
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

func checkWebhooks(ctx context.Context, ghApp *types.GitHubApp, repos []string) {
	var err error

	// renew token in case it expired
	if time.Now().After(ghApp.InstallationToken.ExpiresAt) {
		log.Debugln("Renewing App Installation Token...")
		ghApp.InstallationToken.Token, ghApp.InstallationToken.ExpiresAt, err = getGitHubAppInstallationToken(context.Background(), ghApp)
		if err != nil {
			log.Errorln("Failed to get GH App Installation Token")
			log.Fatalln(err)
		}
	}

	// loop through list of repositories
	for _, repo := range repos {
		log.Debugf("Getting hooks for repo '%s'...", repo)
		resp, err := doRequest(ghApp.InstallationToken.Token, fmt.Sprintf("https://api.github.com/repos/%s/hooks", repo), http.MethodGet)
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
}

func main() {
	var err error

	// expose metrics for Prometheus
	http.Handle("/metrics", promhttp.Handler())

	// configure application from environment variables
	ghApp, repoListConfig, waitTime, err := configFromEnv()
	if err != nil {
		log.Errorln("Failed to create configuration")
		log.Fatalln(err)
	}

	// authenticate against GitHub as a GitHub app
	ghApp.InstallationToken.Token, ghApp.InstallationToken.ExpiresAt, err = getGitHubAppInstallationToken(context.Background(), ghApp)
	if err != nil {
		log.Errorln("Failed to get GH App Installation Token")
		log.Fatalln(err)
	}

	// continuously check webhook statuses for all repos
	go func(ctx context.Context, ghApp *types.GitHubApp, waitTime time.Duration, repositories []string) {
		for {
			checkWebhooks(context.Background(), ghApp, repositories)
			log.Infof("Waiting for %s...", waitTime)
			time.Sleep(waitTime)
		}
	}(context.Background(), ghApp, waitTime, repoListConfig.IncludeRepositories)

	log.Fatal(http.ListenAndServe(":8080", nil))

}
