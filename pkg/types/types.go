package types

import (
	"regexp"
	"time"
)

const (
	DEFAULT_WAIT_TIME              = 5 * time.Minute
	DEFAULT_REPO_REFRESH_WAIT_TIME = 1 * time.Hour
)

// RepositoryConfig describes the configuration for targeted repositories
type RepositoryConfig struct {
	IncludeRepositories     []string       `mapstructure:"include" yaml:"include"`
	ExcludeRepositories     []string       `mapstructure:"exclude" yaml:"exclude"`
	IncludeRepositoryRegexp *regexp.Regexp `mapstructure:"includeRegexp" yaml:"includeRegexp"`
	ExcludeRepositoryRegexp *regexp.Regexp `mapstructure:"excludeRegexp" yaml:"excludeRegexp"`
	FilterTeamSlugs         []string       `mapstructure:"teamSlugs" yaml:"teamSlugs"`
}

type WebhookConfig struct {
	FilterTargetURLRegexp *regexp.Regexp
}
