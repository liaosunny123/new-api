package setting

import (
	"fmt"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

type GroupRateLimitConfig struct {
	RPM         int `json:"rpm"`
	Concurrency int `json:"concurrency"`
}

var GroupRPMConcurrencyLimit = map[string]GroupRateLimitConfig{}
var GroupGroupRPMConcurrencyLimit = map[string]map[string]GroupRateLimitConfig{}
var RateLimitExceededMessage = ""
var groupRateLimitMutex sync.RWMutex

func GroupRPMConcurrencyLimit2JSONString() string {
	groupRateLimitMutex.RLock()
	defer groupRateLimitMutex.RUnlock()

	jsonBytes, err := common.Marshal(GroupRPMConcurrencyLimit)
	if err != nil {
		common.SysLog("error marshalling GroupRPMConcurrencyLimit: " + err.Error())
		return "{}"
	}
	return string(jsonBytes)
}

func UpdateGroupRPMConcurrencyLimitByJSONString(jsonStr string) error {
	groupRateLimitMutex.Lock()
	defer groupRateLimitMutex.Unlock()

	GroupRPMConcurrencyLimit = make(map[string]GroupRateLimitConfig)
	return common.UnmarshalJsonStr(jsonStr, &GroupRPMConcurrencyLimit)
}

func CheckGroupRPMConcurrencyLimit(jsonStr string) error {
	var check map[string]GroupRateLimitConfig
	err := common.UnmarshalJsonStr(jsonStr, &check)
	if err != nil {
		return err
	}
	for group, cfg := range check {
		if cfg.RPM < 0 {
			return fmt.Errorf("group %s has negative RPM value: %d", group, cfg.RPM)
		}
		if cfg.Concurrency < 0 {
			return fmt.Errorf("group %s has negative concurrency value: %d", group, cfg.Concurrency)
		}
	}
	return nil
}

func GetGroupRPMConcurrencyConfig(group string) (GroupRateLimitConfig, bool) {
	groupRateLimitMutex.RLock()
	defer groupRateLimitMutex.RUnlock()

	cfg, found := GroupRPMConcurrencyLimit[group]
	return cfg, found
}

func GroupGroupRPMConcurrencyLimit2JSONString() string {
	groupRateLimitMutex.RLock()
	defer groupRateLimitMutex.RUnlock()

	jsonBytes, err := common.Marshal(GroupGroupRPMConcurrencyLimit)
	if err != nil {
		common.SysLog("error marshalling GroupGroupRPMConcurrencyLimit: " + err.Error())
		return "{}"
	}
	return string(jsonBytes)
}

func UpdateGroupGroupRPMConcurrencyLimitByJSONString(jsonStr string) error {
	groupRateLimitMutex.Lock()
	defer groupRateLimitMutex.Unlock()

	GroupGroupRPMConcurrencyLimit = make(map[string]map[string]GroupRateLimitConfig)
	return common.UnmarshalJsonStr(jsonStr, &GroupGroupRPMConcurrencyLimit)
}

func CheckGroupGroupRPMConcurrencyLimit(jsonStr string) error {
	var check map[string]map[string]GroupRateLimitConfig
	err := common.UnmarshalJsonStr(jsonStr, &check)
	if err != nil {
		return err
	}
	for userGroup, inner := range check {
		for channelGroup, cfg := range inner {
			if cfg.RPM < 0 {
				return fmt.Errorf("group %s -> %s has negative RPM value: %d", userGroup, channelGroup, cfg.RPM)
			}
			if cfg.Concurrency < 0 {
				return fmt.Errorf("group %s -> %s has negative concurrency value: %d", userGroup, channelGroup, cfg.Concurrency)
			}
		}
	}
	return nil
}

func GetGroupGroupRPMConcurrencyConfig(userGroup, channelGroup string) (GroupRateLimitConfig, bool) {
	groupRateLimitMutex.RLock()
	defer groupRateLimitMutex.RUnlock()

	inner, found := GroupGroupRPMConcurrencyLimit[userGroup]
	if !found {
		return GroupRateLimitConfig{}, false
	}
	cfg, found := inner[channelGroup]
	return cfg, found
}

// ResolveRPMConcurrencyLimit resolves effective RPM and concurrency limits
// Priority: user override > group-group override > group default > no limit (0)
func ResolveRPMConcurrencyLimit(userGroup, usingGroup string, userSetting dto.UserSetting) (rpm, concurrency int) {
	// 1. Check user-level override
	if userSetting.RateLimitOverrides != nil {
		if override, found := userSetting.RateLimitOverrides[usingGroup]; found {
			if override.RPM != nil {
				rpm = *override.RPM
			}
			if override.Concurrency != nil {
				concurrency = *override.Concurrency
			}
			// If user has override for this group, both fields are authoritative
			// (nil means "use lower level default" for that specific field)
			rpmResolved := override.RPM != nil
			concResolved := override.Concurrency != nil
			if rpmResolved && concResolved {
				return rpm, concurrency
			}
			// Partially resolved - continue to fill in missing fields
			if !rpmResolved {
				rpm = 0
			}
			if !concResolved {
				concurrency = 0
			}
			// Fill from lower levels
			lowerRpm, lowerConc := resolveFromGroupSettings(userGroup, usingGroup)
			if !rpmResolved {
				rpm = lowerRpm
			}
			if !concResolved {
				concurrency = lowerConc
			}
			return rpm, concurrency
		}
	}

	// 2. Check group-group and group defaults
	return resolveFromGroupSettings(userGroup, usingGroup)
}

// GetAllGroupsWithRateLimits returns all group names that have any rate limit
// configured for the given user group, including:
// 1. Groups with basic GroupRPMConcurrencyLimit
// 2. Channel groups from GroupGroupRPMConcurrencyLimit[userGroup]
// 3. Groups from user-level overrides
func GetAllGroupsWithRateLimits(userGroup string, userSetting dto.UserSetting) map[string]struct{} {
	groups := make(map[string]struct{})

	groupRateLimitMutex.RLock()
	// Basic group limits
	for group := range GroupRPMConcurrencyLimit {
		groups[group] = struct{}{}
	}
	// Group-group limits for this user's group
	if inner, found := GroupGroupRPMConcurrencyLimit[userGroup]; found {
		for channelGroup := range inner {
			groups[channelGroup] = struct{}{}
		}
	}
	groupRateLimitMutex.RUnlock()

	// User-level overrides
	if userSetting.RateLimitOverrides != nil {
		for group := range userSetting.RateLimitOverrides {
			groups[group] = struct{}{}
		}
	}

	return groups
}

func resolveFromGroupSettings(userGroup, usingGroup string) (rpm, concurrency int) {
	// Check group-group override
	if cfg, found := GetGroupGroupRPMConcurrencyConfig(userGroup, usingGroup); found {
		return cfg.RPM, cfg.Concurrency
	}

	// Check group default
	if cfg, found := GetGroupRPMConcurrencyConfig(usingGroup); found {
		return cfg.RPM, cfg.Concurrency
	}

	return 0, 0
}
