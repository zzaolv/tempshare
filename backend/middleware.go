// backend/middleware.go
package main

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter 存储每个IP地址的速率限制器
type IPRateLimiter struct {
	ips      map[string]*rate.Limiter
	mu       sync.Mutex
	requests int
	duration time.Duration
}

// NewIPRateLimiter 创建一个新的速率限制器实例
func NewIPRateLimiter(r int, d time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		ips:      make(map[string]*rate.Limiter),
		requests: r,
		duration: d,
	}
}

// addIP 创建一个新的速率限制器并添加到IP映射中
func (i *IPRateLimiter) addIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 使用 rate.NewLimiter(每秒事件数, 桶的大小)
	// 我们希望在 'duration' 内允许 'requests' 次请求
	// 所以速率是 requests / duration_in_seconds
	limiter := rate.NewLimiter(rate.Limit(float64(i.requests)/i.duration.Seconds()), i.requests)
	i.ips[ip] = limiter

	// 启动一个goroutine，在持续时间后从map中删除此IP，以防止内存泄漏
	go func() {
		time.Sleep(i.duration)
		i.mu.Lock()
		delete(i.ips, ip)
		i.mu.Unlock()
	}()

	return limiter
}

// getLimiter 从map中获取一个IP的速率限制器，如果不存在则创建一个
func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.ips[ip]
	i.mu.Unlock()

	if !exists {
		// 使用双重检查锁定模式来避免不必要的锁定
		return i.addIP(ip)
	}

	return limiter
}

// RateLimitMiddleware 是 Gin 中间件函数
func (i *IPRateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := i.getLimiter(c.ClientIP())
		if !limiter.Allow() {
			slog.Warn("速率限制触发", "clientIP", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "请求过于频繁，请稍后再试。"})
			return
		}
		c.Next()
	}
}
