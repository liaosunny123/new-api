package dto

type GroupRateLimitOverride struct {
	RPM         *int `json:"rpm,omitempty"`
	Concurrency *int `json:"concurrency,omitempty"`
}

type UserSetting struct {
	NotifyType                       string  `json:"notify_type,omitempty"`                          // QuotaWarningType 额度预警类型
	QuotaWarningThreshold            float64 `json:"quota_warning_threshold,omitempty"`              // QuotaWarningThreshold 额度预警阈值
	WebhookUrl                       string  `json:"webhook_url,omitempty"`                          // WebhookUrl webhook地址
	WebhookSecret                    string  `json:"webhook_secret,omitempty"`                       // WebhookSecret webhook密钥
	NotificationEmail                string  `json:"notification_email,omitempty"`                   // NotificationEmail 通知邮箱地址
	BarkUrl                          string  `json:"bark_url,omitempty"`                             // BarkUrl Bark推送URL
	GotifyUrl                        string  `json:"gotify_url,omitempty"`                           // GotifyUrl Gotify服务器地址
	GotifyToken                      string  `json:"gotify_token,omitempty"`                         // GotifyToken Gotify应用令牌
	GotifyPriority                   int     `json:"gotify_priority"`                                // GotifyPriority Gotify消息优先级
	UpstreamModelUpdateNotifyEnabled bool    `json:"upstream_model_update_notify_enabled,omitempty"` // 是否接收上游模型更新定时检测通知（仅管理员）
	AcceptUnsetRatioModel            bool    `json:"accept_unset_model_ratio_model,omitempty"`       // AcceptUnsetRatioModel 是否接受未设置价格的模型
	RecordIpLog                      bool    `json:"record_ip_log,omitempty"`                        // 是否记录请求和错误日志IP
	SidebarModules                   string  `json:"sidebar_modules,omitempty"`                      // SidebarModules 左侧边栏模块配置
	BillingPreference                string  `json:"billing_preference,omitempty"`                   // BillingPreference 扣费策略（订阅/钱包）
	Language                         string                          `json:"language,omitempty"`                             // Language 用户语言偏好 (zh, en)
	RateLimitOverrides               map[string]GroupRateLimitOverride `json:"rate_limit_overrides,omitempty"`                 // 用户级别的分组RPM/并发限制覆盖
	GroupRatioEnabled                bool                              `json:"group_ratio_enabled,omitempty"`                  // 分组价格设置总开关，关闭时所有分组倍率覆盖均不生效
	GroupRatioOverrides              map[string]float64                `json:"group_ratio_overrides,omitempty"`                // 用户级别的分组倍率覆盖：分组名 -> 倍率，优先级高于系统分组倍率
	HedgeEnabled                     bool                              `json:"hedge_enabled,omitempty"`                        // HedgeEnabled 请求对冲总开关（仅 Claude /v1/messages 生效），关闭时走原串行重试逻辑
	HedgeFirstResponseTimeout        int                               `json:"hedge_first_response_timeout"`                   // HedgeFirstResponseTimeout 首响应超时（秒），某路超时未出响应头则并行发起下一路；默认 15
	HedgeMaxAttempts                 int                               `json:"hedge_max_attempts"`                             // HedgeMaxAttempts 最大并行尝试路数；默认 5
	FirstByteTimeout                 int                               `json:"first_byte_timeout"`                             // FirstByteTimeout 上游首字节（首个响应头）硬超时（秒），0=未设置用系统默认；生效值取 max(系统默认, 此值)，上限 3000
}

const (
	// HedgeDefaultFirstResponseTimeout 首响应超时默认值（秒）
	HedgeDefaultFirstResponseTimeout = 15
	// HedgeMinFirstResponseTimeout / HedgeMaxFirstResponseTimeout 首响应超时可配置范围（秒）
	HedgeMinFirstResponseTimeout = 1
	HedgeMaxFirstResponseTimeout = 600
	// HedgeDefaultMaxAttempts 最大尝试路数默认值
	HedgeDefaultMaxAttempts = 5
	// HedgeMinMaxAttempts / HedgeMaxMaxAttempts 最大尝试路数可配置范围
	HedgeMinMaxAttempts = 1
	HedgeMaxMaxAttempts = 10

	// FirstByteMax / FirstByteMin 用户可自定义的首字节超时范围（秒）
	FirstByteMax = 3000
	FirstByteMin = 1
)

// GetFirstByteTimeout 返回归一化后的用户首字节超时（秒）：
// 0 或越界（<1）视为"未设置"返回 0（调用方回退到系统默认）；超过上限则截断到上限。
func (s UserSetting) GetFirstByteTimeout() int {
	if s.FirstByteTimeout < FirstByteMin {
		return 0
	}
	if s.FirstByteTimeout > FirstByteMax {
		return FirstByteMax
	}
	return s.FirstByteTimeout
}

// GetHedgeFirstResponseTimeout 返回归一化后的首响应超时（秒），缺省/越界时回退到默认值。
func (s UserSetting) GetHedgeFirstResponseTimeout() int {
	if s.HedgeFirstResponseTimeout < HedgeMinFirstResponseTimeout || s.HedgeFirstResponseTimeout > HedgeMaxFirstResponseTimeout {
		return HedgeDefaultFirstResponseTimeout
	}
	return s.HedgeFirstResponseTimeout
}

// GetHedgeMaxAttempts 返回归一化后的最大尝试路数，缺省/越界时回退到默认值。
func (s UserSetting) GetHedgeMaxAttempts() int {
	if s.HedgeMaxAttempts < HedgeMinMaxAttempts || s.HedgeMaxAttempts > HedgeMaxMaxAttempts {
		return HedgeDefaultMaxAttempts
	}
	return s.HedgeMaxAttempts
}

// GetGroupRatioOverride 返回该用户针对指定分组的倍率覆盖。
// 仅当总开关开启且为该分组配置了倍率时返回 (ratio, true)，否则返回 (0, false)。
// 所有计费路径都应通过此方法读取覆盖，确保行为一致。
func (s UserSetting) GetGroupRatioOverride(group string) (float64, bool) {
	if !s.GroupRatioEnabled || s.GroupRatioOverrides == nil {
		return 0, false
	}
	ratio, ok := s.GroupRatioOverrides[group]
	return ratio, ok
}

var (
	NotifyTypeEmail   = "email"   // Email 邮件
	NotifyTypeWebhook = "webhook" // Webhook
	NotifyTypeBark    = "bark"    // Bark 推送
	NotifyTypeGotify  = "gotify"  // Gotify 推送
)
