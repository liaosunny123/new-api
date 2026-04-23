package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

type GroupRateLimitUsage struct {
	RPMCurrent         int64 `json:"rpm_current"`
	RPMLimit           int   `json:"rpm_limit"`
	ConcurrencyCurrent int64 `json:"concurrency_current"`
	ConcurrencyLimit   int   `json:"concurrency_limit"`
}

func GetUserRateLimitUsage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data, err := getUserRateLimitUsageById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func GetSelfRateLimitUsage(c *gin.Context) {
	id := c.GetInt("id")
	data, err := getUserRateLimitUsageById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}

func getUserRateLimitUsageById(id int) (map[string]GroupRateLimitUsage, error) {
	user, err := model.GetUserById(id, false)
	if err != nil {
		return nil, err
	}

	userSetting := user.GetSetting()
	userGroup := user.Group

	candidateGroups := make(map[string]struct{})
	for g := range service.GetUserUsableGroups(userGroup) {
		candidateGroups[g] = struct{}{}
	}
	for g := range setting.GetAllGroupsWithRateLimits(userGroup, userSetting) {
		candidateGroups[g] = struct{}{}
	}

	result := make(map[string]GroupRateLimitUsage)

	for groupName := range candidateGroups {
		rpmLimit, concLimit := setting.ResolveRPMConcurrencyLimit(userGroup, groupName, userSetting)
		if rpmLimit <= 0 && concLimit <= 0 {
			continue
		}

		rpmKey := fmt.Sprintf("groupRpm:%d:%s", id, groupName)
		concKey := fmt.Sprintf("groupConcurrency:%d:%s", id, groupName)

		usage := GroupRateLimitUsage{
			RPMLimit:         rpmLimit,
			ConcurrencyLimit: concLimit,
		}
		if rpmLimit > 0 {
			usage.RPMCurrent = middleware.GetSlidingWindowCount(rpmKey)
		}
		if concLimit > 0 {
			usage.ConcurrencyCurrent = middleware.GetSlidingWindowCount(concKey)
		}
		result[groupName] = usage
	}
	return result, nil
}

func GetUserRateLimitOverrides(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	userSetting, err := model.GetUserSetting(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	overrides := userSetting.RateLimitOverrides
	if overrides == nil {
		overrides = make(map[string]dto.GroupRateLimitOverride)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    overrides,
	})
}

type UpdateRateLimitOverridesRequest struct {
	Overrides map[string]dto.GroupRateLimitOverride `json:"overrides"`
}

func UpdateUserRateLimitOverrides(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req UpdateRateLimitOverridesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid request body",
		})
		return
	}

	// Validate override values
	for group, override := range req.Overrides {
		if override.RPM != nil && *override.RPM < 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": fmt.Sprintf("group %s has negative RPM value", group),
			})
			return
		}
		if override.Concurrency != nil && *override.Concurrency < 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": fmt.Sprintf("group %s has negative concurrency value", group),
			})
			return
		}
	}

	// Get current user setting
	userSetting, err := model.GetUserSetting(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// Update overrides
	userSetting.RateLimitOverrides = req.Overrides

	// Save back to user
	user, err := model.GetUserById(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user.SetSetting(userSetting)
	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
