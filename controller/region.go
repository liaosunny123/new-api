package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/geoip"

	"github.com/gin-gonic/gin"
)

// candidateIPs returns possible client IPs in priority order, preferring the
// real client IP forwarded by proxies/CDNs over the immediate peer address.
func candidateIPs(c *gin.Context) []string {
	ips := make([]string, 0, 6)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		// X-Forwarded-For may be "client, proxy1, proxy2" — keep the leftmost (original client).
		if idx := strings.Index(v, ","); idx >= 0 {
			v = strings.TrimSpace(v[:idx])
		}
		ips = append(ips, v)
	}
	// Cloudflare / common proxy headers first.
	add(c.GetHeader("CF-Connecting-IP"))
	add(c.GetHeader("True-Client-IP"))
	add(c.GetHeader("X-Real-IP"))
	add(c.GetHeader("X-Forwarded-For"))
	// Gin's resolved client IP (respects trusted proxies) and raw peer last.
	add(c.ClientIP())
	add(c.RemoteIP())
	return ips
}

// GetRegion resolves the caller's country via GeoIP and reports whether they are
// in mainland China. The frontend uses this to gate region-restricted groups;
// the backend itself keeps serving all groups regardless.
//
// Supports an optional ?ip= override for testing.
func GetRegion(c *gin.Context) {
	var country string
	if override := strings.TrimSpace(c.Query("ip")); override != "" {
		country = geoip.CountryCode(override)
	} else {
		// Try each candidate IP; use the first that resolves to a country
		// (private/loopback IPs resolve to empty and are skipped).
		for _, ip := range candidateIPs(c) {
			if cc := geoip.CountryCode(ip); cc != "" {
				country = cc
				break
			}
		}
	}
	common.ApiSuccess(c, gin.H{
		"country":     country,
		"is_mainland": country == "CN",
	})
}
