package apistation

import "testing"

func TestDetectVersionDrift(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		isDrift  bool
		severity string
	}{
		{"same version", "1.0.29", "1.0.29", false, "none"},
		{"patch drift", "1.0.29", "1.0.31", true, "minor"},
		{"minor drift", "1.0.29", "1.1.0", true, "major"},
		{"major drift", "1.0.29", "2.0.0", true, "major"},
		{"empty current", "", "1.0.29", false, "none"},
		{"empty latest", "1.0.29", "", false, "none"},
		{"both empty", "", "", false, "none"},
		{"current ahead patch", "1.0.31", "1.0.29", false, "none"},
		{"current ahead minor", "1.1.0", "1.0.31", false, "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectVersionDrift(tt.current, tt.latest)
			if result.IsDrift != tt.isDrift {
				t.Errorf("IsDrift = %v, want %v", result.IsDrift, tt.isDrift)
			}
			if result.Severity != tt.severity {
				t.Errorf("Severity = %q, want %q", result.Severity, tt.severity)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.0.29", [3]int{1, 0, 29}},
		{"v2.1.0", [3]int{2, 1, 0}},
		{"1.0.29-beta", [3]int{1, 0, 29}},
		{"", [3]int{0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVersion(tt.input)
			if got != tt.want {
				t.Errorf("parseVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
