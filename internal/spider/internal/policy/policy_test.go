package policy

import (
	"spider/internal/utils"
	"strings"
	"testing"
)

func TestEvaluateURL(t *testing.T) {
	allowedDomains := utils.ParseAllowedDomains("github.com,stackoverflow.com")

	testCases := []struct {
		name           string
		rawURL         string
		allowed        bool
		adjustment     float64
		reasonContains string
		wantErr        bool
	}{
		{name: "disallowed host denied", rawURL: "https://example.com/docs", allowed: false, reasonContains: "host not allowed"},
		{name: "malformed URL errors", rawURL: "://bad-url", wantErr: true},
		{name: "github login denied", rawURL: "https://github.com/login", allowed: false, reasonContains: "login"},
		{name: "github repo prioritized", rawURL: "https://github.com/openai/openai-go", allowed: true, adjustment: -2.0, reasonContains: "repository"},
		{name: "github blob prioritized", rawURL: "https://github.com/openai/openai-go/blob/main/README.md", allowed: true, adjustment: -1.0, reasonContains: "source file"},
		{name: "github issues deprioritized", rawURL: "https://github.com/openai/openai-go/issues", allowed: true, adjustment: 1.5, reasonContains: "collaboration"},
		{name: "stackoverflow question prioritized", rawURL: "https://stackoverflow.com/questions/1/example", allowed: true, adjustment: -1.5, reasonContains: "question"},
		{name: "stackoverflow tagged questions deprioritized", rawURL: "https://stackoverflow.com/questions/tagged/go", allowed: true, adjustment: 1.0, reasonContains: "tagged"},
		{name: "stackoverflow users deprioritized", rawURL: "https://stackoverflow.com/users/123/name", allowed: true, adjustment: 1.5, reasonContains: "user"},
		{name: "allowed host fallback neutral", rawURL: "https://github.com/openai", allowed: true, adjustment: 0, reasonContains: "fallback"},
		{name: "rule ordering keeps tagged questions low priority", rawURL: "https://stackoverflow.com/questions/tagged/go", allowed: true, adjustment: 1.0, reasonContains: "tagged"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := EvaluateURL(tc.rawURL, allowedDomains)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.rawURL)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.rawURL, err)
			}

			if decision.Allowed != tc.allowed {
				t.Fatalf("expected allowed=%v for %q, got %v", tc.allowed, tc.rawURL, decision.Allowed)
			}

			if decision.ScoreAdjustment != tc.adjustment {
				t.Fatalf("expected adjustment=%v for %q, got %v", tc.adjustment, tc.rawURL, decision.ScoreAdjustment)
			}

			if tc.reasonContains != "" && !strings.Contains(decision.Reason, tc.reasonContains) {
				t.Fatalf("expected reason %q to contain %q", decision.Reason, tc.reasonContains)
			}
		})
	}
}

func TestEvaluateURLWithEmptyAllowlist(t *testing.T) {
	decision, err := EvaluateURL("https://example.com/docs", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !decision.Allowed {
		t.Fatal("expected empty allowlist to allow URL")
	}

	if decision.ScoreAdjustment != 0 {
		t.Fatalf("expected neutral adjustment, got %v", decision.ScoreAdjustment)
	}
}
