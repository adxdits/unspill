package detect

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSyntheticDetection(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	token := "ghp_" + "7Fj29kLmN4pQr8StU1vWx5YzA0bCd3EfG6hI"
	matches := engine.Scan("synthetic marker " + token + " gitleaks:allow")
	if len(matches) != 1 || matches[0].RuleID != "github-pat" {
		t.Fatalf("expected one GitHub pattern finding, got %d", len(matches))
	}
	encoded, err := json.Marshal(matches)
	if err != nil || strings.Contains(string(encoded), token) || strings.Contains(string(encoded), "synthetic marker") {
		t.Fatal("detector result must contain no credential or source context")
	}
	if len(engine.Scan("ordinary code 550e8400-e29b-41d4-a716-446655440000 password=example")) != 0 {
		t.Fatal("ordinary synthetic content should not trigger a rule")
	}
}

func TestCredentialFamilies(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, text, rule, severity string }{
		{"private key", "-----BEGIN PRIVATE KEY-----\n" + strings.Repeat("U3ludGhldGljLW5vbmZ1bmN0aW9uYWw=\n", 4) + "-----END PRIVATE KEY-----", "private-key", "high"},
		{"access key identifier", "AKIA" + "7FJ29KLMN4PQR8ST", "aws-access-token", "medium"},
		{"generic assignment", `api_key = "t7Fk29LmN4pQr8StU1vWx5YzA0bCd3Ef"`, "generic-api-key", "medium"},
	} {
		t.Run(test.name, func(t *testing.T) {
			matches := engine.Scan(test.text)
			found := false
			for _, match := range matches {
				found = found || (match.RuleID == test.rule && match.Severity == test.severity)
			}
			if !found {
				t.Fatal("seeded credential family was not detected with the expected classification")
			}
		})
	}
}
