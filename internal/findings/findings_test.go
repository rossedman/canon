package findings

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParseSeverity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    Severity
		wantErr bool
	}{
		{"", Warning, false},
		{"error", Error, false},
		{"warning", Warning, false},
		{"suggestion", Suggestion, false},
		{"fatal", "", true},
	}
	for _, tt := range tests {
		got, err := ParseSeverity(tt.in)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("ParseSeverity(%q) = %q, %v", tt.in, got, err)
		}
	}
}

func TestAtLeast(t *testing.T) {
	t.Parallel()
	if !Error.AtLeast(Warning) || Suggestion.AtLeast(Warning) || !Warning.AtLeast(Warning) {
		t.Fatal("severity ordering is wrong")
	}
	if Severity("bogus").AtLeast(Suggestion) {
		t.Fatal("unknown severity ranks above suggestion")
	}
}

// sample returns findings in no particular order, with one duplicate.
func sample() []Finding {
	return []Finding{
		{Rule: "r2", Path: "b.go", Line: 3, Severity: Warning, Message: "m"},
		{Rule: "r1", Path: "a.go", Line: 9, Severity: Suggestion, Message: "m"},
		{Rule: "r1", Path: "a.go", Line: 2, Severity: Error, Message: "m"},
		{Rule: "r1", Path: "a.go", Line: 2, Severity: Error, Message: "m"},
		{Rule: "r0", Path: "a.go", Line: 2, Severity: Warning, Message: "other"},
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	got := Normalize(sample())
	if len(got) != 4 {
		t.Fatalf("got %d findings, want 4 after dedupe", len(got))
	}
	want := []string{"a.go:2 r1", "a.go:2 r0", "a.go:9 r1", "b.go:3 r2"}
	for i, f := range got {
		if s := fmt.Sprintf("%s:%d %s", f.Path, f.Line, f.Rule); s != want[i] {
			t.Errorf("finding %d = %s, want %s", i, s, want[i])
		}
	}
}

func TestFilterAndSummarize(t *testing.T) {
	t.Parallel()
	fs := Normalize(sample())
	if n := len(Filter(fs, Warning)); n != 3 {
		t.Fatalf("Filter(warning) kept %d, want 3", n)
	}
	s := Summarize(fs)
	if s != (Summary{Error: 1, Warning: 2, Suggestion: 1}) || s.Total() != 4 {
		t.Fatalf("Summarize = %+v", s)
	}
}

func TestSARIF(t *testing.T) {
	t.Parallel()
	fs := []Finding{
		{Rule: "r1", Record: "rec", Severity: Error, Path: "Dockerfile", Line: 3, Column: 1, Message: "bad", Why: "because"},
		{Rule: "r2", Record: "rec", Severity: Suggestion, Path: "LICENSE", Message: "missing"},
		{Rule: "r3", Record: "rec", Severity: Warning, Path: "x", Line: 1, Message: "warn"},
	}
	rules := []RuleInfo{{ID: "r2", Record: "rec", Description: "two"}, {ID: "r1", Record: "rec", Description: "one", Why: "because"}}
	b, err := SARIF(fs, rules, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	var log struct {
		Version string
		Runs    []struct {
			Tool struct {
				Driver struct {
					Version string
					Rules   []struct{ ID string }
				}
			}
			Results []struct {
				RuleID    string
				Level     string
				Message   struct{ Text string }
				Locations []struct {
					PhysicalLocation struct {
						Region *struct{ StartLine int }
					}
				}
			}
		}
	}
	if err := json.Unmarshal(b, &log); err != nil {
		t.Fatal(err)
	}
	run := log.Runs[0]
	if log.Version != "2.1.0" || run.Tool.Driver.Version != "1.0.0" {
		t.Fatalf("bad header: %+v", log)
	}
	if run.Tool.Driver.Rules[0].ID != "r1" {
		t.Fatalf("rules not sorted: %+v", run.Tool.Driver.Rules)
	}
	levels := []string{run.Results[0].Level, run.Results[1].Level, run.Results[2].Level}
	if levels[0] != "error" || levels[1] != "note" || levels[2] != "warning" {
		t.Fatalf("levels = %v", levels)
	}
	if run.Results[0].Message.Text != "bad (rec: because)" {
		t.Fatalf("message = %q", run.Results[0].Message.Text)
	}
	if run.Results[1].Locations[0].PhysicalLocation.Region != nil {
		t.Fatal("finding without a line should have no region")
	}
}

func BenchmarkNormalize(b *testing.B) {
	var fs []Finding
	for i := range 2000 {
		fs = append(fs, Finding{Rule: fmt.Sprintf("r%d", i%7), Path: fmt.Sprintf("f%d.go", i%50), Line: i % 300, Severity: Warning, Message: "m"})
	}
	b.ReportAllocs()
	for b.Loop() {
		Normalize(fs)
	}
}
