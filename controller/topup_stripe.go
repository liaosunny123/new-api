package controller

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/webhook"
	"github.com/thanhpk/randstr"
)

const (
	PaymentMethodStripe = "stripe"
)

var stripeAdaptor = &StripeAdaptor{}

// StripePayRequest represents a payment request for Stripe checkout.
type StripePayRequest struct {
	// Amount is the quantity of units to purchase.
	Amount int64 `json:"amount"`
	// PaymentMethod specifies the payment method (e.g., "stripe").
	PaymentMethod string `json:"payment_method"`
	// SuccessURL is the optional custom URL to redirect after successful payment.
	// If empty, defaults to the server's console log page.
	SuccessURL string `json:"success_url,omitempty"`
	// CancelURL is the optional custom URL to redirect when payment is canceled.
	// If empty, defaults to the server's console topup page.
	CancelURL string `json:"cancel_url,omitempty"`
}

type StripeAdaptor struct {
}

func (*StripeAdaptor) RequestAmount(c *gin.Context, req *StripePayRequest) {
	if req.Amount < getStripeMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getStripeMinTopup())})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getStripePayMoney(float64(req.Amount), group)
	if payMoney <= 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	if msg := checkStripeAmountLimit(payMoney); msg != "" {
		c.JSON(200, gin.H{"message": "error", "data": msg})
		return
	}
	c.JSON(200, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func (*StripeAdaptor) RequestPay(c *gin.Context, req *StripePayRequest) {
	if req.PaymentMethod != PaymentMethodStripe {
		c.JSON(200, gin.H{"message": "error", "data": "不支持的支付渠道"})
		return
	}
	if !setting.StripeEnabled {
		c.JSON(200, gin.H{"message": "error", "data": "管理员未开启 Stripe 支付"})
		return
	}
	if req.Amount < getStripeMinTopup() {
		c.JSON(200, gin.H{"message": fmt.Sprintf("充值数量不能小于 %d", getStripeMinTopup()), "data": 10})
		return
	}
	if req.Amount > 10000 {
		c.JSON(200, gin.H{"message": "充值数量不能大于 10000", "data": 10})
		return
	}
	if msg := checkSingleTopUpLimit(req.Amount); msg != "" {
		c.JSON(200, gin.H{"message": "error", "data": msg})
		return
	}

	id := c.GetInt("id")
	if msg := checkDailyTopUpLimit(id); msg != "" {
		c.JSON(200, gin.H{"message": "error", "data": msg})
		return
	}

	if req.SuccessURL != "" && common.ValidateRedirectURL(req.SuccessURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "支付成功重定向URL不在可信任域名列表中", "data": ""})
		return
	}

	if req.CancelURL != "" && common.ValidateRedirectURL(req.CancelURL) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "支付取消重定向URL不在可信任域名列表中", "data": ""})
		return
	}

	user, _ := model.GetUserById(id, false)
	chargedMoney := GetChargedAmount(float64(req.Amount), *user)

	// 计算 CNY 充值金额并做金额限额校验（min/max 针对 CNY 充值金额）
	group, gErr := model.GetUserGroup(id, true)
	if gErr != nil {
		group = user.Group
	}
	payMoney := getStripePayMoney(float64(req.Amount), group)
	if msg := checkStripeAmountLimit(payMoney); msg != "" {
		c.JSON(200, gin.H{"message": "error", "data": msg})
		return
	}
	// 换算为扣款货币并计算（gross-up）处理费
	chargeAmount, feeAmount, currency := stripeCharge(payMoney)

	reference := fmt.Sprintf("new-api-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "ref_" + common.Sha1([]byte(reference))

	payLink, err := genStripeLink(referenceId, user.StripeCustomer, user.Email, chargeAmount, feeAmount, currency, req.SuccessURL, req.CancelURL)
	if err != nil {
		log.Println("获取Stripe Checkout支付链接失败", err)
		c.JSON(200, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}

	topUp := &model.TopUp{
		UserId:        id,
		Amount:        req.Amount,
		Money:         chargedMoney,
		TradeNo:       referenceId,
		PaymentMethod: PaymentMethodStripe,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	c.JSON(200, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payLink,
		},
	})
}

