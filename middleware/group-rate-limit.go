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

		// Check Concurrency first (does not consume a slot until acquired)
		var concRelease func()
		if concurrencyLimit > 0 {
			concKey := fmt.Sprintf("%s%d:%s", groupConcurrencyKeyPrefix, userId, usingGroup)
			allowed, release := common.GlobalConcurrencyTracker.Acquire(concKey, concurrencyLimit)
			if !allowed {
				msg := setting.RateLimitExceededMessage
				if msg == "" {
					msg = fmt.Sprintf("当前分组 %s 并发请求超限（并发上限: %d），请稍后再试", usingGroup, concurrencyLimit)
				}
				recordRateLimitLog(c, userId, usingGroup, msg)
				abortWithOpenAiMessage(c, http.StatusServiceUnavailable, msg)
				return
			}
			concRelease = release
			defer concRelease()
		}

		// Check RPM (check only, do not record yet)
		var rpmKey string
		if rpmLimit > 0 {
			rpmKey = fmt.Sprintf("%s%d:%s", groupRpmKeyPrefix, userId, usingGroup)
			if !checkGroupRpm(rpmKey, rpmLimit) {
				msg := setting.RateLimitExceededMessage
				if msg == "" {
					msg = fmt.Sprintf("当前分组 %s 请求速率超限（RPM上限: %d），请稍后再试", usingGroup, rpmLimit)
				}
				// Release concurrency slot before aborting
				if concRelease != nil {
					concRelease()
				}
				recordRateLimitLog(c, userId, usingGroup, msg)
				abortWithOpenAiMessage(c, http.StatusServiceUnavailable, msg)
				return
			}
		}

		c.Next()

		// Record RPM only on successful response (status < 400)
		if rpmLimit > 0 && c.Writer.Status() < 400 {
			recordGroupRpm(rpmKey, rpmLimit)
		}
	}
}

func checkGroupRpm(key string, limit int) bool {
	if common.RedisEnabled {
		return checkGroupRpmRedis(key, limit)
	}
	return checkGroupRpmMemory(key, limit)
}

func checkGroupRpmRedis(key string, limit int) bool {
	ctx := context.Background()
	length := cleanExpiredEntries(ctx, common.RDB, key)
	return length < int64(limit)
}

// Lua script: remove expired entries from list tail and return remaining length.
// Single round-trip to Redis regardless of how many entries are expired.
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

// cleanExpiredEntries removes entries older than 60 seconds from the tail
// and returns the remaining list length. Single Redis round-trip via Lua.
func cleanExpiredEntries(ctx context.Context, rdb *redis.Client, key string) int64 {
	cutoff := time.Now().Add(-60 * time.Second).Format(timeFormat)
	result, err := cleanAndCountScript.Run(ctx, rdb, []string{key}, cutoff).Int64()
	if err != nil {
		return 0
	}
	return result
}

func checkGroupRpmMemory(key string, limit int) bool {
	inMemoryRateLimiter.Init(2 * time.Minute)
	return inMemoryRateLimiter.Check(key, limit, 60)
}

func recordGroupRpm(key string, limit int) {
	if common.RedisEnabled {
		recordGroupRpmRedis(key, limit)
	} else {
		inMemoryRateLimiter.Init(2 * time.Minute)
		inMemoryRateLimiter.Request(key, limit, 60)
	}
}

func recordGroupRpmRedis(key string, limit int) {
	ctx := context.Background()
	rdb := common.RDB
	now := time.Now().Format(timeFormat)
	rdb.LPush(ctx, key, now)
	rdb.Expire(ctx, key, 2*time.Minute)
}

func recordRateLimitLog(c *gin.Context, userId int, group string, content string) {
	tokenName := c.GetString("token_name")
	tokenId := c.GetInt("token_id")
	model.RecordErrorLog(c, userId, 0, "", tokenName, content, tokenId, 0, false, group, map[string]interface{}{
		"status_code": http.StatusServiceUnavailable,
		"error_type":  "group_rate_limit",
	})
}

// GetGroupRpmCount returns current RPM count for the given key
func GetGroupRpmCount(key string) int64 {
	if common.RedisEnabled {
		ctx := context.Background()
		return cleanExpiredEntries(ctx, common.RDB, key)
	}
	return 0
}
