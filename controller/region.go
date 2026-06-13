package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/geoip"

	"github.com/gin-gonic/gin"
)

// GetRegion resolves the caller's country via GeoIP and reports whether they are
// in mainland China. The frontend uses this to gate region-restricted groups;
// the backend itself keeps serving all groups regardless.
//
// Supports an optional ?ip= override for testing.
func GetRegion(c *gin.Context) {
	ip := c.ClientIP()
	if override := c.Query("ip"); override != "" {
		ip = override
	}
	country := geoip.CountryCode(ip)
	common.ApiSuccess(c, gin.H{
		"country":     country,
		"is_mainland": country == "CN",
	})
}
