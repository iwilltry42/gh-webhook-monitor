package ghapi

import (
	"fmt"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
)

// doTestRequest tries to get a list of repositories accessible using that token
func doTestRequest(token string) error {
	_, err := DoRequest(token, "https://api.github.com/installation/repositories", http.MethodGet)
	return err
}

// DoRequest does a request against the GitHub API and returns the response
func DoRequest(token, urlString string, method string) (*http.Response, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		log.Errorf("Failed to parse request URL '%s'", urlString)
		return nil, err
	}

	var req = &http.Request{
		Method: method,
		URL:    parsedURL,
		Header: http.Header{
			"Authorization": []string{fmt.Sprintf("Bearer %s", token)},
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
