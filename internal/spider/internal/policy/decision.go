package policy

// URLDecision contains the focused-crawl decision for a URL.
type URLDecision struct {
	Allowed         bool
	ScoreAdjustment float64
	Reason          string
}

// URLRule is a simple ordered host/path rule. The first matching rule wins.
type URLRule struct {
	Host            string
	PathPrefix      string
	Allowed         bool
	ScoreAdjustment float64
	Reason          string
}
