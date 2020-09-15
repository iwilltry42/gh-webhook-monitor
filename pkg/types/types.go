package types

import (
	"regexp"
	"time"
)

const (
	DEFAULT_WAIT_TIME = 5 * time.Minute
)

// TargetRepositoryListConfig describes the configuration for targeted repositories
type TargetRepositoryListConfig struct {
	IncludeRepositories        []string       `mapstructure:"include" yaml:"include"`
	ExcludeRepositories        []string       `mapstructure:"exclude" yaml:"exclude"`
	IncludeRepositoryRegexp    *regexp.Regexp `mapstructure:"includeRegexp" yaml:"includeRegexp"`
	ExcludeRepositoryRegexp    *regexp.Regexp `mapstructure:"excludeRegexp" yaml:"excludeRegexp"`
	IncludeRepositoryTeamSlugs []string       `mapstructure:"includeTeamSlug" yaml:"includeTeamSlug"`
}
