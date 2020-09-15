package repo

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/iwilltry42/gh-webhook-monitor/pkg/util"
)

// repoRegexp matches several variants of repo addresses that can be passed to this application
var repoRegexp = regexp.MustCompile(`^(?P<protocol>http://|https://|git@)?(?P<github_domain>github\.com)?/?(?P<owner>[A-Za-z0-9-_]+)/(?P<repo>[A-Za-z0-9-_]+)(\.git|/.*)?`)

// ValidateAndNormalizeRepositoryIdentifier tries to extract the repository identifier in the <owner>/<repo> format and returns it, if possible
func ValidateAndNormalizeRepositoryIdentifier(identifier string) (string, bool) {
	// trim leading and trailing whitespaces
	identifier = strings.TrimSpace(identifier)

	// match identifier against regexp
	match := repoRegexp.FindStringSubmatch(identifier)
	if len(match) == 0 {
		return identifier, false
	}

	// get matching groups from regexp
	submatches := util.MapSubexpNames(repoRegexp.SubexpNames(), match)

	// return repo identifier in <owner>/<repo> format
	return fmt.Sprintf("%s/%s", submatches["owner"], submatches["repo"]), true
}
