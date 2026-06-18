package setting

var StripeApiSecret = ""
var StripeWebhookSecret = ""
var StripePriceId = ""
var StripeUnitPrice = 8.0
var StripeMinTopUp = 1
var StripePromotionCodesEnabled = false

// 独立开关：是否开启 Stripe 支付（不再依赖是否填写 Price ID）
var StripeEnabled = false

// 扣款货币（小写，如 usd/hkd/cny）
var StripeCurrency = "usd"

// 汇率：1 CNY = ? 扣款货币；货币为 cny 时按 1 处理（不换算）
var StripeExchangeRate = 1.0

// 支付金额限额（按 CNY 充值金额判断），0 表示不限制
var StripeMinAmount = 0.0
var StripeMaxAmount = 0.0

// 是否向客户加收 Stripe 处理费（gross-up，使商家到手≈原价）
var StripeFeeEnabled = false

// 处理费率(%)与每笔固定处理费（扣款货币）
var StripeFeePercent = 0.0
var StripeFeeFixed = 0.0
