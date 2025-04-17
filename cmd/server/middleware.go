package main

import (
	"time"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		logger := log.With().
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Str("user-agent", c.Request.UserAgent()).
			Logger()

		if len(c.Errors) > 0 {
			logger.Error().Msg(c.Errors.String())
		} else {
			logger.Info().Msg("Request processed")
		}
	}
}
