package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type PaymentSetting struct {
	AmountOptions  []int           `json:"amount_options"`
	AmountDiscount map[int]float64 `json:"amount_discount"` // 充值金额对应的折扣，例如 100 元 0.9 表示 100 元充值享受 9 折优惠

	SingleTopUpLimitEnabled bool    `json:"single_topup_limit_enabled"`
	SingleTopUpLimitAmount  float64 `json:"single_topup_limit_amount"`
	SingleTopUpLimitMessage string  `json:"single_topup_limit_message"`

	DailyTopUpLimitEnabled bool    `json:"daily_topup_limit_enabled"`
	DailyTopUpLimitAmount  float64 `json:"daily_topup_limit_amount"`
	DailyTopUpLimitMessage string  `json:"daily_topup_limit_message"`
}

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:  []int{10, 20, 50, 100, 200, 500},
	AmountDiscount: map[int]float64{},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}
