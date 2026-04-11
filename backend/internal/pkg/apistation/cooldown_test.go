package apistation

import (
	"testing"
	"time"
)

func TestComputeCooldown(t *testing.T) {
	cfg := DefaultCooldownConfig

	tests := []struct {
		name     string
		kind     FailureKind
		failures int
		wantMin  time.Duration
		wantMax  time.Duration
	}{
		{"rate_limit first", FailureRateLimit, 1, 60 * time.Second, 60 * time.Second},
		{"rate_limit second", FailureRateLimit, 2, 120 * time.Second, 120 * time.Second},
		{"rate_limit capped", FailureRateLimit, 20, 15 * time.Minute, 15 * time.Minute},
		{"auth first", FailureAuth, 1, 10 * time.Minute, 10 * time.Minute},
		{"auth third", FailureAuth, 3, 40 * time.Minute, 40 * time.Minute},
		{"server first", FailureServer, 1, 5 * time.Second, 5 * time.Second},
		{"server fifth", FailureServer, 5, 80 * time.Second, 80 * time.Second},
		{"zero failures defaults to 1", FailureRateLimit, 0, 60 * time.Second, 60 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeCooldown(tt.kind, tt.failures, cfg)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("ComputeCooldown(%s, %d) = %v, want [%v, %v]", tt.kind, tt.failures, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestStatusCodeToFailureKind(t *testing.T) {
	tests := []struct {
		code int
		want FailureKind
	}{
		{401, FailureAuth},
		{403, FailureForbidden},
		{429, FailureRateLimit},
		{529, FailureServer},
		{500, FailureServer},
		{502, FailureServer},
		{0, FailureNetwork},
	}
	for _, tt := range tests {
		got := StatusCodeToFailureKind(tt.code)
		if got != tt.want {
			t.Errorf("StatusCodeToFailureKind(%d) = %s, want %s", tt.code, got, tt.want)
		}
	}
}

func TestParseCooldownConfig(t *testing.T) {
	// Valid JSON
	cfg := ParseCooldownConfig(`{"rate_limit":{"base_ms":30000,"max_ms":60000}}`)
	if cfg.RateLimit.BaseMs != 30000 {
		t.Errorf("parsed rate_limit.base_ms = %d, want 30000", cfg.RateLimit.BaseMs)
	}

	// Empty string → default
	cfg2 := ParseCooldownConfig("")
	if cfg2.RateLimit.BaseMs != 60000 {
		t.Errorf("empty string should return default, got rate_limit.base_ms = %d", cfg2.RateLimit.BaseMs)
	}

	// Invalid JSON → default
	cfg3 := ParseCooldownConfig("not json")
	if cfg3.RateLimit.BaseMs != 60000 {
		t.Errorf("invalid JSON should return default, got rate_limit.base_ms = %d", cfg3.RateLimit.BaseMs)
	}
}
