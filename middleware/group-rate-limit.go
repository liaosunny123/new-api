package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const (
	groupRpmKeyPrefix         = "groupRpm:"
	groupConcurrencyKeyPrefix = "groupConcurrency:"
)

func GroupRateLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		if userId == 0 {
			c.Next()
			return
		}

		userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
		usingGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
		if usingGroup == "" {
			usingGroup = common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
		}
		if usingGroup == "" {
			usingGroup = userGroup
		}

		userSetting, _ := common.GetContextKeyType[dto.UserSetting](c, constant.ContextKeyUserSetting)

		rpmLimit, concurrencyLimit := setting.ResolveRPMConcurrencyLimit(userGroup, usingGroup, userSetting)

		// No limits configured - skip
		if rpmLimit <= 0 && concurrencyLimit <= 0 {
			c.Next()
			return
		}

		// Check Concurrency (sliding window, success-only)
		var concKey string
		if concurrencyLimit > 0 {
			concKey = fmt.Sprintf("%s%d:%s", groupConcurrencyKeyPrefix, userId, usingGroup)
			if !checkSlidingWindow(concKey, concurrencyLimit) {
				msg := setting.RateLimitExceededMessage
				if msg == "" {
					msg = fmt.Sprintf("当前分组 %s 并发请求超限（并发上限: %d），请稍后再试", usingGroup, concurrencyLimit)
				}
				recordRateLimitLog(c, userId, usingGroup, msg)
				abortWithOpenAiMessage(c, http.StatusServiceUnavailable, msg)
				return
			}
		}

		// Check RPM (sliding window, success-only)
		var rpmKey string
		if rpmLimit > 0 {
			rpmKey = fmt.Sprintf("%s%d:%s", groupRpmKeyPrefix, userId, usingGroup)
			if !checkSlidingWindow(rpmKey, rpmLimit) {
				msg := setting.RateLimitExceededMessage
				if msg == "" {
					msg = fmt.Sprintf("当前分组 %s 请求速率超限（RPM上限: %d），请稍后再试", usingGroup, rpmLimit)
				}
				recordRateLimitLog(c, userId, usingGroup, msg)
				abortWithOpenAiMessage(c, http.StatusServiceUnavailable, msg)
				return
			}
		}

		c.Next()

		// Record only on successful response (status < 400)
		if c.Writer.Status() < 400 {
			if rpmLimit > 0 {
				recordSlidingWindow(rpmKey)
			}
			if concurrencyLimit > 0 {
				recordSlidingWindow(concKey)
			}
		}
	}
}

// ---- Shared sliding window (60s) used by both RPM and concurrency ----

func checkSlidingWindow(key string, limit int) bool {
	if common.RedisEnabled {
		ctx := context.Background()
		length := cleanExpiredEntries(ctx, common.RDB, key)
		return length < int64(limit)
	}
	inMemoryRateLimiter.Init(2 * time.Minute)
	return inMemoryRateLimiter.Check(key, limit, 60)
}

func recordSlidingWindow(key string) {
	if common.RedisEnabled {
		ctx := context.Background()
		now := time.Now().Format(timeFormat)
		common.RDB.LPush(ctx, key, now)
		common.RDB.Expire(ctx, key, 2*time.Minute)
	} else {
		inMemoryRateLimiter.Init(2 * time.Minute)
		// limit=0 here just to record; actual limiting is done in checkSlidingWindow
		inMemoryRateLimiter.Request(key, 1<<30, 60)
	}
}

// Lua script: remove expired entries from list tail and return remaining length.
var cleanAndCountScript = redis.NewScript(`
local key = KEYS[1]
local cutoff = ARGV[1]
while true do
  local val = redis.call('LINDEX', key, -1)
  if not val then break end
  if val < cutoff then
    redis.call('RPOP', key)
  else
    break
  end
end
return redis.call('LLEN', key)
`)

func cleanExpiredEntries(ctx context.Context, rdb *redis.Client, key string) int64 {
	cutoff := time.Now().Add(-60 * time.Second).Format(timeFormat)
	result, err := cleanAndCountScript.Run(ctx, rdb, []string{key}, cutoff).Int64()
	if err != nil {
		return 0
	}
	return result
}

func recordRateLimitLog(c *gin.Context, userId int, group string, content string) {
	tokenName := c.GetString("token_name")
	tokenId := c.GetInt("token_id")
	model.RecordErrorLog(c, userId, 0, "", tokenName, content, tokenId, 0, false, group, map[string]interface{}{
		"status_code": http.StatusServiceUnavailable,
		"error_type":  "group_rate_limit",
	})
}

// GetSlidingWindowCount returns the current count within the 60s window for a key
func GetSlidingWindowCount(key string) int64 {
	if common.RedisEnabled {
		ctx := context.Background()
		return cleanExpiredEntries(ctx, common.RDB, key)
	}
	return 0
}
