package apistation

import (
	"fmt"
	"strconv"
	"strings"
)

// VersionDriftResult holds the result of a version drift check.
type VersionDriftResult struct {
	CurrentVersion string
	LatestVersion  string
	IsDrift        bool
	Severity       string // "none", "minor", "major"
	Message        string
}

// DetectVersionDrift compares the configured CLI version against the known latest.
// Returns drift info. Both versions should be semver-like strings (e.g., "1.0.29").
func DetectVersionDrift(current, latest string) VersionDriftResult {
	result := VersionDriftResult{
		CurrentVersion: current,
		LatestVersion:  latest,
		Severity:       "none",
	}
	if current == "" || latest == "" || current == latest {
		return result
	}

	curParts := parseVersion(current)
	latParts := parseVersion(latest)

	if compareVersionParts(latParts, curParts) <= 0 {
		return result
	}

	result.IsDrift = true
	switch {
	case latParts[0] > curParts[0]:
		result.Severity = "major"
		result.Message = fmt.Sprintf("Major version drift: %s -> %s", current, latest)
	case latParts[1] > curParts[1]:
		result.Severity = "major"
		result.Message = fmt.Sprintf("Minor version drift: %s -> %s", current, latest)
	default:
		result.Severity = "minor"
		result.Message = fmt.Sprintf("Patch version drift: %s -> %s", current, latest)
	}
	return result
}

func parseVersion(v string) [3]int {
	var parts [3]int
	segments := strings.SplitN(v, ".", 3)
	for i, s := range segments {
		if i >= len(parts) {
			break
		}
		numStr := strings.TrimLeft(s, "vV")
		for j, c := range numStr {
			if c < '0' || c > '9' {
				numStr = numStr[:j]
				break
			}
		}
		parts[i], _ = strconv.Atoi(numStr)
	}
	return parts
}

func compareVersionParts(left, right [3]int) int {
	for i := 0; i < len(left); i++ {
		switch {
		case left[i] > right[i]:
			return 1
		case left[i] < right[i]:
			return -1
		}
	}
	return 0
}
