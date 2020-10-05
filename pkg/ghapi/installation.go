package ghapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (ghAppInstallation *GitHubAppInstallation) DoAPIRequest(method, path string) (*http.Response, error) {
	return doAPIRequest(method, path, ghAppInstallation.Token)
}

// RefreshToken uses a JWT token to eventually get an app installation token for git auth
func (ghAppInstallation *GitHubAppInstallation) RefreshToken(ctx context.Context) error {
	var err error

	ghApp := ghAppInstallation.ParentApp

	appToken, err := generateJWT(ghApp.ID, ghApp.PemFile)
	if err != nil {
		return err
	}

	ghAppInstallation.Token, ghAppInstallation.TokenExpirationTime, err = getAppInstallationToken(appToken, ghAppInstallation.ID)
	if err != nil {
		return err
	}

	if _, err := ghAppInstallation.DoAPIRequest(http.MethodGet, "/installation/repositories"); err != nil {
		return err
	}

	return nil
}

// GetDetails fills the GitHub App Installation with some required details (like organization)
func (ghAppInstallation *GitHubAppInstallation) GetDetails() error {
	ghApp := ghAppInstallation.ParentApp

	resp, err := ghApp.DoAPIRequest(http.MethodGet, fmt.Sprintf("/app/installations/%s", ghAppInstallation.ID))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response GHAPIResponseInstallationDetails
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	log.Infof("%+v", response)

	ghAppInstallation.Organization = response.Account.Login
	log.Infof("App Installation belongs to Org '%s'", response.Account.Login)

	return nil

}

// GetAPIRateLimit fills the GitHub App Installation with some required details (like organization)
func (ghAppInstallation *GitHubAppInstallation) GetAPIRateLimit() (GHAPIRate, error) {
	resp, err := ghAppInstallation.DoAPIRequest(http.MethodGet, "/rate_limit")
	if err != nil {
		return GHAPIRate{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GHAPIRate{}, err
	}

	var response GHAPIResponseRateLimit
	if err := json.Unmarshal(body, &response); err != nil {
		return GHAPIRate{}, err
	}

	return response.Rate, nil
}
