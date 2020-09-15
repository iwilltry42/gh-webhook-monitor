package types

import (
	"regexp"
	"time"
)

const (
	DEFAULT_WAIT_TIME = 5 * time.Minute
)

// GHRepositoryHookLastResponse represents the last_response part of a single webhook item in the GitHub repository webhook API response
type GHRepositoryHookLastResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type GHRepositoryHookConfig struct {
	ContentType string `json:"content_type"`
	InsecureSSL string `json:"insecure_ssl"`
	URL         string `json:"url"`
}

// GHRepositoryHook represents a single list item of the GitHub repository webhook API response
type GHRepositoryHook struct {
	Type         string                       `json:"type"`
	ID           int                          `json:"id"`
	Name         string                       `json:"name"`
	Active       bool                         `json:"active"`
	Events       []string                     `json:"events"`
	Config       GHRepositoryHookConfig       `json:"config"`
	UpdatedAt    time.Time                    `json:"updated_at"`
	CreatedAt    time.Time                    `json:"created_at"`
	URL          string                       `json:"url"`
	TestURL      string                       `json:"test_url"`
	PingURL      string                       `json:"ping_url"`
	LastResponse GHRepositoryHookLastResponse `json:"last_response"`
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

type RepositoryListConfig struct {
	IncludeRepositories        []string
	ExcludeRepositories        []string
	IncludeRepositoryRegexp    *regexp.Regexp
	ExcludeRepositoryRegexp    *regexp.Regexp
	IncludeRepositoryTeamSlugs []string
}
