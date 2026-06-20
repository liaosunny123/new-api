package service

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// imageProxyPayload 是加密进 /assets/<token> 路径里的载荷。
// U 为上游原始 URL，C 为渠道 ID（拉取时复用渠道代理）。
type imageProxyPayload struct {
	U string `json:"u"`
	C int    `json:"c"`
}

// IsImageProxyEnabled 返回图片 URL 代理功能是否开启。
func IsImageProxyEnabled() bool {
	return model_setting.GetGlobalSettings().ImageProxyEnabled
}

// EncodeImageProxyToken 将上游 URL 与渠道 ID 加密成不可读的 token。
func EncodeImageProxyToken(originalURL string, channelId int) (string, error) {
	data, err := common.Marshal(imageProxyPayload{U: originalURL, C: channelId})
	if err != nil {
		return "", err
	}
	return common.AESGCMEncrypt(data)
}

// ParseImageProxyToken 解密 token，返回上游 URL 与渠道 ID。
func ParseImageProxyToken(token string) (string, int, error) {
	data, err := common.AESGCMDecrypt(token)
	if err != nil {
		return "", 0, err
	}
	var payload imageProxyPayload
	if err = common.Unmarshal(data, &payload); err != nil {
		return "", 0, err
	}
	if strings.TrimSpace(payload.U) == "" {
		return "", 0, errors.New("empty url in image proxy token")
	}
	return payload.U, payload.C, nil
}

// BuildImageProxyURL 在功能开启且 URL 为非本站 http(s) 链接时，返回本站代理地址
// （scheme://<request-host>/assets/<token>）。否则返回 ok=false 表示无需改写。
func BuildImageProxyURL(c *gin.Context, originalURL string, channelId int) (string, bool) {
	if c == nil || c.Request == nil || !IsImageProxyEnabled() {
		return "", false
	}
	originalURL = strings.TrimSpace(originalURL)
	if originalURL == "" {
		return "", false
	}
	parsed, err := url.Parse(originalURL)
	if err != nil || parsed.Host == "" {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	if isLocalHost(c, parsed.Host) {
		return "", false
	}
	host := c.Request.Host
	if host == "" {
		return "", false
	}
	token, err := EncodeImageProxyToken(originalURL, channelId)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%s://%s/file-assets/%s", detectRequestScheme(c.Request), host, token), true
}

// RewriteImageResponseBody 改写 OpenAI 格式图片响应里 data[].url 的非本站链接。
// 使用 gjson/sjson 以保留 usage、created、b64_json 等其他字段。无改动或解析失败时原样返回。
func RewriteImageResponseBody(c *gin.Context, channelId int, body []byte) []byte {
	if len(body) == 0 || !IsImageProxyEnabled() {
		return body
	}
	dataArr := gjson.GetBytes(body, "data")
	if !dataArr.IsArray() {
		return body
	}
	out := body
	changed := false
	dataArr.ForEach(func(idx, item gjson.Result) bool {
		u := item.Get("url").String()
		if u == "" {
			return true
		}
		proxyURL, ok := BuildImageProxyURL(c, u, channelId)
		if !ok {
			return true
		}
		if nb, err := sjson.SetBytes(out, fmt.Sprintf("data.%d.url", idx.Int()), proxyURL); err == nil {
			out = nb
			changed = true
		}
		return true
	})
	if !changed {
		return body
	}
	return out
}

// isLocalHost 判断 host 是否指向本站（请求 Host 或配置的 ServerAddress）。
func isLocalHost(c *gin.Context, host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	candidates := []string{strings.ToLower(c.Request.Host)}
	if sa := strings.TrimSpace(system_setting.ServerAddress); sa != "" {
		if u, err := url.Parse(sa); err == nil && u.Host != "" {
			candidates = append(candidates, strings.ToLower(u.Host))
		}
	}
	hostNoPort := stripPort(host)
	for _, cand := range candidates {
		if cand == "" {
			continue
		}
		if cand == host || stripPort(cand) == hostNoPort {
			return true
		}
	}
	return false
}

func stripPort(host string) string {
	if !strings.Contains(host, ":") {
		return host
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// detectRequestScheme 推断请求的 scheme，优先 X-Forwarded-Proto，再 TLS，默认 http。
func detectRequestScheme(r *http.Request) string {
	if r == nil {
		return "http"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		parts := strings.Split(proto, ",")
		return strings.ToLower(strings.TrimSpace(parts[0]))
	}
	if r.TLS != nil {
		return "https"
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return strings.ToLower(r.URL.Scheme)
	}
	return "http"
}
