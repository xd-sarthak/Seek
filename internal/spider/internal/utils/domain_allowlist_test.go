package utils

import "testing"

func TestParseAllowedDomains(t *testing.T) {
	allowedDomains := ParseAllowedDomains(DefaultAllowedDomains + ", www.stackoverflow.com , ,developer.mozilla.org")

	if len(allowedDomains) != 12 {
		t.Fatalf("expected 12 domains, got %d", len(allowedDomains))
	}

	if _, ok := allowedDomains["github.com"]; !ok {
		t.Fatalf("expected github.com to be present")
	}

	if _, ok := allowedDomains["stackoverflow.com"]; !ok {
		t.Fatalf("expected stackoverflow.com to be normalized and present")
	}

	if _, ok := allowedDomains["docs.rs"]; !ok {
		t.Fatalf("expected docs.rs to be present")
	}
}

func TestIsURLAllowed(t *testing.T) {
	allowedDomains := ParseAllowedDomains(DefaultAllowedDomains)

	testCases := []struct {
		name    string
		rawURL  string
		allowed bool
		wantErr bool
	}{
		{name: "exact host allowed", rawURL: "https://github.com/openai/openai-go", allowed: true},
		{name: "raw github content allowed", rawURL: "https://raw.githubusercontent.com/openai/openai-go/main/README.md", allowed: true},
		{name: "www host allowed", rawURL: "https://www.stackoverflow.com/questions/1", allowed: true},
		{name: "docs rs allowed", rawURL: "https://docs.rs/serde/latest/serde/", allowed: true},
		{name: "subdomain denied", rawURL: "https://api.github.com/repos/openai/openai-go", allowed: false},
		{name: "unrelated host denied", rawURL: "https://example.com/docs", allowed: false},
		{name: "malformed URL errors", rawURL: "://bad-url", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowed, err := IsURLAllowed(tc.rawURL, allowedDomains)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.rawURL)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.rawURL, err)
			}

			if allowed != tc.allowed {
				t.Fatalf("expected allowed=%v for %q, got %v", tc.allowed, tc.rawURL, allowed)
			}
		})
	}
}

func TestIsURLAllowedWithEmptyAllowlist(t *testing.T) {
	allowed, err := IsURLAllowed("https://example.com/docs", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !allowed {
		t.Fatal("expected empty allowlist to allow URL")
	}
}
