package detect

import (
	"errors"
	"sort"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"github.com/zricethezav/gitleaks/v8/config"
	leaks "github.com/zricethezav/gitleaks/v8/detect"
	"github.com/zricethezav/gitleaks/v8/logging"
)

func init() {
	logging.Logger = zerolog.Nop()
}

type Match struct {
	RuleID     string
	Severity   string
	Confidence string
}

type Engine struct {
	detector *leaks.Detector
}

func New() (*Engine, error) {
	settings := viper.New()
	settings.SetConfigType("toml")
	if err := settings.ReadConfig(strings.NewReader(config.DefaultConfig)); err != nil {
		return nil, errors.New("could not initialize embedded detection rules")
	}
	var source config.ViperConfig
	if err := settings.Unmarshal(&source); err != nil {
		return nil, errors.New("could not initialize embedded detection rules")
	}
	rules, err := source.Translate()
	if err != nil {
		return nil, errors.New("could not initialize embedded detection rules")
	}
	detector := leaks.NewDetector(rules)
	detector.IgnoreGitleaksAllow = true
	detector.Redact = 100
	return &Engine{detector: detector}, nil
}

func (engine *Engine) Scan(text string) []Match {
	findings := engine.detector.DetectString(text)
	matches := make([]Match, 0, len(findings))
	for _, finding := range findings {
		severity, confidence := "high", "high"
		if finding.RuleID == "generic-api-key" {
			severity, confidence = "medium", "low"
		}
		if finding.RuleID == "aws-access-token" {
			severity, confidence = "medium", "high"
		}
		matches = append(matches, Match{finding.RuleID, severity, confidence})
	}
	sort.SliceStable(matches, func(left, right int) bool {
		return matches[left].RuleID < matches[right].RuleID
	})
	return matches
}
