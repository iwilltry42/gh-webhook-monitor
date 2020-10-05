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

func configFromEnv() (*ghapi.GitHubAppInstallation, *types.RepositoryConfig, *types.WebhookConfig, time.Duration, time.Duration, error) {

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
			return nil, nil, nil, 0, 0, fmt.Errorf("Failed to parse wait time '%s' to time.Duration format", wt)
		}
	}

	// repo refresh wait time: time to wait before refreshing the list of repositories again
	rrwt := strings.TrimSpace(os.Getenv("GWM_REPO_REFRESH_WAIT_TIME"))

	var repoRefreshWaitTime time.Duration
	if rrwt == "" {
		repoRefreshWaitTime = types.DEFAULT_REPO_REFRESH_WAIT_TIME
	} else {
		var err error
		repoRefreshWaitTime, err = time.ParseDuration(rrwt)
		if err != nil {
			return nil, nil, nil, 0, 0, fmt.Errorf("Failed to parse repo refresh wait time '%s' to time.Duration format", rrwt)
		}
	}

	if os.Getenv("GWM_DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	}

	// Webhook Configuration
	webhookConfig := types.WebhookConfig{}

	// Regexp to match against webhook target URLs
	webhookTargetRegexp := strings.TrimSpace(os.Getenv("GWM_WEBHOOKS_FILTER_TARGET_REGEXP"))
	if webhookTargetRegexp != "" {
		webhookConfig.FilterTargetURLRegexp = regexp.MustCompile(webhookTargetRegexp)
	}
	log.Debugf("Webhook Filter Target Regexp '%+v'", webhookConfig.FilterTargetURLRegexp)

	// Generate Repository Search Config
	targetRepositoryListConfig := types.RepositoryConfig{
		IncludeRepositories: []string{},
		FilterTeamSlugs:     []string{},
		ExcludeRepositories: []string{},
	}

	includeRepos := strings.Split(os.Getenv("GWM_REPOS_INCLUDE"), ",")
	for _, r := range includeRepos {
		r = strings.TrimSpace(r)
		if r != "" {
			targetRepositoryListConfig.IncludeRepositories = append(targetRepositoryListConfig.IncludeRepositories, r)
		}
	}

	teamSlugs := strings.Split(os.Getenv("GWM_REPOS_FILTER_TEAM_SLUGS"), ",")
	for _, ts := range teamSlugs {
		ts = strings.TrimSpace(ts)
		if ts != "" {
			targetRepositoryListConfig.FilterTeamSlugs = append(targetRepositoryListConfig.FilterTeamSlugs, ts)
		}
	}

	return &ghAppInstallation, &targetRepositoryListConfig, &webhookConfig, waitTime, repoRefreshWaitTime, nil
}

func checkWebhooks(ctx context.Context, ghAppInstallation *ghapi.GitHubAppInstallation, repos []string, webhookConfig *types.WebhookConfig) {
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
			metrics.RepositoryFailedWebhookListTotal.WithLabelValues(repo, "requestError").Inc()
			continue
		}

		var hookResponse []ghapi.GHAPIResponseHook

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body\n%+v", err)
			metrics.RepositoryFailedWebhookListTotal.WithLabelValues(repo, "readResponseError").Inc()
			continue
		}

		resp.Body.Close()

		if err := json.Unmarshal(respBody, &hookResponse); err != nil {
			log.Errorf("Failed to unmarshal hook response for repo '%s'\n%+v", repo, err)
			metrics.RepositoryFailedWebhookListTotal.WithLabelValues(repo, "unmarshalResponseBodyError").Inc()
			continue
		}

		for _, hook := range hookResponse {
			if webhookConfig.FilterTargetURLRegexp != nil && !webhookConfig.FilterTargetURLRegexp.MatchString(hook.Config.URL) { // TODO: add function to filter webhooks before continuing
				log.Debugf("Webhook Target URL '%s' does not match provided Regexp ('%s'), ignoring...", hook.Config.URL, webhookConfig.FilterTargetURLRegexp)
				continue
			}
			log.Infof("Repo %s - Hook %s -> Target %s :: Last Status Code %d (msg: %s)", repo, hook.URL, hook.Config.URL, hook.LastResponse.Code, hook.LastResponse.Status)

			// CodeGroup
			var cgFound *metrics.CodeGroup
			metrics.WebhookLastStatusCodeGroup.Reset() // reset all metrics in this vector
			for _, cg := range metrics.CodeGroups {
				if hook.LastResponse.Code >= cg.LowerBound && hook.LastResponse.Code <= cg.LowerBound {
					metrics.WebhookLastStatusCodeGroup.WithLabelValues(repo, hook.URL, hook.Config.URL, hook.LastResponse.Status, cg.Name).Set(1)
					cgFound = &cg
					break
				}
			}
			if cgFound == nil {
				metrics.WebhookLastStatusCodeGroup.WithLabelValues(repo, hook.URL, hook.Config.URL, hook.LastResponse.Status, metrics.CodeGroupOthers.Name).Set(1)
				metrics.WebhookLastStatusCodeTotal.WithLabelValues(repo, hook.URL, hook.Config.URL, hook.LastResponse.Status, fmt.Sprintf("%d", hook.LastResponse.Code), metrics.CodeGroupOthers.Name).Inc()
			} else {
				metrics.WebhookLastStatusCodeTotal.WithLabelValues(repo, hook.URL, hook.Config.URL, hook.LastResponse.Status, fmt.Sprintf("%d", hook.LastResponse.Code), cgFound.Name).Inc()
			}
			continue
		}
	}
}

