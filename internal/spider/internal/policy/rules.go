package policy

var githubRules = []URLRule{
	{Host: "github.com", PathPrefix: "/login", Allowed: false, Reason: "github login page"},
	{Host: "github.com", PathPrefix: "/settings", Allowed: false, Reason: "github settings page"},
	{Host: "github.com", PathPrefix: "/search", Allowed: false, Reason: "github search page"},
	{Host: "github.com", PathPrefix: "/notifications", Allowed: false, Reason: "github notifications page"},
	{Host: "github.com", PathPrefix: "/sessions", Allowed: false, Reason: "github session page"},
	{Host: "github.com", PathPrefix: "/marketplace", Allowed: true, ScoreAdjustment: 2.0, Reason: "github marketplace page"},
	{Host: "github.com", PathPrefix: "/topics", Allowed: true, ScoreAdjustment: 1.5, Reason: "github topic page"},
	{Host: "github.com", PathPrefix: "/orgs", Allowed: true, ScoreAdjustment: 1.5, Reason: "github organization page"},
	{Host: "github.com", PathPrefix: "/users", Allowed: true, ScoreAdjustment: 2.0, Reason: "github user page"},
}

var stackOverflowRules = []URLRule{
	{Host: "stackoverflow.com", PathPrefix: "/search", Allowed: false, Reason: "stackoverflow search page"},
	{Host: "stackoverflow.com", PathPrefix: "/users/login", Allowed: false, Reason: "stackoverflow login page"},
	{Host: "stackoverflow.com", PathPrefix: "/users", Allowed: true, ScoreAdjustment: 1.5, Reason: "stackoverflow user page"},
	{Host: "stackoverflow.com", PathPrefix: "/tags", Allowed: true, ScoreAdjustment: 1.0, Reason: "stackoverflow tags page"},
	{Host: "stackoverflow.com", PathPrefix: "/questions/tagged", Allowed: true, ScoreAdjustment: 1.0, Reason: "stackoverflow tagged questions page"},
	{Host: "stackoverflow.com", PathPrefix: "/questions", Allowed: true, ScoreAdjustment: -1.5, Reason: "stackoverflow question page"},
}

func rulesForHost(host string) []URLRule {
	switch host {
	case "github.com":
		return githubRules
	case "stackoverflow.com":
		return stackOverflowRules
	default:
		return nil
	}
}
