package ghapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
)

// DoTestRequest tries to get a list of repositories accessible using that token
func (ghApp *GitHubApp) DoTestRequest() error {
	_, err := ghApp.DoAPIRequest(http.MethodGet, "/installation/repositories")
	return err
}

// DoAPIRequest does a request against the GitHub API and returns the response
func (ghApp *GitHubApp) DoAPIRequest(method, path string) (*http.Response, error) {
	// ensure leading slash on path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// construct URL

	parsedURL, err := url.Parse(GitHubAPIBaseURL + path)
	if err != nil {
		log.Errorf("Failed to parse request URL '%s'", path)
		return nil, err
	}

	var req = &http.Request{
		Method: method,
		URL:    parsedURL,
		Header: http.Header{
			"Authorization": []string{fmt.Sprintf("Bearer %s", ghApp.InstallationToken.Token)},
			"Accept":        []string{"application/vnd.github.v3+json"},
		},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("Request returned non-200 status code (%d)", resp.StatusCode)
	}

	return resp, nil
}
