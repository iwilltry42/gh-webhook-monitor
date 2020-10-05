package ghapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/iwilltry42/gh-webhook-monitor/pkg/types"
	"github.com/iwilltry42/gh-webhook-monitor/pkg/util"

	log "github.com/sirupsen/logrus"
)

// repoRegexp matches several variants of repo addresses that can be passed to this application
var repoRegexp = regexp.MustCompile(`^(?P<protocol>http://|https://|git@)?(?P<github_domain>github\.com)?/?(?P<owner>[A-Za-z0-9-_]+)/(?P<repo>[A-Za-z0-9-_]+)(\.git|/.*)?`)

// ValidateAndNormalizeRepositoryIdentifier tries to extract the repository identifier in the <owner>/<repo> format and returns it, if possible
func ValidateAndNormalizeRepositoryIdentifier(identifier string) (string, bool) {
	// trim leading and trailing whitespaces
	identifier = strings.TrimSpace(identifier)

	// match identifier against regexp
	match := repoRegexp.FindStringSubmatch(identifier)
	if len(match) == 0 {
		return identifier, false
	}

	// get matching groups from regexp
	submatches := util.MapSubexpNames(repoRegexp.SubexpNames(), match)

	// return repo identifier in <owner>/<repo> format
	return fmt.Sprintf("%s/%s", submatches["owner"], submatches["repo"]), true
}

func (ghAppInstallation *GitHubAppInstallation) GetReposByTeamSlug(teamSlug string) ([]string, error) {
	resp, err := ghAppInstallation.DoAPIRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/teams/%s/repos?per_page=100", ghAppInstallation.Organization, teamSlug))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response []GHAPIResponseRepos
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	repos := []string{}
	for _, repo := range response {
		r, ok := ValidateAndNormalizeRepositoryIdentifier(repo.FullName)
		if !ok {
			return nil, fmt.Errorf("Failed to validate repo '%s'", repo.FullName)
		}
		repos = append(repos, r)
	}

	return repos, nil
}

// GenerateRepoList generates a list of repositories to target for inspection
func GenerateRepoList(ctx context.Context, ghAppInstallation *GitHubAppInstallation, config *types.RepositoryConfig) ([]string, error) {
	log.Debugf("Generating repository list from config:\n%+v", config)

	repos := make(map[string]bool, 1)

	if config.FilterTeamSlugs != nil {
		for _, teamSlug := range config.FilterTeamSlugs {
			log.Debugf("Fetching repos for team '%s'...", teamSlug)
			newRepos, err := ghAppInstallation.GetReposByTeamSlug(teamSlug)
			if err != nil {
				return nil, err
			}
			log.Debugf("Found %d repos for team '%s'", len(newRepos), teamSlug)

			// only add repos that are not already in the list
			for _, newRepo := range newRepos {
				if _, exists := repos[newRepo]; !exists {
					repos[newRepo] = true
				}
			}
		}
	}

	// drop repos which are on the exlusion list
	for repo := range repos {
		for _, nope := range config.ExcludeRepositories {
			excludeRepo, ok := ValidateAndNormalizeRepositoryIdentifier(nope)
			if !ok {
				return nil, fmt.Errorf("Failed to validate repo from exclusion list '%s'", nope)
			}
			if repo == excludeRepo {
				delete(repos, repo)
			}
		}
	}

	// add repos that are on the inclusion list
	for _, repo := range config.IncludeRepositories {
		if r, ok := ValidateAndNormalizeRepositoryIdentifier(repo); ok {
			repos[r] = true
		} else {
			return nil, fmt.Errorf("Failed to validate repository identifier '%s'", repo)
		}
	}

	repoList := []string{}
	for repo := range repos {
		repoList = append(repoList, repo)
	}

	log.Debugf("Filtered repo list contains %d repos", len(repoList))

	return repoList, nil

}
