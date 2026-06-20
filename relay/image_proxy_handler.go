package relay

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// ImageProxyPathPrefix 是图片代理路径前缀（避免与前端 /assets/ 静态资源冲突）。
const ImageProxyPathPrefix = "/file-assets/"

// TryImageProxy 处理 /file-assets/<token> 形式的图片代理请求：解析 token 得到上游 URL，
// 经 SSRF 校验后从上游拉取并流式回传，从而隐藏上游来源。
//
// 返回 true 表示本次请求已由图片代理处理（无论成功失败）；返回 false 表示这不是一个
// 合法的图片代理 token，调用方应继续走原有逻辑（如返回 404 / 前端静态资源）。
func TryImageProxy(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet || !service.IsImageProxyEnabled() {
		return false
	}
	token := strings.TrimPrefix(c.Request.URL.Path, ImageProxyPathPrefix)
	if token == "" || strings.Contains(token, "/") {
		return false
	}
	targetURL, channelId, err := service.ParseImageProxyToken(token)
	if err != nil {
		return false
	}

	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(targetURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": fmt.Sprintf("request blocked: %v", err),
		})
		return true
	}

	resp, err := imageProxyHttpClient(channelId).Get(targetURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "fetch_upstream_image_failed",
		})
		return true
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.Status(resp.StatusCode)
		_, _ = io.Copy(c.Writer, resp.Body)
		return true
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		log.Println("Failed to stream proxied image:", err)
	}
	return true
}

// imageProxyHttpClient 优先使用渠道配置的代理拉取上游图片，否则使用默认客户端。
func imageProxyHttpClient(channelId int) *http.Client {
	if channelId > 0 {
		if ch, err := model.CacheGetChannel(channelId); err == nil {
			if proxy := ch.GetSetting().Proxy; proxy != "" {
				if hc, err := service.NewProxyHttpClient(proxy); err == nil {
					return hc
				}
			}
		}
	}
	return service.GetHttpClient()
}