func RequestStripeAmount(c *gin.Context) {
	var req StripePayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	stripeAdaptor.RequestAmount(c, &req)
}

func RequestStripePay(c *gin.Context) {
	var req StripePayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	stripeAdaptor.RequestPay(c, &req)
}

func StripeWebhook(c *gin.Context) {
	if setting.StripeWebhookSecret == "" {
		log.Println("Stripe Webhook Secret 未配置，拒绝处理")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("解析Stripe Webhook参数失败: %v\n", err)
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	event, err := webhook.ConstructEventWithOptions(payload, signature, setting.StripeWebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})

	if err != nil {
		log.Printf("Stripe Webhook验签失败: %v\n", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		sessionCompleted(event)
	case stripe.EventTypeCheckoutSessionExpired:
		sessionExpired(event)
	case stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		sessionAsyncPaymentSucceeded(event)
	case stripe.EventTypeCheckoutSessionAsyncPaymentFailed:
		sessionAsyncPaymentFailed(event)
	default:
		log.Printf("不支持的Stripe Webhook事件类型: %s\n", event.Type)
	}

	c.Status(http.StatusOK)
}

func sessionCompleted(event stripe.Event) {
	customerId := event.GetObjectValue("customer")
	referenceId := event.GetObjectValue("client_reference_id")
	status := event.GetObjectValue("status")
	if "complete" != status {
		log.Println("错误的Stripe Checkout完成状态:", status, ",", referenceId)
		return
	}

	paymentStatus := event.GetObjectValue("payment_status")
	if paymentStatus != "paid" {
		log.Printf("Stripe Checkout 支付尚未完成，payment_status: %s, ref: %s（等待异步支付结果）", paymentStatus, referenceId)
		return
	}

	fulfillOrder(event, referenceId, customerId)
}

// sessionAsyncPaymentSucceeded handles delayed payment methods (bank transfer, SEPA, etc.)
// that confirm payment after the checkout session completes.
func sessionAsyncPaymentSucceeded(event stripe.Event) {
	customerId := event.GetObjectValue("customer")
	referenceId := event.GetObjectValue("client_reference_id")
	log.Printf("Stripe 异步支付成功: %s", referenceId)

	fulfillOrder(event, referenceId, customerId)
}

// sessionAsyncPaymentFailed marks orders as failed when delayed payment methods
// ultimately fail (e.g. bank transfer not received, SEPA rejected).
func sessionAsyncPaymentFailed(event stripe.Event) {
	referenceId := event.GetObjectValue("client_reference_id")
	log.Printf("Stripe 异步支付失败: %s", referenceId)

	if len(referenceId) == 0 {
		log.Println("异步支付失败事件未提供支付单号")
		return
	}

	LockOrder(referenceId)
	defer UnlockOrder(referenceId)

	topUp := model.GetTopUpByTradeNo(referenceId)
	if topUp == nil {
		log.Println("异步支付失败，充值订单不存在:", referenceId)
		return
	}

	if topUp.PaymentMethod != PaymentMethodStripe {
		log.Printf("异步支付失败，订单支付方式不匹配: %s, ref: %s", topUp.PaymentMethod, referenceId)
		return
	}

	if topUp.Status != common.TopUpStatusPending {
		log.Printf("异步支付失败，订单状态非pending: %s, ref: %s", topUp.Status, referenceId)
		return
	}

	topUp.Status = common.TopUpStatusFailed
	if err := topUp.Update(); err != nil {
		log.Printf("标记充值订单失败出错: %v, ref: %s", err, referenceId)
		return
	}
	log.Printf("充值订单已标记为失败: %s", referenceId)
}

// fulfillOrder is the shared logic for crediting quota after payment is confirmed.
func fulfillOrder(event stripe.Event, referenceId string, customerId string) {
	if len(referenceId) == 0 {
		log.Println("未提供支付单号")
		return
	}

	LockOrder(referenceId)
	defer UnlockOrder(referenceId)
	payload := map[string]any{
		"customer":     customerId,
		"amount_total": event.GetObjectValue("amount_total"),
		"currency":     strings.ToUpper(event.GetObjectValue("currency")),
		"event_type":   string(event.Type),
	}
	if err := model.CompleteSubscriptionOrder(referenceId, common.GetJsonString(payload)); err == nil {
		return
	} else if err != nil && !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
		log.Println("complete subscription order failed:", err.Error(), referenceId)
		return
	}

	err := model.Recharge(referenceId, customerId)
	if err != nil {
		log.Println(err.Error(), referenceId)
		return
	}

	total, _ := strconv.ParseFloat(event.GetObjectValue("amount_total"), 64)
	currency := strings.ToUpper(event.GetObjectValue("currency"))
	log.Printf("收到款项：%s, %.2f(%s)", referenceId, total/100, currency)
}

