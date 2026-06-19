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
