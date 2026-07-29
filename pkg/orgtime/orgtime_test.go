package orgtime

import (
	"strings"
	"testing"
	"time"
)

func TestParse_ISO8601(t *testing.T) {
	anchor := time.Date(2026, 1, 8, 12, 0, 0, 0, time.Local)

	tests := []struct {
		input   string
		wantY   int
		wantM   time.Month
		wantD   int
		wantH   int
		wantMin int
		hasTime bool
	}{
		{"2026-01-15", 2026, 1, 15, 0, 0, false},
		{"2026-01-15T14:30", 2026, 1, 15, 14, 30, true},
		{"2025-12-25", 2025, 12, 25, 0, 0, false},
	}

	for _, tt := range tests {
		ts, err := Parse(tt.input, anchor)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if ts.Time.Year() != tt.wantY || ts.Time.Month() != tt.wantM || ts.Time.Day() != tt.wantD {
			t.Errorf("Parse(%q) date = %v, want %d-%02d-%02d", tt.input, ts.Time, tt.wantY, tt.wantM, tt.wantD)
		}
		if ts.HasTime != tt.hasTime {
			t.Errorf("Parse(%q) HasTime = %v, want %v", tt.input, ts.HasTime, tt.hasTime)
		}
		if tt.hasTime && (ts.Time.Hour() != tt.wantH || ts.Time.Minute() != tt.wantMin) {
			t.Errorf("Parse(%q) time = %02d:%02d, want %02d:%02d", tt.input, ts.Time.Hour(), ts.Time.Minute(), tt.wantH, tt.wantMin)
		}
	}
}

func TestParse_Relative(t *testing.T) {
	anchor := time.Date(2026, 1, 8, 12, 0, 0, 0, time.Local)

	tests := []struct {
		input string
		wantY int
		wantM time.Month
		wantD int
	}{
		{"+1d", 2026, 1, 9},
		{"+3d", 2026, 1, 11},
		{"-1d", 2026, 1, 7},
		{"+1w", 2026, 1, 15},
		{"+2w", 2026, 1, 22},
		{"+1m", 2026, 2, 8},
		{"-1m", 2025, 12, 8},
		{"+1y", 2027, 1, 8},
	}

	for _, tt := range tests {
		ts, err := Parse(tt.input, anchor)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if ts.Time.Year() != tt.wantY || ts.Time.Month() != tt.wantM || ts.Time.Day() != tt.wantD {
			t.Errorf("Parse(%q) = %v, want %d-%02d-%02d", tt.input, ts.Time, tt.wantY, tt.wantM, tt.wantD)
		}
	}
}

func TestParse_Keywords(t *testing.T) {
	anchor := time.Date(2026, 1, 8, 12, 0, 0, 0, time.Local)

	tests := []struct {
		input string
		wantD int
	}{
		{"today", 8},
		{"tomorrow", 9},
		{"yesterday", 7},
	}

	for _, tt := range tests {
		ts, err := Parse(tt.input, anchor)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if ts.Time.Day() != tt.wantD {
			t.Errorf("Parse(%q) day = %d, want %d", tt.input, ts.Time.Day(), tt.wantD)
		}
	}
}

func TestParse_OrgTimestamp(t *testing.T) {
	anchor := time.Date(2026, 1, 8, 12, 0, 0, 0, time.Local)

	tests := []struct {
		input  string
		wantY  int
		wantM  time.Month
		wantD  int
		active bool
	}{
		{"<2026-01-15 Thu>", 2026, 1, 15, true},
		{"[2026-01-15 Thu]", 2026, 1, 15, false},
		{"<2026-01-15 Thu 14:30>", 2026, 1, 15, true},
	}

	for _, tt := range tests {
		ts, err := Parse(tt.input, anchor)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.input, err)
			continue
		}
		if ts.Time.Year() != tt.wantY || ts.Time.Month() != tt.wantM || ts.Time.Day() != tt.wantD {
			t.Errorf("Parse(%q) date = %v, want %d-%02d-%02d", tt.input, ts.Time, tt.wantY, tt.wantM, tt.wantD)
		}
		if ts.Active != tt.active {
			t.Errorf("Parse(%q) Active = %v, want %v", tt.input, ts.Active, tt.active)
		}
	}
}

func TestTimestamp_Format(t *testing.T) {
	ts := Timestamp{
		Time:    time.Date(2026, 1, 15, 14, 30, 0, 0, time.Local),
		HasTime: true,
		Active:  true,
	}

	active := ts.Format(true)
	if !strings.HasPrefix(active, "<") || !strings.HasSuffix(active, ">") {
		t.Errorf("Format(true) = %q, want angle brackets", active)
	}
	if !strings.Contains(active, "2026-01-15") {
		t.Errorf("Format(true) = %q, missing date", active)
	}
	if !strings.Contains(active, "14:30") {
		t.Errorf("Format(true) = %q, missing time", active)
	}

	inactive := ts.Format(false)
	if !strings.HasPrefix(inactive, "[") || !strings.HasSuffix(inactive, "]") {
		t.Errorf("Format(false) = %q, want square brackets", inactive)
	}
}

func TestFormatStateChange(t *testing.T) {
	ts := Timestamp{
		Time:    time.Date(2026, 1, 8, 14, 30, 0, 0, time.Local),
		HasTime: true,
	}

	result := FormatStateChange("DONE", "TODO", ts)
	if !strings.Contains(result, "DONE") {
		t.Error("should contain DONE")
	}
	if !strings.Contains(result, "TODO") {
		t.Error("should contain TODO")
	}
	if !strings.Contains(result, "State") {
		t.Error("should contain State")
	}
	if !strings.Contains(result, "from") {
		t.Error("should contain from")
	}
}

func TestParse_TimezoneFromAnchor(t *testing.T) {
	singapore, _ := time.LoadLocation("Asia/Singapore")
	utc := time.UTC

	anchorSG := time.Date(2026, 1, 8, 12, 0, 0, 0, singapore)
	anchorUTC := time.Date(2026, 1, 8, 12, 0, 0, 0, utc)

	tsSG, err := Parse("2026-01-15", anchorSG)
	if err != nil {
		t.Fatalf("Parse with SG timezone failed: %v", err)
	}
	if tsSG.Time.Location() != singapore {
		t.Errorf("expected Singapore timezone, got %v", tsSG.Time.Location())
	}

	tsUTC, err := Parse("2026-01-15", anchorUTC)
	if err != nil {
		t.Fatalf("Parse with UTC timezone failed: %v", err)
	}
	if tsUTC.Time.Location() != utc {
		t.Errorf("expected UTC timezone, got %v", tsUTC.Time.Location())
	}

	tsRelSG, _ := Parse("+1d", anchorSG)
	if tsRelSG.Time.Location() != singapore {
		t.Errorf("relative date should inherit anchor timezone")
	}

	tsOrgSG, _ := Parse("<2026-01-15 Thu>", anchorSG)
	if tsOrgSG.Time.Location() != singapore {
		t.Errorf("org timestamp should use anchor timezone")
	}
}
