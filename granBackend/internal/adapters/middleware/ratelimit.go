package guards

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
)

type RateLimiterConfig struct {
	Limit  int
	Burst  int
	Window time.Duration
}

func RedisRateLimiter(client *redis.Client, config RateLimiterConfig) echo.MiddlewareFunc {
	limiter := redis_rate.NewLimiter(client)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ip := c.RealIP()

			key := fmt.Sprintf("rate_limit:%s", ip)

			res, err := limiter.Allow(c.Request().Context(), key, redis_rate.Limit{
				Rate:   config.Limit,
				Period: config.Window,
				Burst:  config.Burst,
			})

			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error interno de seguridad"})
			}

			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Limit))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
			c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(res.ResetAfter.Milliseconds(), 10))

			if res.Allowed == 0 {
				secondsWait := int(res.ResetAfter.Seconds())
				return c.JSON(http.StatusTooManyRequests, map[string]any{
					"error":       "Has excedido el límite de intentos.",
					"retry_after": secondsWait,
					"message":     fmt.Sprintf("Espera %d segundos antes de intentar de nuevo.", secondsWait),
				})
			}

			return next(c)
		}
	}
}
