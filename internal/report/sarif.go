package report

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/meysam81/scry/internal/model"
)

// SARIFReporter writes the CrawlResult issues in SARIF 2.1.0 format.
type SARIFReporter struct{}

// Name returns "sarif".
func (r *SARIFReporter) Name() string { return "sarif" }

// sarifDocument is the top-level SARIF 2.1.0 envelope.
type sarifDocument struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string                    `json:"id"`
	ShortDescription     sarifMessage              `json:"shortDescription"`
	DefaultConfiguration sarifDefaultConfiguration `json:"defaultConfiguration"`
}

type sarifDefaultConfiguration struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

// severityToSARIFLevel maps model.Severity to the SARIF level string.
func severityToSARIFLevel(s model.Severity) string {
	switch s {
	case model.SeverityCritical:
		return "error"
	case model.SeverityWarning:
		return "warning"
	case model.SeverityInfo:
		return "note"
	default:
		return "none"
	}
}

// Write renders result as SARIF 2.1.0 JSON and writes it to w.
func (r *SARIFReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}

	// Deduplicate rules: each unique CheckName becomes one rule entry.
	// Preserve insertion order via a slice plus a seen map.
	type ruleEntry struct {
		id       string
		severity model.Severity
	}
	seen := make(map[string]struct{})
	var ruleOrder []ruleEntry
	for _, iss := range result.Issues {
		if _, ok := seen[iss.CheckName]; ok {
			continue
		}
		seen[iss.CheckName] = struct{}{}
		ruleOrder = append(ruleOrder, ruleEntry{id: iss.CheckName, severity: iss.Severity})
	}

	rules := make([]sarifRule, 0, len(ruleOrder))
	for _, re := range ruleOrder {
		rules = append(rules, sarifRule{
			ID:               re.id,
			ShortDescription: sarifMessage{Text: re.id},
			DefaultConfiguration: sarifDefaultConfiguration{
				Level: severityToSARIFLevel(re.severity),
			},
		})
	}

	results := make([]sarifResult, 0, len(result.Issues))
	for _, iss := range result.Issues {
		msg := iss.Message
		if iss.Detail != "" {
			msg = msg + ": " + iss.Detail
		}
		results = append(results, sarifResult{
			RuleID:  iss.CheckName,
			Level:   severityToSARIFLevel(iss.Severity),
			Message: sarifMessage{Text: msg},
			Locations: []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: iss.URL},
					},
				},
			},
		})
	}

	doc := sarifDocument{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "scry",
						Version:        "1.0.0",
						InformationURI: "https://github.com/meysam81/scry",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling SARIF document: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing SARIF output: %w", err)
	}

	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing trailing newline: %w", err)
	}

	return nil
}
