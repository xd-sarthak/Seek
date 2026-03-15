package policy

import (
	"fmt"
	"net/url"
	"spider/internal/utils"
	"strings"
)

var githubReservedOwners = map[string]struct{}{
	"about":            {},
	"account":          {},
	"collections":      {},
	"contact":          {},
	"customer-stories": {},
	"enterprise":       {},
	"events":           {},
	"explore":          {},
	"features":         {},
	"issues":           {},
	"login":            {},
	"marketplace":      {},
	"notifications":    {},
	"orgs":             {},
	"organizations":    {},
	"pricing":          {},
	"pulls":            {},
	"search":           {},
	"security":         {},
	"session":          {},
	"sessions":         {},
	"settings":         {},
	"signup":           {},
	"sponsors":         {},
	"stars":            {},
	"topics":           {},
	"trending":         {},
	"users":            {},
}

// EvaluateURL applies the host allowlist and focused-crawl policy for rawURL.
func EvaluateURL(rawURL string, allowedDomains map[string]struct{}) (URLDecision, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return URLDecision{}, fmt.Errorf("could not parse URL: %w", err)
	}

	host := utils.NormalizeHost(parsedURL.Host)
	if host == "" {
		return URLDecision{}, fmt.Errorf("URL has no host")
	}

	if !utils.IsHostAllowed(host, allowedDomains) {
		return URLDecision{
			Allowed: false,
			Reason:  "host not allowed",
		}, nil
	}

	path := normalizePath(parsedURL.Path)

	for _, rule := range rulesForHost(host) {
		if matchesRule(rule, host, path) {
			return URLDecision{
				Allowed:         rule.Allowed,
				ScoreAdjustment: rule.ScoreAdjustment,
				Reason:          rule.Reason,
			}, nil
		}
	}

	switch host {
	case "github.com":
		return evaluateGitHubPath(path), nil
	case "stackoverflow.com":
		return evaluateStackOverflowPath(path), nil
	default:
		return URLDecision{Allowed: true, ScoreAdjustment: 0, Reason: "default allowed host"}, nil
	}
}

func matchesRule(rule URLRule, host, path string) bool {
	if rule.Host != host {
		return false
	}

	return pathMatchesPrefix(path, rule.PathPrefix)
}

func pathMatchesPrefix(path, prefix string) bool {
	if prefix == "/" {
		return true
	}

	if strings.HasSuffix(prefix, "/") {
		return strings.HasPrefix(path, prefix)
	}

	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}

	if path != "/" {
		path = strings.TrimSuffix(path, "/")
		if path == "" {
			return "/"
		}
	}

	return path
}

func evaluateGitHubPath(path string) URLDecision {
	segments := pathSegments(path)
	if len(segments) < 2 {
		return URLDecision{Allowed: true, ScoreAdjustment: 0, Reason: "github fallback"}
	}

	if _, reserved := githubReservedOwners[segments[0]]; reserved {
		return URLDecision{Allowed: true, ScoreAdjustment: 0.5, Reason: "github reserved surface"}
	}

	if len(segments) == 2 {
		return URLDecision{Allowed: true, ScoreAdjustment: -2.0, Reason: "github repository root"}
	}

	switch segments[2] {
	case "blob":
		return URLDecision{Allowed: true, ScoreAdjustment: -1.0, Reason: "github source file"}
	case "tree":
		return URLDecision{Allowed: true, ScoreAdjustment: -0.5, Reason: "github repository tree"}
	case "issues", "pulls", "discussions":
		return URLDecision{Allowed: true, ScoreAdjustment: 1.5, Reason: "github collaboration surface"}
	default:
		return URLDecision{Allowed: true, ScoreAdjustment: 0, Reason: "github repository page"}
	}
}

func evaluateStackOverflowPath(path string) URLDecision {
	switch {
	case pathMatchesPrefix(path, "/questions/"):
		return URLDecision{Allowed: true, ScoreAdjustment: -1.5, Reason: "stackoverflow question page"}
	case pathMatchesPrefix(path, "/users/"):
		return URLDecision{Allowed: true, ScoreAdjustment: 1.5, Reason: "stackoverflow user page"}
	case pathMatchesPrefix(path, "/tags/"):
		return URLDecision{Allowed: true, ScoreAdjustment: 1.0, Reason: "stackoverflow tag page"}
	default:
		return URLDecision{Allowed: true, ScoreAdjustment: 0, Reason: "stackoverflow fallback"}
	}
}

func pathSegments(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "/")
}
