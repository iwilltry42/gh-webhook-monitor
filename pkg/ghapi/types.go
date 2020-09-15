package ghapi

import (
	"time"
)

const (
	DEFAULT_GITHUB_API_BASE_URL = "https://api.github.com"
)

var GitHubAPIBaseURL string = DEFAULT_GITHUB_API_BASE_URL

// GHAPIResponseHookLastStatus represents the last_response part of a single webhook item in the GitHub repository webhook API response
type GHAPIResponseHookLastStatus struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type GHAPIResponseHookConfig struct {
	ContentType string `json:"content_type"`
	InsecureSSL string `json:"insecure_ssl"`
	URL         string `json:"url"`
}

// GHAPIResponseHook represents a single list item of the GitHub repository webhook API response
type GHAPIResponseHook struct {
	Type         string                      `json:"type"`
	ID           int                         `json:"id"`
	Name         string                      `json:"name"`
	Active       bool                        `json:"active"`
	Events       []string                    `json:"events"`
	Config       GHAPIResponseHookConfig     `json:"config"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	CreatedAt    time.Time                   `json:"created_at"`
	URL          string                      `json:"url"`
	TestURL      string                      `json:"test_url"`
	PingURL      string                      `json:"ping_url"`
	LastResponse GHAPIResponseHookLastStatus `json:"last_response"`
}

// GHAPIResponseInstallationTokenSimplified is a simple representation of the response you get when requesting
// a GitHub App installation token (see https://docs.github.com/en/rest/reference/apps#create-an-installation-access-token-for-an-app)
type GHAPIResponseInstallationTokenSimplified struct {
	Token               string            `json:"token"`
	ExpiresAt           time.Time         `json:"expires_at"`
	RepositorySelection string            `json:"repository_selection"`
	Permissions         map[string]string `json:"permissions"`
}

type GitHubAppInstallationToken struct {
	Token     string
	ExpiresAt time.Time
}

// GitHubApp holds all config options that we need to authenticate as a GitHub App installation
type GitHubApp struct {
	ID                string
	InstallationID    string
	PemFile           string
	InstallationToken GitHubAppInstallationToken
}
