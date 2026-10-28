package findings

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
)

// RuleInfo describes a rule for the SARIF tool driver, so code scanning UIs
// can show the reason next to each result.
type RuleInfo struct {
	ID          string
	Record      string
	Description string
	Why         string
}

// sarifLog is the subset of SARIF 2.1.0 that canon writes.
type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

// sarifRun is one tool invocation.
type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

// sarifTool wraps the driver.
type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

// sarifDriver names the tool and lists its rules.
type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

// sarifRule is a reportingDescriptor.
type sarifRule struct {
	ID               string         `json:"id"`
	ShortDescription sarifText      `json:"shortDescription"`
	FullDescription  sarifText      `json:"fullDescription"`
	Properties       map[string]any `json:"properties,omitempty"`
}

// sarifText is a message object.
type sarifText struct {
	Text string `json:"text"`
}

// sarifResult is one finding.
type sarifResult struct {
	RuleID     string          `json:"ruleId"`
	Level      string          `json:"level"`
	Message    sarifText       `json:"message"`
	Locations  []sarifLocation `json:"locations"`
	Properties map[string]any  `json:"properties,omitempty"`
}

// sarifLocation wraps a physical location.
type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

// sarifPhysical points at a file and region.
type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           *sarifRegion  `json:"region,omitempty"`
}

// sarifArtifact is a file URI relative to the repo root.
type sarifArtifact struct {
	URI string `json:"uri"`
}

// sarifRegion is a line and column, both 1-based.
type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}

// level maps a severity to a SARIF result level.
func (s Severity) level() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	}
	return "note"
}

// SARIF encodes findings as a SARIF 2.1.0 log. rules describes every rule a
// finding may reference; rules without findings are still listed so the log
// documents what was checked.
func SARIF(fs []Finding, rules []RuleInfo, version string) ([]byte, error) {
	driver := sarifDriver{
		Name:           "canon",
		Version:        version,
		InformationURI: "https://github.com/rossedman/canon",
		Rules:          []sarifRule{},
	}
	sorted := slices.Clone(rules)
	slices.SortFunc(sorted, func(a, b RuleInfo) int { return cmp.Compare(a.ID, b.ID) })
	for _, r := range sorted {
		driver.Rules = append(driver.Rules, sarifRule{
			ID:               r.ID,
			ShortDescription: sarifText{Text: r.Description},
			FullDescription:  sarifText{Text: r.Why},
			Properties:       map[string]any{"record": r.Record},
		})
	}
	results := make([]sarifResult, 0, len(fs))
	for _, f := range fs {
		loc := sarifPhysical{ArtifactLocation: sarifArtifact{URI: f.Path}}
		if f.Line > 0 {
			loc.Region = &sarifRegion{StartLine: f.Line, StartColumn: f.Column}
		}
		msg := f.Message
		if f.Why != "" {
			msg = fmt.Sprintf("%s (%s: %s)", f.Message, f.Record, f.Why)
		}
		results = append(results, sarifResult{
			RuleID:     f.Rule,
			Level:      f.Severity.level(),
			Message:    sarifText{Text: msg},
			Locations:  []sarifLocation{{PhysicalLocation: loc}},
			Properties: map[string]any{"record": f.Record},
		})
	}
	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs:    []sarifRun{{Tool: sarifTool{Driver: driver}, Results: results}},
	}
	b, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode sarif: %w", err)
	}
	return b, nil
}
