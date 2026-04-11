package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apistation"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	apistationMonitorDefaultInterval          = 5 * time.Minute
	apistationMonitorInitialDelay             = 30 * time.Second
	apistationMonitorSettingsTimeout          = 2 * time.Second
	apistationMonitorCheckTimeout             = 10 * time.Second
	apistationMonitorRecentAccountLogLimit    = 200
	apistationMonitorDefaultBanAlertThreshold = 3
)

// ApistationMonitorService runs background checks for account ban warnings
// and CC version drift, sending alerts to Feishu webhook.
type ApistationMonitorService struct {
	opsService     *OpsService
	settingService *SettingService
	feishu         *apistation.FeishuWebhookClient

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
}

func NewApistationMonitorService(
	opsService *OpsService,
	settingService *SettingService,
) *ApistationMonitorService {
	return &ApistationMonitorService{
		opsService:     opsService,
		settingService: settingService,
		feishu:         apistation.NewFeishuWebhookClient(),
	}
}

func (s *ApistationMonitorService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		if s.stopCh == nil {
			s.stopCh = make(chan struct{})
		}
		if s.feishu == nil {
			s.feishu = apistation.NewFeishuWebhookClient()
		}
		s.wg.Add(1)
		go s.run()
	})
}

func (s *ApistationMonitorService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
	s.wg.Wait()
}

func (s *ApistationMonitorService) run() {
	defer s.wg.Done()

	timer := time.NewTimer(apistationMonitorInitialDelay)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			s.checkOnce()
			timer.Reset(s.getCheckInterval())
		case <-s.stopCh:
			return
		}
	}
}

func (s *ApistationMonitorService) getCheckInterval() time.Duration {
	secs, ok := s.getPositiveIntSetting(SettingKeyMonitorCheckInterval)
	if !ok {
		return apistationMonitorDefaultInterval
	}
	return time.Duration(secs) * time.Second
}

func (s *ApistationMonitorService) getWebhookURL() string {
	val, err := s.getSettingValue(SettingKeyFeishuWebhookURL)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val)
}

func (s *ApistationMonitorService) checkOnce() {
	s.checkBanWarnings()
	s.checkVersionDrift()
}

// checkBanWarnings queries recent account_state_change events for consecutive auth failures.
func (s *ApistationMonitorService) checkBanWarnings() {
	if s == nil || s.opsService == nil {
		return
	}

	webhookURL := s.getWebhookURL()
	if webhookURL == "" {
		return
	}

	now := time.Now().UTC()
	oneHourAgo := now.Add(-1 * time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), apistationMonitorCheckTimeout)
	defer cancel()

	logs, err := s.opsService.ListSystemLogs(ctx, &OpsSystemLogFilter{
		StartTime: &oneHourAgo,
		EndTime:   &now,
		Query:     "account_state_change",
		Page:      1,
		PageSize:  apistationMonitorRecentAccountLogLimit,
	})
	if err != nil {
		logger.LegacyPrintf("service.apistation_monitor", "[warn] failed to query account state changes: %v", err)
		return
	}

	threshold, ok := s.getPositiveIntSetting(SettingKeyBanAlertThreshold)
	if !ok {
		threshold = apistationMonitorDefaultBanAlertThreshold
	}

	accountFailures := make(map[int64]int)
	accountSequenceClosed := make(map[int64]bool)
	for _, log := range logs.Logs {
		event, ok := accountStateEventFromSystemLog(log)
		if !ok || event.AccountID <= 0 || accountSequenceClosed[event.AccountID] {
			continue
		}
		if isBanWarningEvent(event) {
			accountFailures[event.AccountID]++
			continue
		}
		accountSequenceClosed[event.AccountID] = true
	}

	for accountID, failures := range accountFailures {
		if failures < threshold {
			continue
		}
		content := fmt.Sprintf(
			"**Account**: %d\n**Consecutive failures**: %d (threshold: %d)\n**Window**: last 1 hour\n**Action**: Check account status and consider disabling",
			accountID,
			failures,
			threshold,
		)
		if alertErr := s.feishu.SendAlert(ctx, webhookURL, "[API Station] Ban Warning", content, "red"); alertErr != nil {
			logger.LegacyPrintf("service.apistation_monitor", "[warn] failed to send ban warning alert: %v", alertErr)
		}
	}
}

// checkVersionDrift compares the configured CC version against the latest known version.
func (s *ApistationMonitorService) checkVersionDrift() {
	webhookURL := s.getWebhookURL()
	if webhookURL == "" {
		return
	}

	currentVersion, err := s.getSettingValue(SettingKeyCLIVersion)
	if err != nil || strings.TrimSpace(currentVersion) == "" {
		return
	}

	latestVersion, err := s.getSettingValue(SettingKeyLatestKnownCLIVersion)
	if err != nil || strings.TrimSpace(latestVersion) == "" {
		return
	}

	result := apistation.DetectVersionDrift(currentVersion, latestVersion)
	if !result.IsDrift || result.Severity != "major" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), apistationMonitorCheckTimeout)
	defer cancel()

	content := fmt.Sprintf(
		"**Current version**: %s\n**Latest version**: %s\n**Severity**: %s\n**Message**: %s\n**Action**: Update `%s` setting",
		result.CurrentVersion,
		result.LatestVersion,
		result.Severity,
		result.Message,
		SettingKeyCLIVersion,
	)
	if alertErr := s.feishu.SendAlert(ctx, webhookURL, "[API Station] CC Version Drift", content, "orange"); alertErr != nil {
		logger.LegacyPrintf("service.apistation_monitor", "[warn] failed to send version drift alert: %v", alertErr)
	}
}

func (s *ApistationMonitorService) getSettingValue(key string) (string, error) {
	if s == nil || s.settingService == nil || s.settingService.settingRepo == nil {
		return "", ErrSettingNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), apistationMonitorSettingsTimeout)
	defer cancel()
	return s.settingService.settingRepo.GetValue(ctx, key)
}

func (s *ApistationMonitorService) getPositiveIntSetting(key string) (int, bool) {
	val, err := s.getSettingValue(key)
	if err != nil {
		return 0, false
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func accountStateEventFromSystemLog(log *OpsSystemLog) (apistation.AccountStateEvent, bool) {
	var event apistation.AccountStateEvent
	if log == nil || len(log.Extra) == 0 {
		return event, false
	}

	rawEvent, ok := log.Extra["event"]
	if !ok {
		return event, false
	}

	switch v := rawEvent.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &event); err != nil {
			return event, false
		}
	case map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return event, false
		}
		if err := json.Unmarshal(data, &event); err != nil {
			return event, false
		}
	default:
		return event, false
	}

	if event.AccountID <= 0 && log.AccountID != nil {
		event.AccountID = *log.AccountID
	}
	return event, event.AccountID > 0
}

func isBanWarningEvent(event apistation.AccountStateEvent) bool {
	trigger := strings.ToLower(strings.TrimSpace(event.Trigger))
	reason := strings.ToLower(strings.TrimSpace(event.Reason))

	switch trigger {
	case "auth", "401", "403":
		return true
	}

	return strings.Contains(reason, "401") || strings.Contains(reason, "403")
}