func main() {
	var err error

	// expose metrics for Prometheus
	http.Handle("/metrics", promhttp.Handler())

	// configure application from environment variables
	ghAppInstallation, repoListConfig, webhookConfig, waitTime, repoRefreshWaitTime, err := configFromEnv()
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

	var repos []string

	// get list of repositories
	repos, err = ghapi.GenerateRepoList(context.Background(), ghAppInstallation, repoListConfig)
	if err != nil {
		log.Fatalln(err)
	}

	// Prepare Label Values for the Repo List Metric

	teamSlugsStr := strings.Join(repoListConfig.FilterTeamSlugs, "|")
	var includeFiltersStr string
	var excludeFiltersStr string
	if repoListConfig.IncludeRepositoryRegexp != nil {
		includeFiltersStr = strings.Join(append(repoListConfig.IncludeRepositories, repoListConfig.IncludeRepositoryRegexp.String()), "|")
	} else {
		includeFiltersStr = strings.Join(repoListConfig.IncludeRepositories, "|")
	}
	if repoListConfig.ExcludeRepositoryRegexp != nil {
		excludeFiltersStr = strings.Join(append(repoListConfig.ExcludeRepositories, repoListConfig.ExcludeRepositoryRegexp.String()), "|")
	} else {
		excludeFiltersStr = strings.Join(repoListConfig.ExcludeRepositories, "|")
	}
	// update list of repositories every now and then
	go func(ctx context.Context, ghAppInstallation *ghapi.GitHubAppInstallation, waitTime time.Duration, repoListConfig *types.RepositoryConfig) {
		for {
			var err error
			repos, err = ghapi.GenerateRepoList(ctx, ghAppInstallation, repoListConfig)
			if err != nil {
				log.Errorf("Failed to refresh list of repositories: %+v", err)
			}
			metrics.RepositoryListCount.WithLabelValues(teamSlugsStr, includeFiltersStr, excludeFiltersStr).Set(float64(len(repos)))
			log.Infof("Refreshed Repository List: Found %d repositories -> Next refresh in %s...", len(repos), waitTime)
			time.Sleep(waitTime)
		}
	}(context.Background(), ghAppInstallation, repoRefreshWaitTime, repoListConfig)

	// continuously check webhook statuses for all repos
	go func(ctx context.Context, ghAppInstallation *ghapi.GitHubAppInstallation, waitTime time.Duration, webhookConfig *types.WebhookConfig) {
		for {
			apiRate, err := ghAppInstallation.GetAPIRateLimit()
			if err != nil {
				log.Errorf("Failed to get Rate Limit data from API: %+v", err)
			}
			reset := time.Unix(apiRate.Reset, 0)
			log.Infof("API Rate Limit Usage: %d/%d remaining, resets at %s", apiRate.Remaining, apiRate.Limit, reset)
			metrics.APIRateLimitRemaining.WithLabelValues(ghAppInstallation.ParentApp.ID, ghAppInstallation.ID).Set(float64(apiRate.Remaining))

			checkWebhooks(ctx, ghAppInstallation, repos, webhookConfig)
			log.Infof("Processed webhooks for %d repositories -> Next iteration in %s...", len(repos), waitTime)
			time.Sleep(waitTime)
		}
	}(context.Background(), ghAppInstallation, waitTime, webhookConfig)

	log.Fatal(http.ListenAndServe(":8080", nil))

}