func sessionExpired(event stripe.Event) {
	referenceId := event.GetObjectValue("client_reference_id")
	status := event.GetObjectValue("status")
	if "expired" != status {
		log.Println("错误的Stripe Checkout过期状态:", status, ",", referenceId)
		return
	}

	if len(referenceId) == 0 {
		log.Println("未提供支付单号")
		return
	}

	// Subscription order expiration
	LockOrder(referenceId)
	defer UnlockOrder(referenceId)
	if err := model.ExpireSubscriptionOrder(referenceId); err == nil {
		return
	} else if err != nil && !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
		log.Println("过期订阅订单失败", referenceId, ", err:", err.Error())
		return
	}

	topUp := model.GetTopUpByTradeNo(referenceId)
	if topUp == nil {
		log.Println("充值订单不存在", referenceId)
		return
	}

	if topUp.Status != common.TopUpStatusPending {
		log.Println("充值订单状态错误", referenceId)
	}

	topUp.Status = common.TopUpStatusExpired
	err := topUp.Update()
	if err != nil {
		log.Println("过期充值订单失败", referenceId, ", err:", err.Error())
		return
	}

	log.Println("充值订单已过期", referenceId)
}

// genStripeLink 生成 Stripe Checkout 会话链接（直接按金额生成订单，不依赖 Price ID）。
//
// 参数：
//   - referenceId: 交易唯一引用号
//   - customerId:  已有的 Stripe Customer ID（为空表示新客户，按邮箱创建）
//   - email:       新客户使用的邮箱
//   - chargeAmount: 扣款货币下的产品金额（已按汇率换算）
//   - feeAmount:    扣款货币下的处理费金额（0 表示不加收）
//   - currency:    扣款货币（小写）
//   - successURL/cancelURL: 自定义跳转地址（为空用默认）
func genStripeLink(referenceId string, customerId string, email string, chargeAmount float64, feeAmount float64, currency string, successURL string, cancelURL string) (string, error) {
	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		return "", fmt.Errorf("无效的Stripe API密钥")
	}

	stripe.Key = setting.StripeApiSecret

	// Use custom URLs if provided, otherwise use defaults
	if successURL == "" {
		successURL = system_setting.ServerAddress + "/console/log"
	}
	if cancelURL == "" {
		cancelURL = system_setting.ServerAddress + "/console/topup"
	}

	// 按金额内联生成 line item（无需 Price ID）
	lineItems := []*stripe.CheckoutSessionLineItemParams{
		{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(currency),
				UnitAmount: stripe.Int64(stripeToMinor(chargeAmount, currency)),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String("余额充值 / Account Top-up"),
				},
			},
		},
	}
	// 处理费作为独立行展示（金额已 gross-up 算好）
	if feeAmount > 0 {
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(currency),
				UnitAmount: stripe.Int64(stripeToMinor(feeAmount, currency)),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String("Stripe 处理费 / Stripe Processing Fee"),
				},
			},
		})
	}

	params := &stripe.CheckoutSessionParams{
		ClientReferenceID:   stripe.String(referenceId),
		SuccessURL:          stripe.String(successURL),
		CancelURL:           stripe.String(cancelURL),
		LineItems:           lineItems,
		Mode:                stripe.String(string(stripe.CheckoutSessionModePayment)),
		AllowPromotionCodes: stripe.Bool(setting.StripePromotionCodesEnabled),
	}

	if "" == customerId {
		if "" != email {
			params.CustomerEmail = stripe.String(email)
		}

		params.CustomerCreation = stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways))
	} else {
		params.Customer = stripe.String(customerId)
	}

	result, err := session.New(params)
	if err != nil {
		return "", err
	}

	return result.URL, nil
}

