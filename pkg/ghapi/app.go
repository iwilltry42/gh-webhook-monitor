package ghapi

import "net/http"

// DoAPIRequest does a request against the GitHub API and returns the response
func (ghApp *GitHubApp) DoAPIRequest(method, path string) (*http.Response, error) {
	appJWTToken, err := generateJWT(ghApp.ID, ghApp.PemFile)
	if err != nil {
		return nil, err
	}
	return doAPIRequest(method, path, appJWTToken)
}
