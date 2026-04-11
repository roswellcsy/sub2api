package apistation

import (
	"encoding/json"
	"math"
	"time"
)

// FailureKind 错误分类，对应不同的退避策略
type FailureKind string

const (
	FailureRateLimit FailureKind = "rate_limit"
	FailureAuth      FailureKind = "auth"
	FailureForbidden FailureKind = "forbidden"
	FailureServer    FailureKind = "server"
	FailureNetwork   FailureKind = "network"
)

// CooldownTier 单个错误类型的退避参数
type CooldownTier struct {
	BaseMs int64 `json:"base_ms"`
	MaxMs  int64 `json:"max_ms"`
}

// CooldownConfig 完整的分级退避配置
type CooldownConfig struct {
	RateLimit CooldownTier `json:"rate_limit"`
	Auth      CooldownTier `json:"auth"`
	Forbidden CooldownTier `json:"forbidden"`
	Server    CooldownTier `json:"server"`
	Network   CooldownTier `json:"network"`
}

// DefaultCooldownConfig 默认配置，来自 auth2api manager.ts lines 16-25
var DefaultCooldownConfig = CooldownConfig{
	RateLimit: CooldownTier{BaseMs: 60_000, MaxMs: 900_000},    // 60s base, 15min max
	Auth:      CooldownTier{BaseMs: 600_000, MaxMs: 3_600_000}, // 10min base, 60min max
	Forbidden: CooldownTier{BaseMs: 600_000, MaxMs: 3_600_000}, // 10min base, 60min max
	Server:    CooldownTier{BaseMs: 5_000, MaxMs: 300_000},     // 5s base, 5min max
	Network:   CooldownTier{BaseMs: 5_000, MaxMs: 300_000},     // 5s base, 5min max
}

// ParseCooldownConfig 从 JSON 字符串解析配置，解析失败返回默认值
func ParseCooldownConfig(jsonStr string) CooldownConfig {
	if jsonStr == "" {
		return DefaultCooldownConfig
	}
	var cfg CooldownConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		return DefaultCooldownConfig
	}
	return cfg
}

// ComputeCooldown 计算分级退避时长
// 公式: min(base * 2^(failures-1), max)
func ComputeCooldown(kind FailureKind, failures int, cfg CooldownConfig) time.Duration {
	tier := getTier(kind, cfg)
	if failures <= 0 {
		failures = 1
	}
	backoffMs := float64(tier.BaseMs) * math.Pow(2, float64(failures-1))
	if backoffMs > float64(tier.MaxMs) {
		backoffMs = float64(tier.MaxMs)
	}
	return time.Duration(int64(backoffMs)) * time.Millisecond
}

// StatusCodeToFailureKind 将 HTTP 状态码映射到错误分类
func StatusCodeToFailureKind(statusCode int) FailureKind {
	switch {
	case statusCode == 401:
		return FailureAuth
	case statusCode == 403:
		return FailureForbidden
	case statusCode == 429:
		return FailureRateLimit
	case statusCode == 529:
		return FailureServer
	case statusCode >= 500:
		return FailureServer
	default:
		return FailureNetwork
	}
}

func getTier(kind FailureKind, cfg CooldownConfig) CooldownTier {
	switch kind {
	case FailureRateLimit:
		return cfg.RateLimit
	case FailureAuth:
		return cfg.Auth
	case FailureForbidden:
		return cfg.Forbidden
	case FailureServer:
		return cfg.Server
	case FailureNetwork:
		return cfg.Network
	default:
		return cfg.Server
	}
}
