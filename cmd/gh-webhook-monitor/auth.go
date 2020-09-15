package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/iwilltry42/gh-webhook-monitor/pkg/types"
)

// generateJWT generates a new JSON Web Token out of the App's private pem
func generateJWT(appID string, pemFile string) (string, error) {
	pemReader, err := os.Open(pemFile)
	if err != nil {
		return "", err
	}
	defer pemReader.Close()

	pemBytes, err := ioutil.ReadAll(pemReader)
	if err != nil {
		return "", err
	}

	pemReader.Close()

	block, _ := pem.Decode(pemBytes)
	key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	claims := jwt.StandardClaims{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(1 * time.Minute).Unix(), // using the maximum expiration time of 10 minutes
		Issuer:    appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(key)

	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// getGitHubAppInstallationToken uses a JWT token to eventually get an app installation token for git auth
func getGitHubAppInstallationToken(ctx context.Context, ghApp types.GitHubApp) (string, time.Time, error) {
	appToken, err := generateJWT(ghApp.ID, ghApp.PemFile)
	if err != nil {
		return "", time.Time{}, err
	}

	appInstToken, tokenExpirationTime, err := getAppInstallationToken(appToken, ghApp.InstallationID)
	if err != nil {
		return "", time.Time{}, err
	}

	if err := doTestRequest(appInstToken); err != nil {
		return "", time.Time{}, err
	}

	return appInstToken, tokenExpirationTime, nil
}

// getAppInstallationToken requests an app installation token from GitHub
func getAppInstallationToken(appToken, installationID string) (string, time.Time, error) {

	ghURL, err := url.Parse(fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installationID))
	if err != nil {
		return "", time.Time{}, err
	}

	var req = &http.Request{
		Method: http.MethodPost,
		URL:    ghURL,
		Header: http.Header{
			"Authorization": []string{fmt.Sprintf("Bearer %s", appToken)},
			"Accept":        []string{"application/vnd.github.v3+json"},
		},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", time.Time{}, fmt.Errorf("ERROR: Failed to create GitHub App installation token (Response code '%d')", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, err
	}

	var response types.SimplifiedGitHubInstallationAccessTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", time.Time{}, err
	}

	return response.Token, response.ExpiresAt, nil
}
