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

// GHAPIResponseInstallationDetails for /app/installations/{installation_id}
type GHAPIResponseInstallationDetails struct {
	ID                  int                  `json:"id"`
	Account             GHAPIResponseAccount `json:"account"`
	AccessTokensURL     string               `json:"access_tokens_url"`
	RepositoriesURL     string               `json:"repositories_url"`
	HTMLURL             string               `json:"html_url"`
	AppID               int                  `json:"app_id"`
	TargetID            int                  `json:"target_id"`
	TargetType          string               `json:"target_type"`
	Permissions         map[string]string    `json:"permissions"`
	Events              []string             `json:"events"`
	SingleFileName      string               `json:"single_file_name"`
	RepositorySelection string               `json:"repository_selection"`
}

type GHAPIResponseAccount struct {
	Login            string `json:"login"`
	ID               int    `json:"id"`
	NodeID           string `json:"node_id"`
	URL              string `json:"url"`
	ReposURL         string `json:"repos_url"`
	EventsURL        string `json:"events_url"`
	HooksURL         string `json:"hooks_url"`
	IssuesURL        string `json:"issues_url"`
	MembersURL       string `json:"members_url"`
	PublicMembersURL string `json:"public_members_url"`
	AvatarURL        string `json:"avatar_url"`
	Description      string `json:"description"`
}

type GitHubAppInstallation struct {
	ID                  string
	Token               string
	TokenExpirationTime time.Time
	Organization        string
	ParentApp           *GitHubApp
}

// GitHubApp holds all config options that we need to authenticate as a GitHub App installation
type GitHubApp struct {
	ID      string
	PemFile string
}
