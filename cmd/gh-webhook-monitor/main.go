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

	"github.com/iwilltry42/gh-webhook-monitor/pkg/ghapi"
	"github.com/iwilltry42/gh-webhook-monitor/pkg/metrics"
	"github.com/iwilltry42/gh-webhook-monitor/pkg/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	WebhookTargetRegexp *regexp.Regexp
)

func configFromEnv() (*ghapi.GitHubAppInstallation, *types.TargetRepositoryListConfig, time.Duration, error) {

	// Setup GitHub App used for authentication
	ghApp := ghapi.GitHubApp{
		ID:      os.Getenv("GWM_GH_APP_ID"),
		PemFile: os.Getenv("GWM_GH_APP_PEM"),
	}

	ghAppInstallation := ghapi.GitHubAppInstallation{
		ID:        os.Getenv("GWM_GH_APP_INST_ID"),
		ParentApp: &ghApp,
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
			return nil, nil, 0, fmt.Errorf("Failed to parse wait time '%s' to time.Duration format", wt)
		}
	}

	if os.Getenv("GWM_DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	}

	// Regexp to match against webhook target URLs
	webhookTargetRegexp := strings.TrimSpace(os.Getenv("GWM_WEBHOOK_TARGET_REGEXP"))
	if webhookTargetRegexp != "" {
		WebhookTargetRegexp = regexp.MustCompile(webhookTargetRegexp)
	}

	// Generate Repository Search Config
	targetRepositoryListConfig := types.TargetRepositoryListConfig{}

	targetRepositoryListConfig.IncludeRepositories = strings.Split(os.Getenv("GWM_REPOS_INCLUDE"), ",")
	targetRepositoryListConfig.FilterTeamSlugs = strings.Split(os.Getenv("GWM_REPOS_FILTER_TEAM_SLUGS"), ",")

	return &ghAppInstallation, &targetRepositoryListConfig, waitTime, nil
}

func checkWebhooks(ctx context.Context, ghAppInstallation *ghapi.GitHubAppInstallation, repos []string) {
	// renew token in case it expired
	if time.Now().After(ghAppInstallation.TokenExpirationTime) {
		log.Debugln("Renewing App Installation Token...")
		if err := ghAppInstallation.RefreshToken(context.Background()); err != nil {
			log.Errorln("Failed to get GH App Installation Token")
			log.Fatalln(err)
		}
	}

	// loop through list of repositories
	for _, repo := range repos {
		log.Debugf("Getting hooks for repo '%s'...", repo)
		resp, err := ghAppInstallation.DoAPIRequest(http.MethodGet, fmt.Sprintf("/repos/%s/hooks", repo))
		if err != nil {
			log.Errorf("Failed to get hooks for repo '%s'\n%+v", repo, err)
			metrics.RepositoryFailedWebhookList.WithLabelValues(repo, "requestError").Inc()
			continue
		}

		var hookResponse []ghapi.GHAPIResponseHook

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body\n%+v", err)
			metrics.RepositoryFailedWebhookList.WithLabelValues(repo, "readResponseError").Inc()
			continue
		}

		resp.Body.Close()

		if err := json.Unmarshal(respBody, &hookResponse); err != nil {
			log.Errorf("Failed to unmarshal hook response for repo '%s'\n%+v", repo, err)
			metrics.RepositoryFailedWebhookList.WithLabelValues(repo, "unmarshalResponseBodyError").Inc()
			continue
		}

		for _, hook := range hookResponse {
			if WebhookTargetRegexp != nil && !WebhookTargetRegexp.MatchString(hook.Config.URL) {
				log.Infof("Webhook Target URL '%s' does not match provided Regexp ('%s'), ignoring...", hook.Config.URL, WebhookTargetRegexp)
				continue
			}
			log.Infof("Repo %s - Hook %s -> %s :: %d", repo, hook.URL, hook.Config.URL, hook.LastResponse.Code)
			metrics.WebhookLastStatusCode.WithLabelValues(repo, hook.URL, hook.Config.URL, fmt.Sprintf("%d", hook.LastResponse.Code)).Inc()
			continue
		}
	}
}

func main() {
	var err error

	// expose metrics for Prometheus
	http.Handle("/metrics", promhttp.Handler())

	// configure application from environment variables
	ghAppInstallation, repoListConfig, waitTime, err := configFromEnv()
	if err != nil {
		log.Errorln("Failed to create configuration")
		log.Fatalln(err)
	}

	// get some installation details
	if err := ghAppInstallation.GetDetails(); err != nil {
		log.Errorln("Failed to get App Installation Details")
		log.Fatalln(err)
	}

	// authenticate against GitHub as a GitHub app
	if err := ghAppInstallation.RefreshToken(context.Background()); err != nil {
		log.Errorln("Failed to get GH App Installation Token")
		log.Fatalln(err)
	}

	// get list of repositories
	repos, err := ghapi.GenerateRepoList(context.Background(), ghAppInstallation, repoListConfig)
	if err != nil {
		log.Fatalln(err)
	}

	// continuously check webhook statuses for all repos
	go func(ctx context.Context, ghAppInstallation *ghapi.GitHubAppInstallation, waitTime time.Duration, repositories []string) {
		for {
			checkWebhooks(context.Background(), ghAppInstallation, repositories)
			log.Infof("Waiting for %s...", waitTime)
			time.Sleep(waitTime)
		}
	}(context.Background(), ghAppInstallation, waitTime, repos)

	log.Fatal(http.ListenAndServe(":8080", nil))

}