// stripeZeroDecimalCurrencies 不使用最小货币单位（不 ×100）的货币
var stripeZeroDecimalCurrencies = map[string]bool{
	"bif": true, "clp": true, "djf": true, "gnf": true, "jpy": true,
	"kmf": true, "krw": true, "mga": true, "pyg": true, "rwf": true,
	"ugx": true, "vnd": true, "vuv": true, "xaf": true, "xof": true, "xpf": true,
}

// stripeToMinor 将金额换算成 Stripe 使用的最小货币单位
func stripeToMinor(amount float64, currency string) int64 {
	if stripeZeroDecimalCurrencies[strings.ToLower(currency)] {
		return int64(amount + 0.5)
	}
	return int64(amount*100 + 0.5)
}

// stripeRound2 四舍五入到分
func stripeRound2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// stripeCharge 把 CNY 充值金额换算为扣款货币，并按 gross-up 计算处理费。
// 返回：产品金额、处理费金额（均为扣款货币）、货币代码。
func stripeCharge(payMoneyCNY float64) (product float64, fee float64, currency string) {
	currency = strings.ToLower(setting.StripeCurrency)
	if currency == "" {
		currency = "usd"
	}
	rate := setting.StripeExchangeRate
	if currency == "cny" || rate <= 0 {
		rate = 1
	}
	product = stripeRound2(payMoneyCNY * rate)

	if setting.StripeFeeEnabled {
		r := setting.StripeFeePercent / 100
		if r > 0 && r < 1 {
			// gross-up: charge=(product+fixed)/(1-r)，fee=charge-product，使商家到手≈product
			fee = stripeRound2((product+setting.StripeFeeFixed)/(1-r) - product)
		} else {
			fee = stripeRound2(setting.StripeFeeFixed)
		}
	}
	return
}

// checkStripeAmountLimit 校验 CNY 充值金额是否在配置的 [min, max] 内（0 表示不限）。
// 超限返回给用户的提示文案，正常返回空串。
func checkStripeAmountLimit(payMoneyCNY float64) string {
	if (setting.StripeMinAmount > 0 && payMoneyCNY < setting.StripeMinAmount) ||
		(setting.StripeMaxAmount > 0 && payMoneyCNY > setting.StripeMaxAmount) {
		return "我们暂时无法为您设定的金额发起 Stripe 支付，请使用其他支付方式或联系管理员"
	}
	return ""
}

func GetChargedAmount(count float64, user model.User) float64 {
	topUpGroupRatio := common.GetTopupGroupRatio(user.Group)
	if topUpGroupRatio == 0 {
		topUpGroupRatio = 1
	}

	return count * topUpGroupRatio
}

func getStripePayMoney(amount float64, group string) float64 {
	originalAmount := amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = amount / common.QuotaPerUnit
	}
	// Using float64 for monetary calculations is acceptable here due to the small amounts involved
	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}
	// apply optional preset discount by the original request amount (if configured), default 1.0
	discount := 1.0
	if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(originalAmount)]; ok {
		if ds > 0 {
			discount = ds
		}
	}
	payMoney := amount * setting.StripeUnitPrice * topupGroupRatio * discount
	return payMoney
}

func getStripeMinTopup() int64 {
	minTopup := setting.StripeMinTopUp
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		minTopup = minTopup * int(common.QuotaPerUnit)
	}
	return int64(minTopup)
}
