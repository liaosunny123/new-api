package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetUserGroupRatioOverrides 返回指定用户的分组倍率覆盖配置（分组价格设置）
func GetUserGroupRatioOverrides(c *gin.Context) {
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

	overrides := userSetting.GroupRatioOverrides
	if overrides == nil {
		overrides = make(map[string]float64)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled":   userSetting.GroupRatioEnabled,
			"overrides": overrides,
		},
	})
}

type UpdateGroupRatioOverridesRequest struct {
	Enabled   bool               `json:"enabled"`
	Overrides map[string]float64 `json:"overrides"`
}

// UpdateUserGroupRatioOverrides 更新指定用户的分组倍率覆盖配置（分组价格设置）
func UpdateUserGroupRatioOverrides(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req UpdateGroupRatioOverridesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "invalid request body",
		})
		return
	}

	// 校验倍率取值，不允许负数
	for group, ratio := range req.Overrides {
		if ratio < 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": fmt.Sprintf("group %s has negative ratio value", group),
			})
			return
		}
	}

	// 读取当前用户设置（保留其它字段，避免覆盖限流等设置）
	userSetting, err := model.GetUserSetting(id, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 仅更新分组价格设置相关字段
	userSetting.GroupRatioEnabled = req.Enabled
	if len(req.Overrides) == 0 {
		userSetting.GroupRatioOverrides = nil
	} else {
		userSetting.GroupRatioOverrides = req.Overrides
	}

	// 写回用户（同时刷新缓存）
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
